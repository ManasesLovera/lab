# Cloudflared - Secure Remote Access

Provides a secure tunnel from Cloudflare to your local lab without opening router ports.

## How to Run
```bash
lab up cloudflared
```

## Configuration
- **External Ingress**: Managed in `core/cloudflared/config.yml`.
- **Wildcard Routing**: All `*.mlovera.dev` traffic is forwarded to `lab-traefik:80`.
- **SSH**: `rpi.mlovera.dev` is routed to the host machine's SSH port.

## How to Add New Remote Domains
1. Add a label for the domain in the target service's `docker-compose.yml`.
2. Ensure the domain is registered in your Cloudflare Dashboard and pointing to the tunnel.
3. Traefik will automatically handle the routing once the traffic reaches the tunnel.
