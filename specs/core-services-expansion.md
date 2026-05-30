# Specification: Core Shared Infrastructure Expansion

## 1. Objective
To provide a robust, unified set of development databases and storage emulators that are accessible both internally (within the Docker network) and externally (to other devices on the local network). This prevents "container bloat" by allowing multiple projects to share a single, well-configured instance of each data technology.

## 2. Included Services

### 2.2 MongoDB
- **Purpose**: Document-based NoSQL storage.
- **Implementation**: `mongo:latest`.
- **Access**: Direct TCP mapping on port 27017.

### 2.3 Elasticsearch
- **Purpose**: Full-text search and log analytics.
- **Implementation**: `docker.elastic.co/elasticsearch/elasticsearch:8.12.0`.
- **Integration**: Traefik-enabled for HTTP access via `elasticsearch.rpi.local`.

## 3. Standardized Requirements

### 3.1 Credential Isolation
Every core service MUST utilize a local `.env` file for secrets (passwords, API keys). An `example.env` must be provided for every project to ensure reproducibility.

### 3.2 Networking Pattern
- **HTTP Services**: Must include Traefik labels for `.rpi.local` domain resolution.
- **TCP Services (DBs)**: Must map their standard ports to the host to allow connection from non-Dockerized apps on the same Wi-Fi.
- **Container Network**: All services must join the `lab-network` for cross-service communication (e.g., n8n connecting to Postgres).

### 3.3 Persistence
All data-heavy services must map a local `./data` volume to ensure data survives container restarts and upgrades.
