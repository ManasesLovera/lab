# System Monitor

A real-time system monitoring dashboard for Linux (Raspberry Pi / ARM), written in Go.

## Features

- Live CPU, RAM, and disk usage, loaded lazily per card
- App health for every Docker container (healthy / degraded / down)
- Availability history per app (24h / 7d / 30d) with restart counts, stored as a
  transition ledger so it costs a line per state change, not per poll
- Process list sorted by resource usage
- Temperature (via `vcgencmd`, falling back to the host thermal zone)
- Hardware info (board model, serial, kernel, arch)
- Disk breakdown by folder (home, usr, var, etc.) with Docker disk stats
- Network info (Wi-Fi SSID, IP)

## Endpoints

| Route            | Description                                   |
| ---------------- | --------------------------------------------- |
| `/`              | Web dashboard (tabbed, light/dark)            |
| `/api/overview`  | Hostname, uptime, temperature, network        |
| `/api/cpu`       | CPU usage, cores, model                       |
| `/api/memory`    | RAM usage                                     |
| `/api/disk`      | Root disk usage                               |
| `/api/hardware`  | Board, kernel, serial, firmware               |
| `/api/processes` | Top processes                                 |
| `/api/health`    | Health of every container (ok / warn / down)  |
| `/api/uptime`    | 24h up/down timeline segments per container   |
| `/api/folders`   | Folder size breakdown (slow, cached 2 min)    |
| `/api/docker`    | Docker storage usage (cached 30 s)            |

Each dashboard card requests only its own endpoint, so fast data renders
without waiting for the slow `du` scans.

All routes require HTTP basic auth (`MONITOR_USER` / `MONITOR_PASS`).

## Configuration

Copy `example.env` to `.env` and set credentials. `.env` is git-ignored and must
never be committed.

```bash
cp example.env .env
```

The app refuses to start if `MONITOR_USER` or `MONITOR_PASS` is unset.

When run in a container, it reports the host's metrics via these env vars (set
in `docker-compose.yml`):

| Variable     | Purpose                                   |
| ------------ | ----------------------------------------- |
| `HOST_ROOT`  | Host root mount (disk usage, `/etc`)      |
| `HOST_PROC`  | Host `/proc` (CPU, RAM, processes)        |
| `HOST_SYS`   | Host `/sys` (thermal, device tree, NICs)  |
| `HOST_ETC`   | Host `/etc` (distro/platform info)        |
| `UPTIME_DIR` | Directory for the availability ledger; empty disables persistence |
| `UPTIME_INTERVAL` | Docker state poll interval (Go duration, default `30s`) |

Availability is measured by polling `docker ps -a`. A container counts as up
when it is running and its healthcheck (if any) is not failing. Only state
changes are persisted, so the ledger stays tiny. Percentages are shown only for
windows the container has existed through, and the monitor cannot measure
downtime while it is itself stopped — use an external prober for end-to-end
uptime.

## Build & Run

### Locally

```bash
go build -o monitor .
MONITOR_USER=admin MONITOR_PASS=secret ./monitor
```

Opens at `http://localhost:8080`.

### In the lab

```bash
lab up monitor      # build + start
lab logs monitor    # follow logs
lab down monitor    # stop
```

Reachable at `http://monitor.rpi.local` (local) and
`https://monitor.mlovera.dev` (production, via the Cloudflare Tunnel).

> The container mounts the host root read-only and the Docker socket (for
> `docker system df`). Treat it as privileged infrastructure and keep the
> basic-auth credentials strong.
