# Gemini Lab - Home Lab Architectural Context

This file provides the necessary context for Gemini to understand and interact with this project effectively.

## Project Overview

**Gemini Lab** is a modular, "infrastructure-as-code" home lab environment designed for Linux (specifically Ubuntu). It uses Docker Compose to manage isolated services, Nginx for reverse proxying, and Cloudflare Tunnels for secure external access.

### Core Architecture
- **Isolation:** Each project (service) has its own directory and `docker-compose.yml`.
- **Networking:** All services share a common bridge network named `lab-network`.
- **Centralized Data:** A core PostgreSQL instance handles multiple databases for different services (n8n, OpenProject, etc.).
- **Access Management:** 
    - **Local:** Routed via `lab-proxy` (Nginx) using `.local` (or configured) domains.
    - **Remote:** Cloudflare Tunnel (`lab-cloudflared`) exposes services to the internet via `*.mlovera.dev`.

## Directory Structure

- `core/`: Fundamental infrastructure services (always-on).
    - `postgres/`: Centralized DB with dynamic init scripts.
    - `proxy/`: Nginx reverse proxy with dynamic templates.
    - `cloudflared/`: Secure tunnel to Cloudflare Zero Trust.
    - `redis/`: Shared cache service.
- `external/`: Third-party applications (n8n, Ollama, OpenProject, OpenClaw).
- `services/`: Custom applications developed within the lab.
- `shared/`: Automation scripts and the `lab` CLI.

## Key Management Tools

### The `lab` CLI
The `lab` command is the primary way to manage the environment.
- `lab list`: Show all discovered projects.
- `lab ps [-a]`: Show status of running (or all) projects.
- `lab up <name>`: Start a project.
- `lab down <name>`: Stop a project.
- `lab logs <name>`: View service logs.
- `lab sync-dockge`: Sync projects as symlinks for the Dockge dashboard.

### Initialization
Run `source ./shared/setup-lab.sh` to initialize the environment, configure Docker DNS, create the network, and inject the CLI aliases.

## Development Conventions

- **Configuration:** Use `lab/.env` for global environment variables.
- **Service Discovery:** Services should refer to each other by their container names within the `lab-network`.
- **Database Access:** Use the `POSTGRES_MULTIPLE_DATABASES` variable in `lab/core/postgres/docker-compose.yml` to automatically create new databases/users on startup.
- **Proxying:** Add new subdomains to `lab/core/proxy/templates/proxy.conf.template` and ensure they match the `DOMAIN_SUFFIX`.

## Cloudflare Tunnel (Remote Access)
The tunnel is currently managed via **Cloudflare Zero Trust Dashboard** (Remote Management). 
- To expose a new service, add a Public Hostname in the dashboard pointing to `http://lab-proxy:80`.
- To expose SSH, point to `ssh://host.docker.internal:22`.

## Current Roadmap (Tasks)
See `lab/tasks.md` for the latest epic and task tracking.
