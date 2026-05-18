# MongoDB - Core Shared NoSQL Database

## How to Run

### Using Lab CLI (Recommended)
```bash
lab up mongo
```

### Using Docker Compose
```bash
cd core/mongo
docker compose up -d
```

## How to Use
- **Host**: `mongo` (Container Network) or `192.168.1.8:27017` (Direct)
- **Port**: `27017`
- **Default Root User**: `admin`
- **Default Password**: `admin_password` (defined in `.env`)

## Configuration & Customization

### Change Root Password
1. Update `MONGO_INITDB_ROOT_PASSWORD` in `core/mongo/.env`.
2. Restart: `lab up mongo`.

### Create New Users & Databases
Connect to the mongo shell:
```bash
docker exec -it mongo mongosh -u admin -p admin_password --authenticationDatabase admin
```

**Example: Create a user for a specific database**
```javascript
use my_new_db
db.createUser({
  user: "db_user",
  pwd: "password123",
  roles: [{ role: "readWrite", db: "my_new_db" }]
})
```

## Networking
- **Direct Access**: Accessible via `192.168.1.8:27017`.
- **Web UI**: Use mongo-express at `http://mongo-express.rpi.local`.
- **Production**: Not exposed to the internet. Access via local network or VPN.
