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

type SystemStats struct {
	Hostname string        `json:"hostname"`
	Uptime   string        `json:"uptime"`
	Temp     float64       `json:"temp"`
	Hardware HardwareStats `json:"hardware"`
	CPU      CPUStats      `json:"cpu"`
	RAM      RAMStats      `json:"ram"`
	Disk     DiskStats     `json:"disk"`
	Network  NetStats      `json:"network"`
	Procs    []ProcInfo    `json:"processes"`
}

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

func getSystemStats() SystemStats {
	stats := SystemStats{}

	// Hostname (prefer the host's /etc/hostname when running in a container).
	if h, err := host.Info(); err == nil {
		stats.Hostname = h.Hostname
		uptime := time.Duration(h.Uptime) * time.Second
		hours := int(uptime.Hours())
		mins := int(uptime.Minutes()) % 60
		stats.Uptime = fmt.Sprintf("%dh %dm", hours, mins)
	}
	if name, err := os.ReadFile(hostPath("etc/hostname")); err == nil {
		if s := strings.TrimSpace(string(name)); s != "" {
			stats.Hostname = s
		}
	}

	// Temperature
	if temp, err := getTemperature(); err == nil {
		stats.Temp = temp
	}

	// Hardware
	stats.Hardware = getHardwareInfo()

	// CPU
	if cpuPercent, err := cpu.Percent(500*time.Millisecond, false); err == nil && len(cpuPercent) > 0 {
		stats.CPU.Percent = cpuPercent[0]
	}
	if cpuInfo, err := cpu.Info(); err == nil && len(cpuInfo) > 0 {
		stats.CPU.Cores = len(cpuInfo)
		stats.CPU.Model = cpuInfo[0].ModelName
		if stats.CPU.Model == "" {
			stats.CPU.Model = fmt.Sprintf("%s %s", cpuInfo[0].VendorID, cpuInfo[0].Family)
		}
	}

	// RAM
	if vm, err := mem.VirtualMemory(); err == nil {
		stats.RAM.Total = vm.Total
		stats.RAM.Used = vm.Used
		stats.RAM.Available = vm.Available
		stats.RAM.Percent = vm.UsedPercent
	}

	// Disk
	if usage, err := disk.Usage(hostPath("/")); err == nil {
		stats.Disk.Total = usage.Total
		stats.Disk.Used = usage.Used
		stats.Disk.Available = usage.Free
		stats.Disk.Percent = usage.UsedPercent
	}
	stats.Disk.Folders = getDiskFolders()
	stats.Disk.Docker = getDockerStats()

	// Network
	stats.Network = getNetworkInfo()

	// Processes
	stats.Procs = getTopProcesses(15)

	return stats
}

func getTemperature() (float64, error) {
	// Preferred: Raspberry Pi firmware tool (only present on the host).
	if out, err := exec.Command("vcgencmd", "measure_temp").Output(); err == nil {
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
	if gpu, err := exec.Command("vcgencmd", "get_mem", "gpu").Output(); err == nil {
		hw.GPU = strings.TrimSpace(string(gpu))
		// Parse "gpu=76M"
		if idx := strings.Index(hw.GPU, "="); idx >= 0 {
			hw.GPU = hw.GPU[idx+1:]
		}
	}

	// GPU firmware version
	if fw, err := exec.Command("vcgencmd", "version").Output(); err == nil {
		lines := strings.Split(string(fw), "\n")
		if len(lines) >= 2 {
			hw.Firmware = strings.TrimSpace(lines[1])
		} else {
			hw.Firmware = strings.TrimSpace(string(fw))
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
	// Get first user directory
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil
	}

	for _, entry := range entries {
		if entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") {
			userDir := filepath.Join(path, entry.Name())
			// Get contents of user's home directory
			return getSubFolders(userDir)
		}
	}
	return nil
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
	cmd := exec.Command("wpa_cli", "-i", "wlan0", "status")
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

func handleStats(w http.ResponseWriter, r *http.Request) {
	stats := getSystemStats()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func handleStream(w http.ResponseWriter, r *http.Request) {
	// Set headers for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	// Send initial data
	stats := getSystemStats()
	data, _ := json.Marshal(stats)
	fmt.Fprintf(w, "data: %s\n\n", data)
	flusher.Flush()

	// Create ticker for updates
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			stats := getSystemStats()
			data, _ := json.Marshal(stats)
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
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

	http.HandleFunc("/", requireAuth(handleIndex))
	http.HandleFunc("/api/stats", requireAuth(handleStats))
	http.HandleFunc("/api/stream", requireAuth(handleStream))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	addr := ":" + port
	fmt.Printf("Monitor dashboard running at http://localhost%s\n", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
