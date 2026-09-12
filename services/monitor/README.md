# System Monitor

A real-time system monitoring dashboard for Linux (Raspberry Pi / ARM), written in Go.

## Features

- Live CPU, RAM, and disk usage via SSE streaming
- Process list sorted by resource usage
- Temperature (via `vcgencmd`, falling back to the host thermal zone)
- Hardware info (board model, serial, kernel, arch)
- Disk breakdown by folder (home, usr, var, etc.) with Docker disk stats
- Network info (Wi-Fi SSID, IP)

## Endpoints

| Route           | Description              |
| --------------- | ------------------------ |
| `/`             | Web dashboard (HTML)     |
| `/api/stats`    | JSON snapshot            |
| `/api/stream`   | SSE stream (5s interval) |

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
