# Local Development Lab

This repository contains the architectural blueprint for a modular, local development environment. It is designed for isolation, portability, and "as-code" infrastructure management, allowing for seamless replication across fresh Linux installations or Cloud VPS.

## Architecture Overview

The lab is organized into four main directories based on the intent and lifecycle of the services:

### 1. core/ - Infrastructure & Shared Services

Baseline services that support the entire lab. These are generally "always-on."

* **postgres**: Centralized PostgreSQL 17 instance with dynamic multi-database/user initialization.
* **proxy**: Nginx-based reverse proxy managing `.rpi.local` and `.mlovera.dev` domain routing.
* **redis**: Centralized Redis instance for shared caching.
* **elasticsearch**: Search and analytics engine, powers Kibana.
* **mongo**: MongoDB document database.
* **mssql**: Microsoft SQL Server 2022.
* **azurite**: Azure Storage API emulator (blob, queue, table).
* **cloudflared**: Cloudflare Tunnel for secure remote access.

### 2. external/ - Third-Party Applications

Pre-built tools and platforms managed via Docker Compose.

* **n8n**: Workflow automation (backed by core Postgres).
* **kibana**: Elasticsearch visualization and management UI.
* **mongo-express**: Web-based MongoDB admin interface.

### 3. services/ - Internal/Custom Development

Placeholder for custom apps and internal source code.

### 4. shared/ - Scripts & Utilities

Core automation logic, network configuration, and the lab CLI.

---

## Management CLI (lab)

The lab includes a custom Bash CLI for streamlined management. Once setup, it is available globally via the `lab` command.

### Usage

* `lab list` - Scans all directories and lists available projects.
* `lab ps` - Shows running status of all lab projects.
* `lab ps -a` - Shows status of all projects, including stopped ones.
* `lab up <name>` - Starts a specific project (e.g., `lab up n8n`).
* `lab down <name>` - Stops a specific project.
* `lab logs <name>` - Tails the real-time logs for a project.
* `lab help` - Shows the full command guide.

---

## Quick Start

### 1. Requirements

* **OS**: Ubuntu 22.04 LTS or newer.
* **Docker**: Docker Engine and Docker Compose V2 installed.
* **Permissions**: Current user must be in the `docker` group.

### 2. Initialization

Run the idempotent setup script to configure DNS, initialize the lab-network, inject aliases, and install the lab CLI into your path:

```bash
source ./shared/setup-lab.sh
```

### 3. Accessing the Lab

Once initialized, the core infrastructure (Postgres, Proxy, Redis, etc.) will start automatically. Access your services at:

| Service | Local URL | Direct IP:Port |
|---|---|---|
| n8n | http://n8n.rpi.local | `192.168.1.8:5678` |
| elasticsearch | http://elasticsearch.rpi.local | `192.168.1.8:9200` |
| kibana | http://kibana.rpi.local | `192.168.1.8:5601` |
| mongo-express | http://mongo-express.rpi.local | — |
| postgres | — | `192.168.1.8:5432` |
| mongo | — | `192.168.1.8:27017` |
| redis | — | `192.168.1.8:6379` |
| mssql | — | `192.168.1.8:1433` |
| azurite | — | `192.168.1.8:10000-10002` |

*(Note: Add `192.168.1.8` entries to your `/etc/hosts` or use the helper command provided during setup for `.rpi.local` resolution.)*

### 4. Production Access

Services with a `*.mlovera.dev` production domain are exposed via Cloudflare Tunnel. See `core/cloudflared/README.md` for details.

---

## Default Credentials

| Service | User | Password | Auth Method |
|---|---|---|---|
| **Postgres** | `admin` | `P@ssw0rd!Adm1n#2024` | Password (`.env`) |
| **MongoDB** | `admin` | `admin_password` | Password (`.env`) |
| **MSSQL** | `sa` | `StrongPassword123!` | SQL Server Auth (`.env`) |
| **Elasticsearch** | `elastic` | `admin_password` | Basic Auth (`.env`) |
| **Kibana** | `elastic` | `admin_password` | Login form (delegates to ES) |
| **Azurite** | `devstoreaccount1` | `Eby8vdM02x...` (well-known key) | Storage account key |
| **Redis** | *(none — no auth)* | — | Network-only |
| **n8n** | *(self-registered)* | *(first-run setup)* | Registration form |
| **Mongo Express** | `admin` | `pass` | Basic Auth (UI) |

See `docs/credentials.md` for detailed user management, permission grants, and how to add new users to each database.

---

## Networking

All containers communicate over a unified bridge network named `lab-network`. This enables service discovery via container names (e.g., n8n connecting to `postgres:5432`).

### HTTP Routing

HTTP traffic is handled by Nginx (`lab-proxy`) via `core/proxy/conf.d/proxy.conf`. To expose a new HTTP service:
1. Edit `core/proxy/conf.d/proxy.conf` to add a `server_name` and `proxy_pass` block.
2. Reload the proxy: `docker exec lab-proxy nginx -s reload`.

### Remote Access

Production traffic (`*.mlovera.dev`) is routed through Cloudflare Tunnel (`cloudflared`) to `lab-proxy:80`, which matches the production `server_name` entries in the Nginx config.

See `docs/networking.md` for full routing and access control details.
