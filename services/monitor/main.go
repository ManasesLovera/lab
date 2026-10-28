package main

import (
	"context"
	"crypto/subtle"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/process"
)

//go:embed templates/index.html
var templateFS embed.FS

// Host paths allow the dashboard to run in a container while reporting the
// host's metrics. HOST_ROOT points at the host root mount (e.g. /host),
// HOST_PROC/HOST_SYS at the host pseudo-filesystems. They default to "/" so
// the binary still works when executed directly on the host.
var (
	hostRoot string
	procRoot string
	sysRoot  string
)

func init() {
	hostRoot = os.Getenv("HOST_ROOT")
	procRoot = os.Getenv("HOST_PROC")
	if procRoot == "" {
		procRoot = "/proc"
	}
	sysRoot = os.Getenv("HOST_SYS")
	if sysRoot == "" {
		sysRoot = "/sys"
	}
}

func hostPath(p string) string { return filepath.Join(hostRoot, p) }
func procPath(p string) string { return filepath.Join(procRoot, p) }
func sysPath(p string) string  { return filepath.Join(sysRoot, p) }

type HardwareStats struct {
	Board        string `json:"board"`
	Manufacturer string `json:"manufacturer"`
	Platform     string `json:"platform"`
	Kernel       string `json:"kernel"`
	Arch         string `json:"arch"`
	Serial       string `json:"serial"`
	Revision     string `json:"revision"`
	GPU          string `json:"gpu"`
	GPUMMAL      string `json:"gpu_mmal"`
	Firmware     string `json:"firmware"`
	USB          string `json:"usb"`
	Ethernet     string `json:"ethernet"`
}

type CPUStats struct {
	Percent float64 `json:"percent"`
	Cores   int     `json:"cores"`
	Model   string  `json:"model"`
}

type RAMStats struct {
	Total     uint64  `json:"total"`
	Used      uint64  `json:"used"`
	Available uint64  `json:"available"`
	Percent   float64 `json:"percent"`
}

type DiskStats struct {
	Total     uint64       `json:"total"`
	Used      uint64       `json:"used"`
	Available uint64       `json:"available"`
	Percent   float64      `json:"percent"`
	Folders   []FolderInfo `json:"folders"`
	Docker    DockerStats  `json:"docker"`
}

type FolderInfo struct {
	Path string       `json:"path"`
	Size uint64       `json:"size"`
	Subs []FolderInfo `json:"subs,omitempty"`
}

type DockerStats struct {
	Images     uint64 `json:"images"`
	Containers uint64 `json:"containers"`
	Volumes    uint64 `json:"volumes"`
	Total      uint64 `json:"total"`
}

type NetStats struct {
	SSID string `json:"ssid"`
	IP   string `json:"ip"`
}

type ProcInfo struct {
	PID  int32   `json:"pid"`
	Name string  `json:"name"`
	CPU  float64 `json:"cpu"`
	RAM  float32 `json:"ram"`
}

// Each function below backs exactly one API endpoint, so the frontend can
// request (and render) every section independently.

type OverviewStats struct {
	Hostname string   `json:"hostname"`
	Uptime   string   `json:"uptime"`
	Temp     float64  `json:"temp"`
	Network  NetStats `json:"network"`
}

func getOverview() OverviewStats {
	o := OverviewStats{}
	// Hostname (prefer the host's /etc/hostname when running in a container).
	if h, err := host.Info(); err == nil {
		o.Hostname = h.Hostname
		uptime := time.Duration(h.Uptime) * time.Second
		o.Uptime = fmt.Sprintf("%dh %dm", int(uptime.Hours()), int(uptime.Minutes())%60)
	}
	if name, err := os.ReadFile(hostPath("etc/hostname")); err == nil {
		if s := strings.TrimSpace(string(name)); s != "" {
			o.Hostname = s
		}
	}
	if temp, err := getTemperature(); err == nil {
		o.Temp = temp
	}
	o.Network = getNetworkInfo()
	return o
}

func getCPU() CPUStats {
	c := CPUStats{}
	if p, err := cpu.Percent(500*time.Millisecond, false); err == nil && len(p) > 0 {
		c.Percent = p[0]
	}
	if info, err := cpu.Info(); err == nil && len(info) > 0 {
		c.Cores = len(info)
		c.Model = info[0].ModelName
		if c.Model == "" {
			c.Model = fmt.Sprintf("%s %s", info[0].VendorID, info[0].Family)
		}
	}
	return c
}

