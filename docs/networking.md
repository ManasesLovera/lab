# Networking Guide: Lab Routing & Access Control

This guide explains how to manage access to your services within the Gemini Lab environment.

## Current Lab Status (Default Configuration)

The following table shows the default accessibility for all services in the lab:

| Service | Local Network (*.rpi.local) | Production (*.mlovera.dev) | Port Mapping (Direct) |
| :--- | :---: | :---: | :---: |
| **Traefik** | `http://traefik.rpi.local` | - | 80, 8080 (Dashboard) |
| **n8n** | `http://n8n.rpi.local` | `https://n8n.mlovera.dev` | 5678 |
| **Postgres** | `postgres.rpi.local` | - | 5432 |
| **Redis** | `redis.rpi.local` | - | 6379 |
| **MongoDB** | `mongo.rpi.local` | - | 27017 |
| **Elasticsearch**| `elasticsearch.rpi.local`| - | 9200 |
| **Azurite** | `azurite.rpi.local` | - | 10000, 10001, 10002 |
| **MSSQL** | `mssql.rpi.local` | - | 1433 |

---

## Overview
The lab uses **Traefik** as a dynamic reverse proxy. Routing is controlled by **Docker Labels** defined in each service's `docker-compose.yml`.

---

## 1. Local Network Access (*.rpi.local)
To make a service accessible to other devices on your home network via a domain:
1. Ensure the service maps its port to the host in `ports:`.
2. Add Traefik labels for the `.rpi.local` domain.

```yaml
services:
  myservice:
    ports:
      - "8080:8080" # Optional: Direct TCP access
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.myservice-local.rule=Host(`myservice.rpi.local`)"
      - "traefik.http.routers.myservice-local.entrypoints=web"
```

## 2. Production/Remote Access (*.mlovera.dev)
To expose a service to the internet via the Cloudflare Tunnel, you must configure both the Cloudflare Dashboard and the local lab settings.

### Step A: Cloudflare Dashboard Configuration
1. Log in to the [Cloudflare Zero Trust Dashboard](https://one.dash.cloudflare.com/).
2. Navigate to **Networks** > **Tunnels**.
3. Select your active tunnel (matching `${TUNNEL_ID}`) and click **Configure**.
4. Go to the **Public Hostname** tab.
5. Click **Add a public hostname**.
6. **Public Hostname Settings**:
   - **Subdomain**: e.g., `myservice`
   - **Domain**: `mlovera.dev`
7. **Service Settings**:
   - **Type**: `HTTP`
   - **URL**: `lab-traefik:80` (This sends the traffic to our Traefik proxy).
8. Click **Save hostname**.

### Step B: Local Traefik Labels
Add the production hostname label to your service's `docker-compose.yml`:
```yaml
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.myservice-prod.rule=Host(`myservice.mlovera.dev`)"
      - "traefik.http.routers.myservice-prod.entrypoints=web"
```

### Step C: Cloudflare Local Configuration (`core/cloudflared/config.yml`)
The lab uses a **Wildcard Ingress Rule** to minimize local changes. Ensure your `config.yml` contains the following:

```yaml
ingress:
  # Specific rules (like SSH) come first
  - hostname: rpi.mlovera.dev
    service: ssh://host.docker.internal:22
  
  # Wildcard rule: Routes all subdomains to Traefik
  - hostname: "*.mlovera.dev"
    service: http://lab-traefik:80

  # Catch-all rule (Mandatory)
  - service: http_status:404
```
*Note: The wildcard rule in `config.yml` means you usually don't need to edit this file when adding new subdomains; you only need to add them in the Cloudflare Dashboard and add the Traefik labels locally.*

## 3. Dual Access (Local + Production)
You can combine both rules in a single router using the OR (`||`) operator:

```yaml
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.myservice.rule=Host(`myservice.rpi.local`) || Host(`myservice.mlovera.dev`)"
```

## 4. Localhost Only (Internal)
To restrict a service so it is **only** accessible from the Raspberry Pi itself or other containers:
1. **Remove all Traefik labels.**
2. **Remove the `ports:` section** (or bind specifically to `127.0.0.1`).
3. Access it via its container name (e.g., `http://postgres:5432`) from within the `lab-network`.

```yaml
services:
  internal-app:
    # No ports exposed to 0.0.0.0
    # No traefik labels
    networks:
      - lab-network
```

## 5. Shared Infrastructure (TCP/Database)
For non-HTTP services (Postgres, Redis, Mongo), Traefik routing by domain is more complex. The standard practice in this lab is:
1. **Direct Port Mapping**: Map the port in `docker-compose.yml` (e.g., `5432:5432`).
2. **DNS Resolution**: Your router/DNS should point `postgres.rpi.local` to the Pi's static IP.
3. Applications connect directly to `postgres.rpi.local:5432`.
