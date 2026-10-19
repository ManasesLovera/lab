package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// Availability is tracked as a transition ledger: a record is written only when
// a container changes state (or is seen for the first time), not on every poll.
// A steady app therefore costs one line per deploy instead of one per sample.

const (
	// uptimeRetention keeps a little more than the longest window we report.
	uptimeRetention = 35 * 24 * time.Hour
	// uptimeMinSpan avoids reporting a percentage over a negligible period.
	uptimeMinSpan = time.Minute
)

type utTransition struct {
	T  time.Time `json:"t"`
	Up bool      `json:"up"`
}

// uptimeRecord is one line in the on-disk ledger.
type uptimeRecord struct {
	Name  string    `json:"name"`
	T     time.Time `json:"t"`
	Up    bool      `json:"up"`
	First bool      `json:"first,omitempty"`
}

type appUptime struct {
	Name      string
	FirstSeen time.Time
	LastUp    bool
	Trans     []utTransition
}

// UptimeInfo is the availability summary attached to every app in /api/health.
type UptimeInfo struct {
	Day        *float64 `json:"24h,omitempty"`
	Week       *float64 `json:"7d,omitempty"`
	Month      *float64 `json:"30d,omitempty"`
	Restarts   int      `json:"restarts24h"`
	LastChange string   `json:"lastChange,omitempty"`
}

type uptimeSegment struct {
	Start int64 `json:"start"`
	End   int64 `json:"end"`
	Up    bool  `json:"up"`
}

type appUptimeSeries struct {
	Name     string          `json:"name"`
	Segments []uptimeSegment `json:"segments"`
}

type uptimeResponse struct {
	From int64             `json:"from"`
	To   int64             `json:"to"`
	Apps []appUptimeSeries `json:"apps"`
}

type uptimeLedger struct {
	mu          sync.Mutex
	file        string
	apps        map[string]*appUptime
	pending     []uptimeRecord
	lastCompact time.Time
}

func newUptimeLedger(dir string) *uptimeLedger {
	l := &uptimeLedger{apps: map[string]*appUptime{}}
	if dir == "" {
		return l
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		log.Printf("uptime: cannot create %s: %v (history will not persist)", dir, err)
		return l
	}
	l.file = filepath.Join(dir, "uptime.jsonl")
	l.load()
	l.compact(time.Now())
	return l
}

// run polls Docker until stop is closed. A nil stop channel blocks forever.
func (l *uptimeLedger) run(interval time.Duration, stop <-chan struct{}) {
	if interval <= 0 {
		interval = 30 * time.Second
	}
	l.poll(time.Now())
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			l.flush()
			return
		case now := <-ticker.C:
			l.poll(now)
		}
	}
}

// poll records the current up/down state of every container.
func (l *uptimeLedger) poll(now time.Time) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "docker", "ps", "-a",
		"--format", "{{.Names}}\t{{.State}}\t{{.Status}}").Output()
	if err != nil {
		log.Printf("uptime: docker ps: %v", err)
		return
	}

	observed := map[string]bool{}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		f := strings.Split(line, "\t")
		if len(f) < 3 || f[0] == "" {
			continue
		}
		// Running and not failing its healthcheck counts as up. `starting`
		// is up too: the container is running, only the probe is warming up.
		observed[f[0]] = f[1] == "running" && !strings.Contains(f[2], "(unhealthy)")
	}

	l.mu.Lock()
	for name, up := range observed {
		l.observeLocked(name, up, now)
	}
	// A container that vanished from `docker ps -a` stopped being available.
	for name, h := range l.apps {
		if _, ok := observed[name]; !ok && h.LastUp {
			l.observeLocked(name, false, now)
		}
	}
	needCompact := l.file != "" && (l.lastCompact.IsZero() || now.Sub(l.lastCompact) > 24*time.Hour)
	l.mu.Unlock()

	l.flush()
	if needCompact {
		l.compact(now)
	}
}

func (l *uptimeLedger) observeLocked(name string, up bool, now time.Time) {
	h, ok := l.apps[name]
	if !ok {
		h = &appUptime{Name: name, FirstSeen: now, LastUp: up, Trans: []utTransition{{now, up}}}
		l.apps[name] = h
		l.pending = append(l.pending, uptimeRecord{Name: name, T: now, Up: up, First: true})
		return
	}
	if h.LastUp == up {
		return
	}
	h.Trans = append(h.Trans, utTransition{now, up})
	h.LastUp = up
	l.pending = append(l.pending, uptimeRecord{Name: name, T: now, Up: up})
}

// applyRecord replays one persisted record while loading.
func (l *uptimeLedger) applyRecord(r uptimeRecord) {
	h, ok := l.apps[r.Name]
	if !ok {
		l.apps[r.Name] = &appUptime{
			Name:      r.Name,
			FirstSeen: r.T,
			LastUp:    r.Up,
			Trans:     []utTransition{{r.T, r.Up}},
		}
		return
	}
	if r.T.Before(h.FirstSeen) {
		h.FirstSeen = r.T
	}
	if r.Up != h.LastUp {
		h.Trans = append(h.Trans, utTransition{r.T, r.Up})
	}
	h.LastUp = r.Up
}

func (l *uptimeLedger) load() {
	data, err := os.ReadFile(l.file)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("uptime: cannot read %s: %v", l.file, err)
		}
		return
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var r uptimeRecord
		if err := json.Unmarshal([]byte(line), &r); err != nil {
			log.Printf("uptime: skipping corrupt record: %v", err)
			continue
		}
		if r.Name == "" || r.T.IsZero() {
			continue
		}
		l.applyRecord(r)
	}
}

