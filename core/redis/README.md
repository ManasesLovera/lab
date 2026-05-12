# Redis - Core Shared Cache

## How to Run
```bash
lab up redis
```

## How to Use
- **Host**: `redis.rpi.local`
- **Port**: `6379`
- **Password**: None (Default config). Use within trusted local network only.

## Configuration
To add a password, update `command` in `docker-compose.yml`:
`redis-server --appendonly yes --requirepass yourpassword`

## Networking
- **Local Access**: Port `6379` is exposed.
- **Production**: Not exposed.
