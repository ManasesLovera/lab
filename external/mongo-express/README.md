# Mongo Express - MongoDB Visualization

Mongo Express is a web-based MongoDB administrative interface.

## How to Run

### Using Lab CLI (Recommended)
```bash
lab up mongo-express
```

## How to Use
- **URL**: `http://mongo-express.rpi.local`
- **Default Auth (UI)**:
  - **Username**: `admin`
  - **Password**: `pass` (defined in `.env` as `ME_CONFIG_BASICAUTH_PASSWORD`)

## Configuration

### Connecting to MongoDB
It connects to the `mongo` core service via the `lab-network`.
- **Server**: `mongo`
- **Credentials**: Uses the root admin credentials defined in `core/mongo/.env`.

## Networking
- **Local Network Only**: Accessible via `http://mongo-express.rpi.local` through the lab-proxy (Nginx).
- **Production**: By default, this is **not** exposed to the internet. To expose it, add `mongo-express.mlovera.dev` to the `server_name` directive in `core/proxy/conf.d/proxy.conf` and reload: `docker exec lab-proxy nginx -s reload`.
