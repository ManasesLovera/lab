# Elasticsearch - Search & Analytics Engine

## How to Run
```bash
lab up elasticsearch
```

## How to Use
- **Host**: `elasticsearch.rpi.local` (HTTP via proxy) or `elasticsearch:9200` (Internal) or `192.168.1.8:9200` (Direct)
- **Default User**: `elastic`
- **Default Password**: `admin_password` (defined in `.env`)

## Configuration

### Password Management
Passwords for built-in users are set via the `.env` file (`ELASTIC_PASSWORD`).

### User Management
Elasticsearch uses its Security API. Use `curl`:
```bash
curl -u elastic:admin_password -X POST "http://localhost:9200/_security/user/new_user" -H 'Content-Type: application/json' -d'
{
  "password" : "user_password",
  "roles" : [ "editor", "viewer" ],
  "full_name" : "New User"
}
'
```

## Networking
- **Local Network**: Accessible via `http://elasticsearch.rpi.local` through the lab-proxy (Nginx).
- **Direct Access**: Port `9200` is exposed on the host.
- **Production**: To expose remotely, add `elasticsearch.mlovera.dev` to the `server_name` directive in `core/proxy/conf.d/proxy.conf` and reload: `docker exec lab-proxy nginx -s reload`.
