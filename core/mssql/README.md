# MSSQL - Microsoft SQL Server 2022

## How to Run

### Using Lab CLI (Recommended)
```bash
lab up mssql
```

## How to Use
- **Host**: `mssql.rpi.local`
- **Port**: `1433`
- **Default User**: `sa`
- **Default Password**: `StrongPassword123!` (defined in `.env`)

## Configuration

### Change SA Password
1. Update `MSSQL_SA_PASSWORD` in `core/mssql/.env`.
2. **Important**: Password must meet SQL Server complexity requirements.
3. Restart: `lab up mssql`.

### User Management
Connect using `sqlcmd` or Azure Data Studio:
```bash
docker exec -it mssql /opt/mssql-tools/bin/sqlcmd -S localhost -U sa -P 'StrongPassword123!'
```

**Example: Create Login and User**
```sql
CREATE LOGIN [new_app_user] WITH PASSWORD = 'AnotherStrongPassword!';
GO
CREATE DATABASE [app_db];
GO
USE [app_db];
GO
CREATE USER [new_app_user] FOR LOGIN [new_app_user];
GO
EXEC sp_addrolemember 'db_owner', 'new_app_user';
GO
```

## Networking
- **Local Access**: Accessible via `mssql.rpi.local:1433`.
- **Production**: Not exposed via Traefik.
