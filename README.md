# Local Development Lab

This repository contains the architectural blueprint for my local development environment on Ubuntu Desktop/Server. It is designed for modularity, isolation, and portability, allowing me to mirror this setup on any Cloud VPS or any fresh Linux installation.

## Architecture Overview

The lab is organized by intent, separating third-party services from internal source code and core infrastructure.

## Core Components

1. Networking:
All containers communicate over a unified bridge network named lab-network. This allows for service discovery via container names, the script that creates the docker network can be found at `./docker-network.sh`.

2. Shared Scripts:
Useful scripts to run specific group of services, scripts that are useful for several apps or personal scripts.

3. Core:
Database and Message Brokers used by all services.

    - PostgreSQL.
    - MongoDB.
    - Kafka.
    - Redis.
    - RabbitMQ.

4. External Services/Apps:
Managed via Docker Compose to keep the host OS clean.

    - Ollama: Running in a container with a persistent volume (ollama_data).
    - n8n: Workflow automation linked to the local Ollama instance.
    - LabelStudio: Label data for model training.

5. Local Development:
Apps being develop locally that requires access to Core or External services, or maybe scripts.

6. CLI Integration
To maintain a "local" feel while using Docker, the following function is added to ~/.bashrc:

## Requirements

- **OS**: Ubuntu 22.04 LTS or newer (64-bit).
- **Docker Engine**: Version 24.0+ recommended.
- **Docker Compose**: Version 2.20.3+ (Required for `include` directive).
- **User Permissions**: User must be in the `docker` group or have `sudo` access.

## Quick Start

1. **One-Time Setup**:
Run the automation script using `source` to fix DNS, initialize the network, and activate aliases immediately:

```bash
source ./shared/setup-lab.sh
```
