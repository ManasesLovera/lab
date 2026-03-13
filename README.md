# Local Development Lab

This repository contains the architectural blueprint for a modular, local development environment. It is designed for isolation, portability, and "as-code" infrastructure management, allowing for seamless replication across fresh Linux installations or Cloud VPS.

## Architecture Overview

The lab is organized into four main directories based on the intent and lifecycle of the services:

### 1. core/ - Infrastructure & Shared Services
Baseline services that support the entire lab. These are generally "always-on."
*   **postgres**: Centralized PostgreSQL 17 instance with dynamic multi-database/user initialization.
*   **proxy**: Nginx-based reverse proxy managing .local domain routing.
*   **dockge**: Web-based dashboard for managing Docker Compose stacks.
*   **redis**: (Optional) Centralized Redis instance for shared caching.

### 2. external/ - Third-Party Applications
Pre-built tools and platforms managed via Docker Compose.
*   **n8n**: Workflow automation (backed by core Postgres).
*   **ollama**: Local LLM runner.
*   **openclaw**: Autonomous AI agent gateway.
*   **openproject**: Project management platform (backed by core Postgres).

### 3. services/ - Internal/Custom Development
Placeholder for custom apps and internal source code.

### 4. shared/ - Scripts & Utilities
Core automation logic, network configuration, and the lab CLI.

---

## Management CLI (lab)

The lab includes a custom Bash CLI for streamlined management. Once setup, it is available globally via the lab command.

### Usage:
*   `lab list` - Scans all directories and lists available projects.
*   `lab ps` - Shows running status of all lab projects.
*   `lab ps -a` - Shows status of all projects, including stopped ones.
*   `lab up <name>` - Starts a specific project (e.g., `lab up n8n`).
*   `lab down <name>` - Stops a specific project.
*   `lab logs <name>` - Tails the real-time logs for a project.
*   `lab help` - Shows the full command guide.

---

## Quick Start

### 1. Requirements
*   **OS**: Ubuntu 22.04 LTS or newer.
*   **Docker**: Docker Engine and Docker Compose V2 installed.
*   **Permissions**: Current user must be in the docker group.

### 2. Initialization
Run the idempotent setup script to configure DNS, initialize the lab-network, inject aliases, and install the lab CLI into your path:

```bash
source ./shared/setup-lab.sh
```

### 3. Accessing the Lab
Once initialized, the core infrastructure (Postgres, Proxy, Dockge) will start automatically. You can access your services at:
*   **Dockge Dashboard**: [http://dockge.local](http://dockge.local)
*   **n8n**: [http://n8n.local](http://n8n.local)
*   **OpenProject**: [http://openproject.local](http://openproject.local)

*(Note: Ensure you add these to your /etc/hosts file or use the helper command provided during setup.)*

---

## Networking
All containers communicate over a unified bridge network named lab-network. This enables service discovery via container names (e.g., n8n connecting to postgres:5432).