func getRAM() RAMStats {
	r := RAMStats{}
	if vm, err := mem.VirtualMemory(); err == nil {
		r.Total, r.Used, r.Available, r.Percent = vm.Total, vm.Used, vm.Available, vm.UsedPercent
	}
	return r
}

func getDiskUsage() DiskStats {
	d := DiskStats{}
	if usage, err := disk.Usage(hostPath("/")); err == nil {
		d.Total, d.Used, d.Available, d.Percent = usage.Total, usage.Used, usage.Free, usage.UsedPercent
	}
	return d
}

// AppHealth is the state of one container (an "app" in the lab).
type AppHealth struct {
	Name    string     `json:"name"`
	Image   string     `json:"image"`
	State   string     `json:"state"`         // docker state: running, exited, restarting...
	Health  string     `json:"health"`        // healthy, unhealthy, starting, none
	Status  string     `json:"status"`        // docker's human status ("Up 2 days (healthy)")
	URL     string     `json:"url,omitempty"` // public link, when the proxy exposes this app
	Level   string     `json:"level"`         // ok, warn, down
	Summary string     `json:"summary"`       // short reason shown in the UI
	Uptime  UptimeInfo `json:"uptime"`        // availability history (transition ledger)
}

type HealthReport struct {
	Apps    []AppHealth `json:"apps"`
	OK      int         `json:"ok"`
	Warn    int         `json:"warn"`
	Down    int         `json:"down"`
	Error   string      `json:"error,omitempty"`
	Checked string      `json:"checked"`
}

// publicHosts returns the public hostnames declared in the lab proxy config
// (server_name entries), excluding the catch-all and the *.local LAN names.
// Read per request so new services show up without restarting the monitor.
func publicHosts() map[string]bool {
	path := os.Getenv("PROXY_CONF")
	if path == "" {
		path = hostPath("/home/mlovera/lab/core/proxy/conf.d/proxy.conf")
	}
	hosts := map[string]bool{}
	data, err := os.ReadFile(path)
	if err != nil {
		return hosts
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "server_name") {
			continue
		}
		for _, h := range strings.Fields(strings.TrimSuffix(strings.TrimPrefix(line, "server_name"), ";")) {
			if strings.Contains(h, ".") && !strings.HasSuffix(h, ".local") {
				hosts[h] = true
			}
		}
	}
	return hosts
}

func getHealth() HealthReport {
	now := time.Now()
	rep := HealthReport{Apps: []AppHealth{}, Checked: now.Format(time.RFC3339)}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "docker", "ps", "-a",
		"--format", "{{.Names}}\t{{.Image}}\t{{.State}}\t{{.Status}}").Output()
	if err != nil {
		rep.Error = "cannot query docker: " + err.Error()
		return rep
	}

	hosts := publicHosts()
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		f := strings.Split(line, "\t")
		if len(f) < 4 {
			continue
		}
		a := AppHealth{Name: f[0], Image: f[1], State: f[2], Status: f[3], Health: "none"}
		switch {
		case strings.Contains(a.Status, "(unhealthy)"):
			a.Health = "unhealthy"
		case strings.Contains(a.Status, "(healthy)"):
			a.Health = "healthy"
		case strings.Contains(a.Status, "health: starting"):
			a.Health = "starting"
		}

		switch {
		case a.State != "running" && a.State != "restarting":
			a.Level, a.Summary = "down", "container "+a.State
		case a.State == "restarting":
			a.Level, a.Summary = "warn", "restarting"
		case a.Health == "unhealthy":
			a.Level, a.Summary = "down", "healthcheck failing"
		case a.Health == "starting":
			a.Level, a.Summary = "warn", "starting up"
		case a.Health == "healthy":
			a.Level, a.Summary = "ok", "healthy"
		default:
			a.Level, a.Summary = "ok", "running (no healthcheck)"
		}

		// Link convention: container "x" is served at x.<public domain>.
		for h := range hosts {
			if strings.HasPrefix(h, a.Name+".") && !strings.Contains(strings.TrimPrefix(h, a.Name+"."), ".rpi") {
				a.URL = "https://" + h
				break
			}
		}

		if uptimeHistory != nil {
			a.Uptime = uptimeHistory.summary(a.Name, now)
		}

		switch a.Level {
		case "ok":
			rep.OK++
		case "warn":
			rep.Warn++
		default:
			rep.Down++
		}
		rep.Apps = append(rep.Apps, a)
	}

	// Problems first, then by name.
	rank := map[string]int{"down": 0, "warn": 1, "ok": 2}
	sort.Slice(rep.Apps, func(i, j int) bool {
		if rank[rep.Apps[i].Level] != rank[rep.Apps[j].Level] {
			return rank[rep.Apps[i].Level] < rank[rep.Apps[j].Level]
		}
		return rep.Apps[i].Name < rep.Apps[j].Name
	})
	return rep
}

