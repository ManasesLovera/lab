# Credentials & User Management

This document covers default credentials for all lab services and how to add new users, databases, and permissions for each storage backend.

---

## Table of Contents

- [PostgreSQL](#postgresql)
- [MongoDB](#mongodb)
- [MSSQL (SQL Server)](#mssql-sql-server)
- [Elasticsearch](#elasticsearch)
- [Redis](#redis)
- [Azurite (Azure Storage Emulator)](#azurite-azure-storage-emulator)

---

## PostgreSQL

### Default Credentials

| Field | Value |
|---|---|
| Host (internal) | `postgres:5432` |
| Host (external) | `192.168.1.8:5432` |
| Admin user | `admin` |
| Admin password | From `core/postgres/.env` — `POSTGRES_PASSWORD` |
| Default databases | `n8n` (with user `n8n`) |

### Connecting

```bash
# Via docker exec
docker exec -it postgres psql -U admin

# Via psql from another container on lab-network
psql -h postgres -U admin -d postgres

# Via external client
psql -h 192.168.1.8 -U admin -d postgres
```

### Add a New Database & User

Postgres uses an auto-initialization script (`core/postgres/scripts/init-db.sh`) that reads `POSTGRES_MULTIPLE_DATABASES` from the `.env`. Append the database name:

```bash
# In core/postgres/.env:
POSTGRES_MULTIPLE_DATABASES=n8n,my_new_app
```

This will:
1. Create a user `my_new_app` with password from `MY_NEW_APP_PASSWORD` (falls back to `POSTGRES_PASSWORD`)
2. Create database `my_new_app` owned by that user
3. Grant all privileges on the `public` schema

Then restart:
```bash
lab down postgres && lab up postgres
```

> **Note**: For existing volumes, the init script only runs once. To add databases to a running instance, connect manually:

```sql
CREATE USER my_new_app WITH PASSWORD 'secure_password';
CREATE DATABASE my_new_app OWNER my_new_app;
\c my_new_app
GRANT ALL ON SCHEMA public TO my_new_app;
```

### Manual User & Permission Management

```sql
-- Create user
CREATE USER app_user WITH PASSWORD 'strong_password';

-- Create database
CREATE DATABASE app_db OWNER app_user;

-- Grant schema permissions
\c app_db
GRANT ALL ON SCHEMA public TO app_user;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO app_user;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO app_user;

-- Read-only user
CREATE USER readonly_user WITH PASSWORD 'readonly_pass';
GRANT CONNECT ON DATABASE app_db TO readonly_user;
\c app_db
GRANT USAGE ON SCHEMA public TO readonly_user;
GRANT SELECT ON ALL TABLES IN SCHEMA public TO readonly_user;

-- Remove user
DROP USER IF EXISTS app_user;
```

---

## MongoDB

### Default Credentials

| Field | Value |
|---|---|
| Host (internal) | `mongo:27017` |
| Host (external) | `192.168.1.8:27017` |
| Admin user | `admin` |
| Admin password | `admin_password` (from `core/mongo/.env`) |
| Auth database | `admin` |

### Connecting

```bash
# Via docker exec
docker exec -it mongo mongosh -u admin -p admin_password --authenticationDatabase admin

# Via mongosh from another container on lab-network
mongosh mongodb://admin:admin_password@mongo:27017/admin

# Via external client
mongosh mongodb://admin:admin_password@192.168.1.8:27017/admin

# Connection string for applications
mongodb://admin:admin_password@mongo:27017/admin
```

### Add a New Database & User

```javascript
// Connect and switch to the new database
use my_app_db

// Create user with readWrite role on this database
db.createUser({
  user: "app_user",
  pwd: "secure_password",
  roles: [{ role: "readWrite", db: "my_app_db" }]
})

// Or create user from admin database with access to multiple databases
use admin
db.createUser({
  user: "multi_user",
  pwd: "secure_password",
  roles: [
    { role: "readWrite", db: "my_app_db" },
    { role: "read", db: "analytics" }
  ]
})
```

### Common Roles

| Role | Description |
|---|---|
| `read` | Read-only on specified database |
| `readWrite` | Read and write on specified database |
| `dbAdmin` | Administrative operations on a database |
| `userAdmin` | Manage users on a database |
| `dbOwner` | Combination of readWrite, dbAdmin, and userAdmin |
| `root` | Superuser across all databases |

### User Management

```javascript
// List users
use admin
db.getUsers()

// Update user password
db.changeUserPassword("app_user", "new_password")

// Grant additional roles
db.grantRolesToUser("app_user", [{ role: "dbAdmin", db: "my_app_db" }])

// Revoke roles
db.revokeRolesFromUser("app_user", [{ role: "readWrite", db: "other_db" }])

// Drop user
db.dropUser("app_user")
```

---

## MSSQL (SQL Server)

### Default Credentials

| Field | Value |
|---|---|
| Host (internal) | `mssql:1433` |
| Host (external) | `192.168.1.8:1433` |
| Admin user | `sa` |
| Admin password | `StrongPassword123!` (from `core/mssql/.env`) |

> **Note**: MSSQL runs Azure SQL Edge (not full SQL Server) due to ARM64 architecture on Raspberry Pi.

### Connecting

```bash
# Via docker exec
docker exec -it mssql /opt/mssql-tools/bin/sqlcmd -S localhost -U sa -P 'StrongPassword123!'

# Via sqlcmd from another container
sqlcmd -S mssql -U sa -P 'StrongPassword123!'

# Connection string for applications
Server=mssql,1433;User Id=sa;Password=StrongPassword123!;
```

### Add a New Login, Database & User

```sql
-- Create login (server-level principal)
CREATE LOGIN [app_user] WITH PASSWORD = 'AnotherStrongPassword!';

-- Create database
CREATE DATABASE [app_db];

-- Switch to the new database and create user
USE [app_db];
CREATE USER [app_user] FOR LOGIN [app_user];

-- Grant db_owner (full control)
EXEC sp_addrolemember 'db_owner', 'app_user';

-- Or grant read/write only
-- EXEC sp_addrolemember 'db_datareader', 'app_user';
-- EXEC sp_addrolemember 'db_datawriter', 'app_user';
```

### Common Database Roles

| Role | Description |
|---|---|
| `db_owner` | Full control over the database |
| `db_datareader` | Read all data |
| `db_datawriter` | Insert/update/delete all data |
| `db_ddladmin` | Create/modify/drop objects |
| `db_securityadmin` | Manage permissions |

### User Management

```sql
-- List logins
SELECT name, type_desc FROM sys.server_principals WHERE type IN ('S', 'U');

-- List database users
USE [app_db];
SELECT name, type_desc FROM sys.database_principals WHERE type IN ('S', 'U');

-- Grant specific permissions
USE [app_db];
GRANT SELECT, INSERT, UPDATE, DELETE ON [dbo].[users] TO [app_user];

-- Remove user from database
USE [app_db];
DROP USER [app_user];

-- Drop login
DROP LOGIN [app_user];
```

---

## Elasticsearch

### Default Credentials

| Field | Value |
|---|---|
| Host (internal) | `elasticsearch:9200` |
| Host (external) | `192.168.1.8:9200` or `http://elasticsearch.rpi.local` |
| Admin user | `elastic` |
| Admin password | `admin_password` (from `core/elasticsearch/.env`) |

### Connecting

```bash
# Via curl
curl -u elastic:admin_password http://localhost:9200

# From another container
curl -u elastic:admin_password http://elasticsearch:9200
```

### Add a New User

```bash
curl -u elastic:admin_password -X POST "http://localhost:9200/_security/user/new_user" -H 'Content-Type: application/json' -d'
{
  "password": "user_password",
  "roles": ["editor", "viewer"],
  "full_name": "New User",
  "email": "user@example.com"
}
'
```

### Add a New Role

```bash
curl -u elastic:admin_password -X POST "http://localhost:9200/_security/role/my_custom_role" -H 'Content-Type: application/json' -d'
{
  "indices": [
    {
      "names": ["my-index-*"],
      "privileges": ["read", "write"],
      "field_security": { "grant": ["*"], "except": ["ssn"] }
    }
  ],
  "cluster": ["monitor"]
}
'
```

### Common Roles

| Role | Description |
|---|---|
| `superuser` | Full access to all indices and cluster operations |
| `editor` | Read/write on all indices |
| `viewer` | Read-only on all indices |

### User Management

```bash
# List users
curl -u elastic:admin_password "http://localhost:9200/_security/user"

# Change password
curl -u elastic:admin_password -X POST "http://localhost:9200/_security/user/jane/_password" -H 'Content-Type: application/json' -d'{"password": "new_password"}'

# Disable user
curl -u elastic:admin_password -X PUT "http://localhost:9200/_security/user/jane/_disable"

# Delete user
curl -u elastic:admin_password -X DELETE "http://localhost:9200/_security/user/jane"
```

---

## Redis

### Default Credentials

| Field | Value |
|---|---|
| Host (internal) | `redis:6379` |
| Host (external) | `192.168.1.8:6379` |
| Password | *(none — no auth configured)* |

> **Warning**: Redis has no password by default. It is only accessible to containers on `lab-network` and the local host. Do not expose port 6379 to the internet without a password.

### Connecting

```bash
# Via docker exec
docker exec -it redis redis-cli

# From another container
redis-cli -h redis

# Via external client
redis-cli -h 192.168.1.8
```

### Enable Password Authentication

Edit `core/redis/docker-compose.yml`:

```yaml
command: redis-server --appendonly yes --requirepass your_secure_password
```

Then restart:
```bash
lab down redis && lab up redis
```

After enabling, include the password in all connections:
```bash
redis-cli -h redis -a your_secure_password
```

### Basic ACL Management (Redis 6+)

Redis 6+ supports ACL rules. Enable via `redis-cli`:

```bash
# Create a read-only user
ACL SETUSER readonly_user on >readonly_pass ~* +@read

# Create a user restricted to specific keys
ACL SETUSER app_user on >app_pass ~cache:* +@all -@dangerous

# List all users
ACL LIST

# Remove user
ACL DELUSER app_user
```

---

## Azurite (Azure Storage Emulator)

### Default Credentials

| Field | Value |
|---|---|
| Host (internal) | `azurite:10000` (blob), `azurite:10001` (queue), `azurite:10002` (table) |
| Host (external) | `192.168.1.8:10000-10002` |
| Account name | `devstoreaccount1` |
| Account key | `Eby8vdM02xNOcqFlqRztJFAkZuzDjt38ADxW7Ef9GEbeG98tkr8WvH06lX6GSEuG8DXvdVH2CTT9S50u30kvw==` |

### Connecting

```bash
# Azure CLI
az storage blob list --account-name devstoreaccount1 \
  --account-key "Eby8vdM02xNOcqFlqRztJFAkZuzDjt38ADxW7Ef9GEbeG98tkr8WvH06lX6GSEuG8DXvdVH2CTT9S50u30kvw==" \
  --blob-endpoint http://192.168.1.8:10000/devstoreaccount1 \
  --container-name mycontainer

# Connection strings
# Blob:
DefaultEndpointsProtocol=http;AccountName=devstoreaccount1;AccountKey=Eby8vdM02xNOcqFlqRztJFAkZuzDjt38ADxW7Ef9GEbeG98tkr8WvH06lX6GSEuG8DXvdVH2CTT9S50u30kvw==;BlobEndpoint=http://192.168.1.8:10000/devstoreaccount1;

# Queue:
DefaultEndpointsProtocol=http;AccountName=devstoreaccount1;AccountKey=Eby8vdM02xNOcqFlqRztJFAkZuzDjt38ADxW7Ef9GEbeG98tkr8WvH06lX6GSEuG8DXvdVH2CTT9S50u30kvw==;QueueEndpoint=http://192.168.1.8:10001/devstoreaccount1;

# Table:
DefaultEndpointsProtocol=http;AccountName=devstoreaccount1;AccountKey=Eby8vdM02xNOcqFlqRztJFAkZuzDjt38ADxW7Ef9GEbeG98tkr8WvH06lX6GSEuG8DXvdVH2CTT9S50u30kvw==;TableEndpoint=http://192.168.1.8:10002/devstoreaccount1;

# All-in-one:
DefaultEndpointsProtocol=http;AccountName=devstoreaccount1;AccountKey=Eby8vdM02xNOcqFlqRztJFAkZuzDjt38ADxW7Ef9GEbeG98tkr8WvH06lX6GSEuG8DXvdVH2CTT9S50u30kvw==;BlobEndpoint=http://192.168.1.8:10000/devstoreaccount1;QueueEndpoint=http://192.168.1.8:10001/devstoreaccount1;TableEndpoint=http://192.168.1.8:10002/devstoreaccount1;
```

### Add Custom Storage Accounts

Define additional accounts in `core/azurite/.env`:

```env
AZURITE_ACCOUNDS=account1:key1
```

The format is `accountName:base64key`. Azurite docs: https://github.com/Azure/Azurite

### SDK Usage Examples

```python
# Python (azure-storage-blob)
from azure.storage.blob import BlobServiceClient

conn_str = "DefaultEndpointsProtocol=http;AccountName=devstoreaccount1;AccountKey=Eby8vdM02x...;BlobEndpoint=http://192.168.1.8:10000/devstoreaccount1;"
client = BlobServiceClient.from_connection_string(conn_str)
client.create_container("mycontainer")
```

```bash
# Using Azure Storage Explorer
# Add connection: Storage account → Connection string → paste the all-in-one string above
```

---

## Quick Reference

| Service | Container Name | Port | Auth Required |
|---|---|---|---|
| Postgres | `postgres` | 5432 | Yes (user/pass) |
| MongoDB | `mongo` | 27017 | Yes (user/pass) |
| MSSQL | `mssql` | 1433 | Yes (user/pass) |
| Elasticsearch | `elasticsearch` | 9200 | Yes (basic auth) |
| Redis | `redis` | 6379 | No (default) |
| Azurite | `azurite` | 10000-10002 | Yes (account key) |
| n8n | `n8n_local` | 5678 | Self-registered |