// flush appends pending transitions to the ledger file.
func (l *uptimeLedger) flush() {
	l.mu.Lock()
	recs := l.pending
	l.pending = nil
	l.mu.Unlock()
	if l.file == "" || len(recs) == 0 {
		return
	}

	f, err := os.OpenFile(l.file, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		log.Printf("uptime: cannot open %s: %v", l.file, err)
		return
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	for _, r := range recs {
		if err := enc.Encode(r); err != nil {
			log.Printf("uptime: cannot write %s: %v", l.file, err)
			return
		}
	}
}

// compact prunes old transitions and rewrites the file so it cannot grow
// without bound. The state at the retention cutoff is kept so percentages for
// every reported window remain correct.
func (l *uptimeLedger) compact(now time.Time) {
	cutoff := now.Add(-uptimeRetention)

	l.mu.Lock()
	var recs []uptimeRecord
	for _, h := range l.apps {
		h.prune(cutoff)
		if len(h.Trans) == 0 {
			continue
		}
		recs = append(recs, uptimeRecord{
			Name:  h.Name,
			T:     h.FirstSeen,
			Up:    h.Trans[0].Up,
			First: true,
		})
		for _, tr := range h.Trans {
			if tr.T.Equal(h.FirstSeen) {
				continue
			}
			recs = append(recs, uptimeRecord{Name: h.Name, T: tr.T, Up: tr.Up})
		}
	}
	l.lastCompact = now
	l.mu.Unlock()

	if l.file == "" {
		return
	}
	tmp := l.file + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		log.Printf("uptime: cannot write %s: %v", tmp, err)
		return
	}
	enc := json.NewEncoder(f)
	for _, r := range recs {
		if err := enc.Encode(r); err != nil {
			f.Close()
			log.Printf("uptime: cannot write %s: %v", tmp, err)
			return
		}
	}
	if err := f.Close(); err != nil {
		log.Printf("uptime: cannot close %s: %v", tmp, err)
		return
	}
	if err := os.Rename(tmp, l.file); err != nil {
		log.Printf("uptime: cannot replace %s: %v", l.file, err)
	}
}

func (h *appUptime) prune(cutoff time.Time) {
	if len(h.Trans) == 0 {
		return
	}
	i := sort.Search(len(h.Trans), func(i int) bool { return h.Trans[i].T.After(cutoff) })
	if i > 0 {
		h.Trans = h.Trans[i-1:]
	}
}

// summary builds the availability figures shown in the App Health tab.
func (l *uptimeLedger) summary(name string, now time.Time) UptimeInfo {
	l.mu.Lock()
	defer l.mu.Unlock()
	h := l.apps[name]
	if h == nil {
		return UptimeInfo{}
	}

	info := UptimeInfo{
		Day:      h.percentage(now.Add(-24*time.Hour), now),
		Week:     h.percentage(now.Add(-7*24*time.Hour), now),
		Month:    h.percentage(now.Add(-30*24*time.Hour), now),
		Restarts: h.restarts(now.Add(-24*time.Hour), now),
	}
	if len(h.Trans) > 1 {
		info.LastChange = h.Trans[len(h.Trans)-1].T.Format(time.RFC3339)
	}
	return info
}

// percentage returns availability for [from, now], or nil when the container
// has not existed for the whole window (a partial window would be misleading).
func (h *appUptime) percentage(from, now time.Time) *float64 {
	if len(h.Trans) == 0 || h.FirstSeen.After(from) {
		return nil
	}
	span := now.Sub(from)
	if span < uptimeMinSpan {
		return nil
	}

	var up time.Duration
	for i, tr := range h.Trans {
		start := tr.T
		if start.Before(from) {
			start = from
		}
		end := now
		if i+1 < len(h.Trans) {
			end = h.Trans[i+1].T
		}
		if end.After(now) {
			end = now
		}
		if end.After(start) && tr.Up {
			up += end.Sub(start)
		}
	}
	pct := 100 * float64(up) / float64(span)
	return &pct
}

func (h *appUptime) restarts(from, now time.Time) int {
	n := 0
	for i := 1; i < len(h.Trans); i++ {
		tr := h.Trans[i]
		if tr.Up && !h.Trans[i-1].Up && !tr.T.Before(from) && !tr.T.After(now) {
			n++
		}
	}
	return n
}

// series returns up/down segments for the given window, for the timeline bars.
func (l *uptimeLedger) series(window time.Duration, now time.Time) uptimeResponse {
	from := now.Add(-window)
	resp := uptimeResponse{From: from.Unix(), To: now.Unix(), Apps: []appUptimeSeries{}}

	l.mu.Lock()
	defer l.mu.Unlock()
	for name, h := range l.apps {
		segs := []uptimeSegment{}
		for i, tr := range h.Trans {
			start := tr.T
			if start.Before(from) {
				start = from
			}
			end := now
			if i+1 < len(h.Trans) {
				end = h.Trans[i+1].T
			}
			if end.After(now) {
				end = now
			}
			if end.After(start) {
				segs = append(segs, uptimeSegment{Start: start.Unix(), End: end.Unix(), Up: tr.Up})
			}
		}
		if len(segs) > 0 {
			resp.Apps = append(resp.Apps, appUptimeSeries{Name: name, Segments: segs})
		}
	}
	sort.Slice(resp.Apps, func(i, j int) bool { return resp.Apps[i].Name < resp.Apps[j].Name })
	return resp
}
