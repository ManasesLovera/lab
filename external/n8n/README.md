# n8n - Workflow Automation

n8n is the primary automation tool for this lab, integrated with the shared Postgres instance and Python runner.

## How to Run

### Using Lab CLI (Recommended)
```bash
lab up n8n
```

### Using Docker Compose
```bash
cd external/n8n
docker compose up -d
```

## Initialization & Default User
1. Access `http://n8n.rpi.local` (or `http://192.168.1.8:5678`).
2. On first run, n8n will prompt you to create an owner account.
3. To change credentials later, use the n8n UI (Settings > Users).

## Using Shared Postgres
n8n is pre-configured to use the `postgres` core service.
- **Database**: `n8n`
- **User**: `n8n`
- **Password**: Managed in `external/n8n/.env` (`DB_POSTGRESDB_PASSWORD`).
- **Connection**: It uses the container name `postgres` as the host within the `lab-network`.

## Python Availability
This setup includes a dedicated `python-runner` service.
- **How to use**: In any n8n node that supports code, select **Python** as the language.
- **Environment**: The runner has common libraries installed. To add more, you would need to rebuild the `python-runner` image with additional `pip install` commands.

## Networking & Access Control

### 1. Local Network (*.rpi.local)
Access n8n via `http://n8n.rpi.local`. Routing is handled by the lab-proxy (Nginx) via `core/proxy/conf.d/proxy.conf`.

### 2. Production Access (*.mlovera.dev)
n8n is exposed to the internet via the Cloudflare Tunnel at `https://n8n.mlovera.dev`. Traffic flows: Cloudflare Tunnel → `lab-proxy:80` → Nginx matches `n8n.mlovera.dev` server_name → `n8n_local:5678`.
- **Security**: Ensure you have set a strong password in the n8n UI since it is public.

### 3. Localhost Only
Remove the `ports` mapping from `docker-compose.yml` to prevent any external access.

## Accessing Files
Your shared lab scripts and data are available inside the n8n container at `/home/node/shared`. This is a read-only mount of the project's `shared/` directory.
