# Elasticsearch - Search & Analytics Engine

## How to Run
```bash
lab up elasticsearch
```

## How to Use
- **Host**: `elasticsearch.rpi.local` (HTTP) or `elasticsearch:9200` (Internal)
- **Default User**: `elastic`
- **Default Password**: `admin_password` (defined in `.env`)

## Configuration

### Password Management
Passwords for built-in users are set via the `.env` file (`ELASTIC_PASSWORD`).

### User Management
Elasticsearch uses its Security API. Use `curl` or Kibana (if installed):
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
- **Local Network**: Accessible via `http://elasticsearch.rpi.local` through Traefik.
- **Production**: To expose remotely, update `core/elasticsearch/docker-compose.yml` labels to include `Host('elasticsearch.mlovera.dev')`.
- **Change Prefix**: Change the Host rule in `docker-compose.yml` from `elasticsearch.rpi.local` to `elasticsearch.rpi5.local`.
