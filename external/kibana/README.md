# Kibana - Elasticsearch Visualization

Kibana is the window into the Elastic Stack, providing visualization and management for Elasticsearch data.

## How to Run

### Using Lab CLI (Recommended)
```bash
lab up kibana
```

## How to Use
- **URL**: `http://kibana.rpi.local` or `http://192.168.1.8:5601`
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
- **Local Network**: Accessible via `http://kibana.rpi.local` through the lab-proxy (Nginx).
- **IP:Port**: Direct access at `192.168.1.8:5601`.
- **Production**: To expose remotely, add `kibana.mlovera.dev` to the `server_name` directive in `core/proxy/conf.d/proxy.conf` and reload: `docker exec lab-proxy nginx -s reload`.