// vcgencmd runs the host's Raspberry Pi firmware tool. Inside the container
// the host binary is visible under HOST_ROOT, but it is dynamically linked
// against the host's glibc, so it must be invoked through the host loader.
func vcgencmd(args ...string) (string, error) {
	if out, err := exec.Command("vcgencmd", args...).Output(); err == nil {
		return string(out), nil
	}

	loaders := []string{
		"lib/ld-linux-aarch64.so.1",
		"lib/ld-linux-armhf.so.3",
		"lib64/ld-linux-x86-64.so.2",
	}
	libDirs := []string{
		"usr/lib/aarch64-linux-gnu",
		"usr/lib/arm-linux-gnueabihf",
		"usr/lib/x86_64-linux-gnu",
	}
	bin := hostPath("usr/bin/vcgencmd")
	for _, loaderRel := range loaders {
		loader := hostPath(loaderRel)
		if _, err := os.Stat(loader); err != nil {
			continue
		}
		for _, libRel := range libDirs {
			libDir := hostPath(libRel)
			if _, err := os.Stat(libDir); err != nil {
				continue
			}
			argv := append([]string{"--library-path", libDir, bin}, args...)
			out, err := exec.Command(loader, argv...).Output()
			if err == nil {
				return string(out), nil
			}
		}
	}
	return "", exec.ErrNotFound
}

func getTemperature() (float64, error) {
	// Preferred: Raspberry Pi firmware tool (only present on the host).
	if out, err := vcgencmd("measure_temp"); err == nil {
		// Parse "temp=48.8'C"
		str := string(out)
		start := strings.Index(str, "=")
		end := strings.Index(str, "'")
		if start != -1 && end != -1 {
			if temp, err := strconv.ParseFloat(str[start+1:end], 64); err == nil {
				return temp, nil
			}
		}
	}

	// Fallback: thermal zone (works inside a container with the host /sys mounted).
	data, err := os.ReadFile(sysPath("class/thermal/thermal_zone0/temp"))
	if err != nil {
		return 0, err
	}
	milli, err := strconv.ParseFloat(strings.TrimSpace(string(data)), 64)
	if err != nil {
		return 0, err
	}
	return milli / 1000, nil
}

