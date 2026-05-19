# Networking Guide: Lab Routing & Access Control

This guide explains how to manage access to your services within the Lab environment.

## Current Lab Status

| Service | Local Network (*.rpi.local) | Production (*.mlovera.dev) | Port Mapping (Direct) |
| :--- | :---: | :---: | :---: |
| **n8n** | `http://n8n.rpi.local` | `https://n8n.mlovera.dev` | 5678 |
| **Elasticsearch**| `http://elasticsearch.rpi.local` | — | 9200 |
| **Postgres** | *(IP only)* | — | 5432 |
| **MongoDB** | *(IP only)* | — | 27017 |
| **Redis** | *(IP only)* | — | 6379 |
| **MSSQL** | *(IP only)* | — | 1433 |
| **Azurite** | *(IP only)* | — | 10000, 10001, 10002 |

---

## Overview
The lab uses **Nginx** as a static reverse proxy. Routing is configured via `server` blocks in `core/proxy/conf.d/proxy.conf`. Each HTTP service has its own `server` block with hardcoded domain names.

Non-HTTP services (Postgres, Mongo, Redis, MSSQL, Azurite) are accessed directly via IP:Port — they are not proxied through Nginx.

---

## 1. Local Network Access (*.rpi.local)
To expose an HTTP service to the local network:
1. Ensure the service is on `lab-network` with a stable container name.
2. Add a `server` block to `core/proxy/conf.d/proxy.conf`:

```nginx
server {
    listen 80;
    server_name myservice.rpi.local;

    resolver 127.0.0.11 valid=30s;
    set $upstream_myservice myservice-container;

    location / {
        proxy_pass http://$upstream_myservice:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

3. Reload nginx: `docker exec lab-proxy nginx -s reload`

## 2. Production/Remote Access (*.mlovera.dev)
To expose a service to the internet via the Cloudflare Tunnel:

### Step A: Cloudflare Dashboard Configuration
1. Log in to the [Cloudflare Zero Trust Dashboard](https://one.dash.cloudflare.com/).
2. Navigate to **Networks** > **Tunnels**.
3. Select your active tunnel and click **Configure**.
4. Go to the **Public Hostname** tab.
5. Click **Add a public hostname**.
6. **Subdomain**: e.g., `myservice`
7. **Domain**: `mlovera.dev`
8. **Service**: `HTTP` → `lab-proxy:80`
9. Click **Save hostname**.

### Step B: Local Nginx Configuration
Add the production hostname to the service's `server_name` in `core/proxy/conf.d/proxy.conf`:

```nginx
server {
    listen 80;
    server_name myservice.rpi.local myservice.mlovera.dev;
    ...
}
```

Then reload: `docker exec lab-proxy nginx -s reload`

### Step C: Cloudflare Local Configuration (`core/cloudflared/config.yml`)
The wildcard ingress rule forwards all `*.mlovera.dev` traffic to the proxy:

```yaml
ingress:
  - hostname: rpi.mlovera.dev
    service: ssh://host.docker.internal:22

  - hostname: "*.mlovera.dev"
    service: http://lab-proxy:80

  - service: http_status:404
```

No changes needed here when adding new services — just add them in Cloudflare Dashboard and add the `server_name` to nginx.

## 3. Localhost Only (Internal)
To restrict a service so it is **only** accessible from the Raspberry Pi itself or other containers:
1. **Do not add an nginx `server` block.**
2. **Remove the `ports:` section** (or bind specifically to `127.0.0.1`).
3. Access it via its container name (e.g., `http://postgres:5432`) from within `lab-network`.

```yaml
services:
  internal-app:
    # No ports exposed to 0.0.0.0
    # No nginx server block
    networks:
      - lab-network
```

## 4. Shared Infrastructure (TCP/Database)
For non-HTTP services (Postgres, Redis, Mongo, MSSQL), nginx does not route traffic. The standard practice is:
1. **Direct Port Mapping**: Map the port in `docker-compose.yml` (e.g., `5432:5432`).
2. Applications connect directly to `192.168.1.8:<port>` or via container name from within `lab-network`.
