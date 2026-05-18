# Cloudflared - Secure Remote Access

Provides a secure tunnel from Cloudflare to your local lab without opening router ports.

## How to Run
```bash
lab up cloudflared
```

## Configuration
- **External Ingress**: Managed in `core/cloudflared/config.yml`.
- **Wildcard Routing**: All `*.mlovera.dev` traffic is forwarded to `lab-proxy:80`.
- **SSH**: `rpi.mlovera.dev` is routed to the host machine's SSH port.

## How to Add New Remote Domains
1. Add the production domain (e.g., `myapp.mlovera.dev`) to the `server_name` directive in `core/proxy/conf.d/proxy.conf`.
2. Reload the proxy: `docker exec lab-proxy nginx -s reload`.
3. Ensure the domain is registered in your Cloudflare Dashboard and pointing to the tunnel.
4. The tunnel will automatically route traffic to the proxy once configured.