func getHardwareInfo() HardwareStats {
	hw := HardwareStats{}

	// Board model (sysfs path works in containers; /proc/device-tree is a symlink).
	if model, err := os.ReadFile(sysPath("firmware/devicetree/base/model")); err == nil {
		hw.Board = strings.TrimRight(string(model), "\x00\n ")
	} else if model, err := os.ReadFile(procPath("device-tree/model")); err == nil {
		hw.Board = strings.TrimRight(string(model), "\x00\n ")
	}
	if hw.Board == "" {
		hw.Board = "Unknown"
	}

	// Manufacturer from host info
	if h, err := host.Info(); err == nil {
		hw.Manufacturer = h.PlatformFamily
		if hw.Manufacturer == "" {
			hw.Manufacturer = "ARM"
		}
		hw.Platform = h.Platform + " " + h.PlatformVersion
		hw.Kernel = h.KernelVersion
		hw.Arch = h.KernelArch
	}

	// Serial
	if serial, err := os.ReadFile(procPath("cpuinfo")); err == nil {
		for _, line := range strings.Split(string(serial), "\n") {
			if strings.HasPrefix(line, "Serial") {
				parts := strings.SplitN(line, ":", 2)
				if len(parts) == 2 {
					hw.Serial = strings.TrimSpace(parts[1])
				}
			}
			if strings.HasPrefix(line, "Revision") {
				parts := strings.SplitN(line, ":", 2)
				if len(parts) == 2 {
					hw.Revision = strings.TrimSpace(parts[1])
				}
			}
		}
	}

	// GPU memory
	if gpu, err := vcgencmd("get_mem", "gpu"); err == nil {
		hw.GPU = strings.TrimSpace(gpu)
		// Parse "gpu=76M"
		if idx := strings.Index(hw.GPU, "="); idx >= 0 {
			hw.GPU = hw.GPU[idx+1:]
		}
	}

	// Firmware type/program. On Pi 4/5 the firmware is the bootloader stored
	// in the SPI EEPROM; there is no separate start.elf anymore.
	if name, err := os.ReadFile(sysPath("firmware/devicetree/base/chosen/bootloader/name")); err == nil && strings.TrimRight(string(name), "\x00\n ") != "" {
		hw.Firmware = "Bootloader EEPROM"
		if compat, err := os.ReadFile(sysPath("firmware/devicetree/base/compatible")); err == nil {
			for _, token := range strings.Split(string(compat), "\x00") {
				if strings.HasPrefix(token, "brcm,bcm") {
					hw.Firmware = "Bootloader EEPROM (bootloader-" + strings.TrimPrefix(token, "brcm,bcm") + ")"
					break
				}
			}
		}
	} else if fw, err := vcgencmd("version"); err == nil {
		// Older Pi: VideoCore firmware loaded from start.elf on the boot partition.
		hw.Firmware = "VideoCore firmware"
		for _, line := range strings.Split(fw, "\n") {
			if strings.HasPrefix(line, "version ") {
				hw.Firmware += " " + strings.TrimSpace(strings.TrimPrefix(line, "version "))
				break
			}
		}
	}

	// USB - check if usb devices exist
	if usb, err := exec.Command("lsusb").Output(); err == nil {
		lines := strings.Split(strings.TrimSpace(string(usb)), "\n")
		hw.USB = fmt.Sprintf("%d devices", len(lines))
	} else {
		hw.USB = "N/A"
	}

	// Ethernet MAC
	if mac, err := os.ReadFile(sysPath("class/net/eth0/address")); err == nil {
		hw.Ethernet = strings.TrimSpace(string(mac))
	}

	return hw
}

// nestedHomeFolders lists user folders that get a second-level breakdown,
// like the "dev" folder does.
var nestedHomeFolders = map[string]bool{
	"dev": true, ".nvm": true, "lab": true, "runners": true,
	".local": true, ".opencode": true,
}

// nestedRootFolders lists sub-folders of a top-level path that get a
// second-level breakdown.
var nestedRootFolders = map[string]map[string]bool{
	"/var": {"lib": true},
	"/usr": {"lib": true, "bin": true},
}

func getDiskFolders() []FolderInfo {
	folders := []string{"/home", "/usr", "/var", "/snap", "/boot", "/tmp", "/opt"}

	var result []FolderInfo
	for _, path := range folders {
		full := hostPath(path)
		size := getDirSize(full)
		if size > 0 {
			label := strings.TrimPrefix(path, "/")
			folder := FolderInfo{
				Path: label,
				Size: size,
			}
			// Get subdirectories for large folders (>500MB)
			if size > 500*1024*1024 {
				// Special handling for /home - go into user directories
				if path == "/home" {
					folder.Subs = getHomeSubFolders(full)
				} else {
					folder.Subs = getSubFolders(full)
					if names, ok := nestedRootFolders[path]; ok {
						for i := range folder.Subs {
							if names[folder.Subs[i].Path] {
								folder.Subs[i].Subs = getSubFolders(filepath.Join(full, folder.Subs[i].Path))
							}
						}
					}
				}
			}
			result = append(result, folder)
		}
	}

	// Sort by size descending
	sort.Slice(result, func(i, j int) bool {
		return result[i].Size > result[j].Size
	})

	return result
}

func getHomeSubFolders(path string) []FolderInfo {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil
	}

	var subs []FolderInfo
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		dir := filepath.Join(path, entry.Name())
		userSubs := getSubFolders(dir)
		for i := range userSubs {
			if nestedHomeFolders[userSubs[i].Path] {
				userSubs[i].Subs = getSubFolders(filepath.Join(dir, userSubs[i].Path))
			}
		}
		subs = append(subs, userSubs...)
	}

	// Sort by size descending
	sort.Slice(subs, func(i, j int) bool {
		return subs[i].Size > subs[j].Size
	})

	// Limit to top 8
	if len(subs) > 8 {
		subs = subs[:8]
	}

	return subs
}

