# Lab Development Environment - Agent Guide

## Setup & Initialization
- **Required**: `source ./shared/setup-lab.sh` (must be sourced to activate aliases)
- Sets up Docker DNS, installs `lab` CLI, creates `cloudflared` alias
- Starts core infrastructure (postgres, proxy, cloudflared)

## Core Commands (via `lab` CLI)
All commands require project name from: core/, external/, services/

- `lab list` - Shows all available projects
- `lab up <name>` - Start project (e.g., `lab up n8n`)
- `lab down <name>` - Stop project
- `lab logs <name>` - Follow logs
- `lab ps [-a]` - Show running (or all with -a) projects

## Project Structure
- **core/** - Infrastructure services (postgres, proxy, redis, cloudflared)
- **external/** - Third-party apps (n8n)
- **services/** - Custom/internal development (currently empty)
- **shared/** - CLI, scripts, network config
- **adr/** - Architecture Decision Records

## Key Configuration
- Global env: `./.env` (controls ENV=local/production, domain suffix)
- Services use `lab-network` (external bridge network)
- Production mode uses Cloudflare Tunnel (requires TUNNEL_TOKEN in .env)
- HTTP routing is handled by Nginx (`lab-proxy`) via `core/proxy/conf.d/proxy.conf`

## Common Workflows
1. Initialize: `source ./shared/setup-lab.sh`
2. List services: `lab list`
3. Start service: `lab up n8n` (or postgres, proxy, etc.)
4. Access: http://n8n.rpi.local (local) or https://n8n.mlovera.dev (production)
5. Stop service: `lab down n8n`

## Notes
- Docker-compose files use project name as service identifier
- Core services (postgres, proxy) start automatically during setup
- `.env` must be present for proper domain configuration
- To add a new HTTP service, edit `core/proxy/conf.d/proxy.conf` and run `docker exec lab-proxy nginx -s reload`