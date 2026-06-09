# Local Development Lab

This repository contains the architectural blueprint for a modular, local development environment. It is designed for isolation, portability, and "as-code" infrastructure management, allowing for seamless replication across fresh Linux installations or Cloud VPS.

## Architecture Overview

The lab is organized into logical directories based on the intent and lifecycle of the services:

### 1. [core/](file:///home/mlovera/lab/core/) - Infrastructure & Shared Services
Baseline services that support the entire lab. These are generally "always-on."
* **proxy**: Nginx-based reverse proxy managing `.rpi.local` and `.mlovera.dev` domain routing ([ADR 002](file:///home/mlovera/lab/adr/002-revert-to-nginx.md)).
* **postgres**: Centralized PostgreSQL 17 instance with dynamic multi-database/user initialization.
* **mongo**: MongoDB document database.
* **redis**: Centralized Redis instance for shared caching.
* **elasticsearch**: Search and analytics engine.
* **cloudflared**: Cloudflare Tunnel for secure remote access.

### 2. [external/](file:///home/mlovera/lab/external/) - Third-Party Applications
Pre-built tools and platforms managed via Docker Compose.
* **n8n**: Workflow automation (backed by core Postgres and featuring a Python runner sidecar).

### 3. [services/](file:///home/mlovera/lab/services/) - Internal/Custom Development
Placeholder for custom apps and internal source code.

### 4. [shared/](file:///home/mlovera/lab/shared/) - Scripts & Utilities
Core automation logic, network configuration, and local management CLIs (`lab`, `secrets`).

### 5. [adr/](file:///home/mlovera/lab/adr/) - Architecture Decision Records
Design log documenting major architectural shifts (e.g., transition to Traefik, reverting to Nginx, removing centralized Azurite).

### 6. [docs/](file:///home/mlovera/lab/docs/) & [specs/](file:///home/mlovera/lab/specs/) - Documentation & Specifications
Detailed guides on networking, user credentials, and core services expansion specifications.

---

## Management Utilities (lab & secrets)

The lab provides two custom Bash CLI tools to streamline development and credentials management.

### The `lab` CLI
Located at [shared/lab](file:///home/mlovera/lab/shared/lab), this manages project lifecycles dynamically:
* `lab list` - Scans directories and lists all available projects.
* `lab ps [-a]` - Shows status of all projects, including active endpoints and direct IP:Port mappings.
* `lab up <name>` - Starts a specific project (e.g., `lab up n8n`).
* `lab down <name>` - Stops a specific project.
* `lab logs <name>` - Tails the real-time logs for a project.

### The `secrets` CLI
Located at [shared/secrets](file:///home/mlovera/lab/shared/secrets), this is backed by a local SQLite DB for credentials and audit logs:
* `secrets list` - Shows all active credentials found in project `.env` files.
* `secrets store <key> <value>` - Securely stores a key-value secret.
* `secrets get <key>` - Retrieves a stored secret.
* `secrets db create-user <postgres|mongo> <dbname> <username> <password>` - Automatically provisions database users and databases in the running containers.
* `secrets history` - Displays the user creation audit trail.

---

## Quick Start

### 1. Requirements
* **OS**: Ubuntu 22.04 LTS or newer (Linux system required).
* **Docker**: Docker Engine and Docker Compose V2 installed.
* **Permissions**: Current user must be in the `docker` group.

### 2. Initialization
Run the idempotent setup script to configure Docker DNS, initialize the `lab-network`, inject aliases, and symlink the CLIs to your PATH:
```bash
source ./shared/setup-lab.sh
```

### 3. Running Services
Once initialized, start your core databases or tools:
```bash
lab up postgres
lab up proxy
```

---

## Networking & Access Control

All containers communicate over a unified bridge network named `lab-network`. This enables service discovery via container names (e.g., n8n connecting to `postgres:5432`).

### Local & Production Access
| Service | Local URL (*.rpi.local) | Production URL (*.mlovera.dev) | Port Mapping (Direct) |
|---|---|---|---|
| **n8n** | http://n8n.rpi.local | https://n8n.mlovera.dev | `5678` |
| **elasticsearch** | http://elasticsearch.rpi.local | — | `9200` |
| **postgres** | — | — | `5432` |
| **mongo** | — | — | `27017` |
| **redis** | — | — | `6379` |

### Exposing a New HTTP Service
1. Edit the Nginx configuration file at [core/proxy/conf.d/proxy.conf](file:///home/mlovera/lab/core/proxy/conf.d/proxy.conf) to add a `server` block.
2. Reload Nginx dynamically:
   ```bash
   docker exec lab-proxy nginx -s reload
   ```
3. For remote production access, configure a public hostname mapping to `http://lab-proxy:80` inside the Cloudflare Zero Trust Dashboard (traffic is routed via the `cloudflared` tunnel).

For full details, refer to the [Networking Guide](file:///home/mlovera/lab/docs/networking.md) and [Credentials Guide](file:///home/mlovera/lab/docs/credentials.md).

