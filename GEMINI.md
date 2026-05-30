# Gemini Lab - Home Lab Architectural Context

This file provides the foundational context for Gemini to understand, maintain, and extend this project effectively.

## Project Overview

**Gemini Lab** is a modular, "infrastructure-as-code" home lab environment designed for Linux (specifically Ubuntu/Raspberry Pi). It uses Docker Compose for service isolation, **Traefik v3** for dynamic reverse proxying, and **Cloudflare Tunnels** for secure remote access.

## Core Mandates & Engineering Standards

- **Conventional Commits:** ALWAYS use Conventional Commits (e.g., `feat:`, `fix:`, `chore:`, `docs:`) for all commit messages.
- **Surgical Updates:** Use `replace` for targeted edits; avoid overwriting files unless necessary.
- **Documentation:** Every new service MUST include a `README.md` and an `example.env`.
- **Credential Safety:** NEVER commit `.env` files. Always check `.gitignore` and use `example.env` as a reference.
- **Inquiry vs. Directive:** Distinguish strictly between **Inquiries** (requests for analysis, advice, or observations) and **Directives** (explicit instructions to perform a task). For Inquiries, your scope is strictly limited to research and analysis; you MUST NOT modify files until a corresponding Directive is issued. Assume all ambiguous requests are Inquiries.

## Core Architecture

### 1. Networking Strategy
- **Shared Network:** All services join the `lab-network` (external bridge).
- **Internal Discovery:** Services communicate via container names (e.g., `http://postgres:5432`).
- **External/Local Access:** 
    - **HTTP:** Managed by **Traefik**. Services use Docker Labels to define domains (e.g., `*.rpi.local`, `*.mlovera.dev`).
    - **TCP (Databases):** Mapped directly to host ports for local network access (e.g., `5432:5432`).
- **Remote Ingress:** `lab-cloudflared` creates a secure tunnel. Wildcard `*.mlovera.dev` is routed to `lab-traefik:80`.

### 2. Centralized Infrastructure
- **Shared Databases:** Multiple services share single instances of Postgres and MongoDB to reduce resource consumption.
- **Dynamic Provisioning:** Postgres uses `init-db.sh` to create multiple databases/users based on the `POSTGRES_MULTIPLE_DATABASES` env var.

## Directory Structure

- `adr/`: Architecture Decision Records (e.g., Nginx to Traefik transition).
- `core/`: Fundamental infrastructure (Always-on: Traefik, Cloudflared, Postgres, Redis, Mongo).
- `docs/`: General documentation (Networking Guide, Troubleshooting).
- `external/`: Third-party applications (n8n).
- `services/`: Custom applications developed within the lab.
- `shared/`: Utility scripts and the `lab` CLI.
- `specs/`: Technical specifications for features and core expansions.

## Key Management Tools

### The `lab` CLI
Located at `shared/lab`, this dynamic tool manages project lifecycles.
- `lab list`: Scans directories for `docker-compose.yml` and lists projects.
- `lab ps [-a]`: Checks running status across all discovered projects.
- `lab up <name>`: Starts a project (automatically handles `.env` if present).
- `lab down <name>`: Stops a project.
- `lab logs <name>`: Follows container logs.

### Environment Setup
Run `source ./shared/setup-lab.sh` to:
1. Initialize the `lab-network`.
2. Configure Docker DNS.
3. Inject the `lab` alias into the shell.

## Development Workflows

### Adding a New Service
1. Create a directory in `core/`, `external/`, or `services/`.
2. Add a `docker-compose.yml` joining `lab-network`.
3. Add Traefik labels for HTTP routing (refer to `docs/networking.md`).
4. Add `example.env` and `README.md`.
5. Run `lab up <name>` to verify.

### Modifying Shared Infrastructure
- Refer to `specs/core-services-expansion.md` before adding new core databases or storage.
- Ensure all TCP ports are mapped to the host if local network access is required.

## Cloudflare Remote Access
1. Add a **Public Hostname** in the Cloudflare Zero Trust Dashboard.
2. Point it to `http://lab-traefik:80`.
3. Add a corresponding `Host('subdomain.mlovera.dev')` label to the service in its `docker-compose.yml`.

## Task Tracking
See `tasks.md` for the current roadmap and epic status.
