# Planboard — Lab Deployment Plan

Add Planboard (web-app, Postgres-backed) to the Gemini Lab as a managed `services/` project, exposed via `lab-proxy` (nginx) + Cloudflare Tunnel at `planboard.mlovera.dev` and `planboard.rpi.local`.

## Pre-requisite — Done ✅
Private `ghcr.io/weleec/planboard:latest` image pull access authenticated:
```bash
gh auth refresh -h github.com -s read:packages      # done
gh auth token | docker login ghcr.io -u ManasesLovera --password-stdin   # done
docker manifest inspect ghcr.io/weleec/planboard:latest   # verified → linux/arm64
```

## Step 1 — Create `services/planboard/`
New directory following `services/todo` conventions.

**`services/planboard/docker-compose.yml`**
```yaml
name: planboard

services:
  planboard:
    image: ghcr.io/weleec/planboard:latest
    container_name: planboard
    restart: unless-stopped
    env_file:
      - .env
    environment:
      PB_DB: postgres
    networks:
      - lab-network
    healthcheck:
      start_period: 60s
      interval: 30s
      timeout: 10s
      retries: 3
    logging:
      driver: json-file
      options:
        max-size: "10m"
        max-file: "3"

networks:
  lab-network:
    external: true
```
- **No `ports:`** — only reachable via proxy over `lab-network` (app's own security model; no Cloudflare-Access-uncovered entry point).
- **No volume** — Postgres mode (`/app/var` SQLite volume not needed).

**`services/planboard/.env`** (git-ignored; real credentials live only in this untracked file)
```
PB_DB=postgres
PB_DATABASE_URL=postgres://weleec_planboard:REDACTED@postgres:5432/weleec_planboard
```
- Host is the `postgres` container name over `lab-network` (not host IP `10.90.100.139`).
- `#` stays encoded as `%23`.

**`services/planboard/example.env`**
```
PB_DB=postgres
PB_DATABASE_URL=postgres://USER:PASSWORD@postgres:5432/DATABASE
```

**`services/planboard/README.md`** — sections: How to Run, How to Use, Configuration, Networking (mirror `services/todo/README.md` style).

## Step 2 — Nginx routing
Append a server block to `core/proxy/conf.d/proxy.conf`:
```nginx
# Planboard — Local + Production
server {
    listen 80;
    server_name planboard.rpi.local planboard.mlovera.dev;

    resolver 127.0.0.11 valid=30s;
    set $upstream_planboard planboard;

    location / {
        proxy_pass http://$upstream_planboard:3000;
        proxy_set_header Host              $host;
        proxy_set_header X-Real-IP         $remote_addr;
        proxy_set_header X-Forwarded-For   $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto https;
        proxy_read_timeout 120s;
    }
}
```
Reload: `docker exec lab-proxy nginx -s reload`
- `X-Forwarded-For` is required (login throttle keys on `x-forwarded-for.split(',')[0]`).
- `proxy_read_timeout 120s` for in-memory workbook exports.

## Step 3 — Cloudflare public hostname
Add **Public Hostname** in the tunnel dashboard (Networks → Tunnels → Public Hostname):
- Subdomain: `planboard`, Domain: `mlovera.dev`
- Service: **HTTP** → `lab-proxy:80`
- The locally-managed tunnel's ingress already wildcards `*.mlovera.dev → lab-proxy:80` (`core/cloudflared/config.yml`), so no cloudflared config change.
- Note: `CLOUDFLARE_API_TOKEN` in `.env` currently returns auth errors → dashboard step is manual.

## Step 4 — Deploy & verify
```bash
lab up planboard
docker exec planboard bun --eval \
  "fetch('http://127.0.0.1:3000/api/health').then(r=>r.text()).then(console.log)"
# expect {"status":"ok","driver":"postgres",...}
```
Verify endpoints:
- `https://planboard.mlovera.dev` (production, via tunnel)
- `http://planboard.rpi.local` (local network)

## Step 5 — Docs
Add a row to `docs/networking.md` "Current Lab Status" table:
```
| **Planboard** | `http://planboard.rpi.local` | `https://planboard.mlovera.dev` | 3000 |
```

## Notes / decisions
- **DB provisioning:** none needed — `weleec_planboard` role + database + 15-table schema already exist in the local `postgres`; the app's baseline migrations are idempotent (`CREATE ... IF NOT EXISTS`), so startup is safe.
- **Image tag:** keep `:latest` (user's choice). Repo recommends sha-pinning later for rollback safety.
- **MCP container deferred:** `planboard-mcp` (HTTP MCP, stage 5) is not built yet (`scripts/mcp-http-server.ts` doesn't exist). Not deployed now; web-app only.
- **No commit:** these changes will not be committed; user may delete them afterward.
