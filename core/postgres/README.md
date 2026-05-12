# Postgres - Core Shared Database

This is the centralized PostgreSQL instance for the lab.

## How to Run

### Using Lab CLI (Recommended)
```bash
lab up postgres
```

### Using Docker Compose
```bash
cd core/postgres
docker compose up -d
```

## How to Use
- **Host**: `postgres.rpi.local` (Local Network) or `postgres` (Container Network)
- **Port**: `5432`
- **Default Admin User**: `admin`
- **Default Password**: `admin_password` (defined in `.env`)

## Configuration & Customization

### Change Local Domain Prefix
Update your DNS server or local `/etc/hosts` to point your desired prefix (e.g., `postgres.rpi5.local`) to the Raspberry Pi's IP.

### Change Admin Password
1. Update `POSTGRES_PASSWORD` in `core/postgres/.env`.
2. Restart the container: `lab up postgres`.

### Create New Databases & Users
This instance uses a custom initialization script. To automatically create a new database and its owner on startup:
1. Open `core/postgres/docker-compose.yml`.
2. Append the database name to `POSTGRES_MULTIPLE_DATABASES` (comma-separated):
   ```yaml
   - POSTGRES_MULTIPLE_DATABASES=n8n,my_new_app
   ```
3. Restart: `lab up postgres`.
4. The script will create a user `my_new_app` with the password from `POSTGRES_PASSWORD` and grant all privileges on the `my_new_app` database.

### Manual User/Permission Management
Connect via `psql`:
```bash
docker exec -it postgres psql -U admin
```
- **Create User**: `CREATE USER new_user WITH PASSWORD 'secure_password';`
- **Create DB**: `CREATE DATABASE new_db OWNER new_user;`
- **Grant Permissions**: `GRANT ALL PRIVILEGES ON DATABASE new_db TO new_user;`

## Networking
- **Local Access**: Accessible via `postgres.rpi.local:5432` because port `5432` is mapped to the host.
- **Production**: Not exposed via Traefik by default for security. Use a VPN or SSH tunnel to access remotely.
