# Kibana - Elasticsearch Visualization

Kibana is the window into the Elastic Stack, providing visualization and management for Elasticsearch data.

## How to Run

### Using Lab CLI (Recommended)
```bash
lab up kibana
```

## How to Use
- **URL**: `http://kibana.rpi.local`
- **Default User**: `elastic` (Use the admin user to log in via the UI)
- **Password**: The `ELASTIC_PASSWORD` defined in `core/elasticsearch/.env`.

## Configuration

### Connecting to Elasticsearch
Kibana is configured via `.env` to connect to the `elasticsearch` container within the `lab-network`.
The `kibana_system` user is used for background communication.

### Initial Setup
1. Start Elasticsearch first: `lab up elasticsearch`.
2. Start Kibana: `lab up kibana`.
3. Open `http://kibana.rpi.local` and log in with the `elastic` superuser.

## Networking
- **Local Network**: Accessible via `http://kibana.rpi.local` through Traefik.
- **Production**: To expose remotely, add the `Host('kibana.mlovera.dev')` label to `external/kibana/docker-compose.yml`.
- **Change Prefix**: Update the `Host` rule in `docker-compose.yml` (e.g., to `kibana.rpi5.local`).
