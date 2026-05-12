# Traefik - Dynamic Edge Proxy

Traefik is the entry point for all HTTP traffic in the lab.

## How to Run
```bash
lab up traefik
```

## How to Use
- **Dashboard**: `http://localhost:8080` (Insecure mode enabled).
- **Service Routing**: Traefik automatically detects services in the `lab-network` that have `traefik.enable=true` labels.

## Networking
- **Local**: Binds to port `80` on the host.
- **Domain Mapping**: Resolves domains like `*.rpi.local` to specific containers based on their labels.

## Customizing Local Prefix
To change the prefix for the entire lab (e.g., to `rpi5.local`), you don't change Traefik itself. Instead, update the `Host(...)` rule labels in every individual service's `docker-compose.yml`.