func getSubFolders(path string) []FolderInfo {
	cmd := exec.Command("find", path, "-maxdepth", "1", "-mindepth", "1", "-type", "d")
	out, err := cmd.Output()
	if err != nil {
		return nil
	}

	var subs []FolderInfo
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")

	// Collect all directories with their sizes
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		size := getDirSize(line)
		if size > 10*1024*1024 { // Only show >10MB
			label := strings.TrimPrefix(line, path+"/")
			subs = append(subs, FolderInfo{
				Path: label,
				Size: size,
			})
		}
	}

	// Sort by size descending
	sort.Slice(subs, func(i, j int) bool {
		return subs[i].Size > subs[j].Size
	})

	// Limit to top 8
	if len(subs) > 8 {
		subs = subs[:8]
	}

	return subs
}

func getDirSize(path string) uint64 {
	// Use timeout to avoid hanging on slow directories
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// -x keeps du on one filesystem, so it does not descend into Docker
	// overlay mounts (which would recurse through the mounted host root).
	cmd := exec.CommandContext(ctx, "du", "-sbx", path)
	// du returns exit 1 on permission errors but still outputs valid data
	out, _ := cmd.Output()
	if len(out) == 0 {
		return 0
	}
	// Parse output: "12345678\t/path"
	parts := strings.Fields(string(out))
	if len(parts) < 1 {
		return 0
	}
	size, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil {
		return 0
	}
	return size
}

func getDockerStats() DockerStats {
	stats := DockerStats{}

	cmd := exec.Command("docker", "system", "df")
	out, err := cmd.Output()
	if err != nil {
		return stats
	}

	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}

		// Parse size strings like "3.619GB", "56.26MB", "1.657GB"
		// Format: TYPE TOTAL ACTIVE SIZE RECLAIMABLE
		// Note: "Local Volumes" takes two fields, so SIZE shifts
		var sizeStr string
		var typeField string

		if fields[0] == "Local" && fields[1] == "Volumes" {
			typeField = "Local Volumes"
			sizeStr = fields[4]
		} else {
			typeField = fields[0]
			sizeStr = fields[3]
		}

		size := parseSize(sizeStr)

		switch typeField {
		case "Images":
			stats.Images = size
		case "Containers":
			stats.Containers = size
		case "Local Volumes":
			stats.Volumes = size
		}
	}

	stats.Total = stats.Images + stats.Containers + stats.Volumes
	return stats
}

func parseSize(s string) uint64 {
	s = strings.TrimSpace(s)
	if s == "0B" || s == "0" {
		return 0
	}

	// Extract number and unit
	var numStr string
	var unit string
	for i, c := range s {
		if c >= '0' && c <= '9' || c == '.' {
			numStr += string(c)
		} else {
			unit = s[i:]
			break
		}
	}

	num, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return 0
	}

	switch strings.ToUpper(unit) {
	case "B":
		return uint64(num)
	case "KB", "KIB":
		return uint64(num * 1024)
	case "MB", "MIB":
		return uint64(num * 1024 * 1024)
	case "GB", "GIB":
		return uint64(num * 1024 * 1024 * 1024)
	case "TB", "TIB":
		return uint64(num * 1024 * 1024 * 1024 * 1024)
	default:
		return uint64(num)
	}
}

func getNetworkInfo() NetStats {
	net := NetStats{}

	// Get SSID
	args := []string{"-i", "wlan0", "status"}
	if hostRoot != "" {
		// In a container the host daemon must be able to reach our client socket,
		// so create it inside the shared wpa_supplicant directory.
		args = append([]string{"-s", "/var/run/wpa_supplicant"}, args...)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "wpa_cli", args...)
	if out, err := cmd.Output(); err == nil {
		for _, line := range strings.Split(string(out), "\n") {
			if strings.HasPrefix(line, "ssid=") {
				net.SSID = strings.TrimPrefix(line, "ssid=")
			}
			if strings.HasPrefix(line, "ip_address=") {
				net.IP = strings.TrimPrefix(line, "ip_address=")
			}
		}
	}

	return net
}

