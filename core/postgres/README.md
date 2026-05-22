# Postgres - Core Shared Database

This is the centralized PostgreSQL instance for the lab.

## Quick Start

### Using Lab CLI (Recommended)
```bash
lab up postgres
```

### Credentials & Connection
- **Host**: `192.168.1.8` (or your Pi's IP) / `postgres` (Container Network)
- **Port**: `5432`
- **Admin User**: `admin`
- **Password**: Defined in `.env` (Use `secrets get postgres_admin_password` to check)

---

## User & Database Management

### 1. Automated Provisioning (Startup)
The lab uses a custom initialization script (`init-db.sh`). To add a service database on startup:
1. Update `POSTGRES_MULTIPLE_DATABASES` in `core/postgres/.env` (comma-separated):
   ```env
   POSTGRES_MULTIPLE_DATABASES=n8n,my_new_app
   ```
2. (Optional) Set a specific password: `my_new_app_PASSWORD=some_secret`.
3. Run `lab up postgres`.

### 2. Management via Lab Tools (Runtime)
You can create users and databases without restarting the service using the `secrets` tool:

```bash
# Usage: secrets db create-user postgres <dbname> <username> <password>
secrets db create-user postgres my_app my_user my_secure_pass
```
This command automatically:
- Creates the user and database.
- Grants all privileges to the user.
- Configures the public schema permissions.
- Logs the action in the local audit history (`secrets history`).

### 3. Credential Synchronization
If you change passwords in `.env` after the database is already initialized, the internal DB password will NOT update automatically. You must sync them manually:

```bash
# Update admin password inside the DB
docker exec -it postgres psql -U admin -c "ALTER USER admin WITH PASSWORD 'new_password';"

# Then update the secrets manager
shared/import-postgres-creds.sh
```

---

## Operations & Troubleshooting

### Check Stored Credentials
The lab maintains a local secrets manager for easy retrieval:
```bash
secrets list                        # Show all discovered .env credentials
secrets get postgres_admin_password # Get specific admin password
secrets get postgres_n8n_password   # Get specific app password
secrets history                     # See history of users created via 'secrets db'
```

### Manual Console Access
```bash
docker exec -it postgres psql -U admin
```

### Common Commands (Inside psql)
- **List Databases**: `\l`
- **List Users**: `\du`
- **Switch Database**: `\c <dbname>`
- **Reset Password**: `ALTER USER <username> WITH PASSWORD '<password>';`

### Maintenance Scripts
- `shared/import-postgres-creds.sh`: Scans the container and imports active database credentials into the `secrets` manager. Use this if you lose track of passwords or after manual changes.

---

## Networking & Security
- **Local Access**: Accessible via port `5432` on the host.
- **Authentication**: Uses `scram-sha-256` for host connections.
- **Production**: Not exposed via Traefik/Cloudflare by default.