func getTopProcesses(limit int) []ProcInfo {
	pids, err := process.Pids()
	if err != nil {
		return nil
	}

	var procs []ProcInfo
	for _, pid := range pids {
		p, err := process.NewProcess(pid)
		if err != nil {
			continue
		}
		name, _ := p.Name()
		cpuPercent, _ := p.CPUPercent()
		memPercent, _ := p.MemoryPercent()

		if cpuPercent > 0 || memPercent > 0.1 {
			procs = append(procs, ProcInfo{
				PID:  pid,
				Name: name,
				CPU:  cpuPercent,
				RAM:  memPercent,
			})
		}
	}

	// Sort by CPU + RAM usage
	sort.Slice(procs, func(i, j int) bool {
		return (procs[i].CPU + float64(procs[i].RAM)) > (procs[j].CPU + float64(procs[j].RAM))
	})

	if len(procs) > limit {
		procs = procs[:limit]
	}
	return procs
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFS(templateFS, "templates/index.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, nil)
}

// ttlCache memoizes an expensive computation. Concurrent callers share one
// computation (the mutex is held while computing), so a slow `du` scan never
// runs twice at once no matter how many tabs are open.
type ttlCache struct {
	mu  sync.Mutex
	at  time.Time
	val any
}

func (c *ttlCache) get(ttl time.Duration, fn func() any) any {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.val == nil || time.Since(c.at) > ttl {
		c.val, c.at = fn(), time.Now()
	}
	return c.val
}

var (
	folderCache   ttlCache
	dockerCache   ttlCache
	uptimeHistory *uptimeLedger
)

// uptimeInterval reads UPTIME_INTERVAL (Go duration, e.g. "30s").
func uptimeInterval() time.Duration {
	if s := os.Getenv("UPTIME_INTERVAL"); s != "" {
		if d, err := time.ParseDuration(s); err == nil && d > 0 {
			return d
		}
		log.Printf("uptime: invalid UPTIME_INTERVAL %q, using 30s", os.Getenv("UPTIME_INTERVAL"))
	}
	return 30 * time.Second
}

// jsonEndpoint wraps a data function as an authenticated JSON endpoint.
func jsonEndpoint(fn func() any) http.HandlerFunc {
	return requireAuth(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		json.NewEncoder(w).Encode(fn())
	})
}

// requireAuth enforces HTTP Basic authentication, comparing credentials in
// constant time. Credentials come from the environment so they never live in
// source control.
func requireAuth(next http.HandlerFunc) http.HandlerFunc {
	user := os.Getenv("MONITOR_USER")
	pass := os.Getenv("MONITOR_PASS")

	return func(w http.ResponseWriter, r *http.Request) {
		u, p, ok := r.BasicAuth()
		userOK := subtle.ConstantTimeCompare([]byte(u), []byte(user)) == 1
		passOK := subtle.ConstantTimeCompare([]byte(p), []byte(pass)) == 1
		if !ok || !userOK || !passOK {
			w.Header().Set("WWW-Authenticate", `Basic realm="monitor", charset="UTF-8"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

func main() {
	if os.Getenv("MONITOR_USER") == "" || os.Getenv("MONITOR_PASS") == "" {
		log.Fatal("MONITOR_USER and MONITOR_PASS must be set (see example.env)")
	}

	uptimeHistory = newUptimeLedger(os.Getenv("UPTIME_DIR"))
	go uptimeHistory.run(uptimeInterval(), nil)

	http.HandleFunc("/", requireAuth(handleIndex))
	http.HandleFunc("/api/overview", jsonEndpoint(func() any { return getOverview() }))
	http.HandleFunc("/api/cpu", jsonEndpoint(func() any { return getCPU() }))
	http.HandleFunc("/api/memory", jsonEndpoint(func() any { return getRAM() }))
	http.HandleFunc("/api/disk", jsonEndpoint(func() any { return getDiskUsage() }))
	http.HandleFunc("/api/hardware", jsonEndpoint(func() any { return getHardwareInfo() }))
	http.HandleFunc("/api/processes", jsonEndpoint(func() any { return getTopProcesses(15) }))
	http.HandleFunc("/api/health", jsonEndpoint(func() any { return getHealth() }))
	http.HandleFunc("/api/uptime", jsonEndpoint(func() any {
		return uptimeHistory.series(24*time.Hour, time.Now())
	}))
	// Slow (du scans / docker system df): cached so repeat loads are instant.
	http.HandleFunc("/api/folders", jsonEndpoint(func() any {
		return folderCache.get(2*time.Minute, func() any { return getDiskFolders() })
	}))
	http.HandleFunc("/api/docker", jsonEndpoint(func() any {
		return dockerCache.get(30*time.Second, func() any { return getDockerStats() })
	}))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	addr := ":" + port
	fmt.Printf("Monitor dashboard running at http://localhost%s\n", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
