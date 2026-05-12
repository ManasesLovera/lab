# ADR 1: Transitioning from Nginx to Traefik for Dynamic Routing

## Status
Accepted

## Context
The initial lab architecture utilized a centralized Nginx proxy (`lab-proxy`) to handle routing for all services. While functional, this approach had several drawbacks:
- **Centralization**: All routing logic was stored in a single `proxy.conf.template` file. Adding a new service required modifying this core file.
- **Manual Configuration**: Environment variables for all subdomains had to be injected into the proxy container.
- **Service Discovery**: While Nginx used container names, the configuration was static and didn't react dynamically to the lifecycle of other containers.

## Decision
We have decided to replace Nginx with **Traefik Proxy v3.0** as the primary edge router for the lab.

## Architectural Explanation
Traefik operates as a dynamic reverse proxy that integrates directly with the Docker socket. The routing logic is now **decentralized**:

1.  **Provider**: Traefik listens to the Docker API for container events.
2.  **Labels**: Each service (e.g., n8n, Elasticsearch) defines its own routing rules (domains, ports, middleware) using Docker labels in its `docker-compose.yml`.
3.  **Entrypoints**: Traefik is configured with a `web` entrypoint on port 80.
4.  **Network**: All services must still reside on the `lab-network` to allow Traefik to reach them via internal IPs.

## Rationale
- **Scalability**: Adding a new project is now "plug-and-play." Adding Traefik labels to a new `docker-compose.yml` automatically registers the service without restarting the proxy.
- **Isolation**: Changes to one service's routing do not risk breaking the configuration of others.
- **Observability**: Traefik provides a real-time dashboard (`:8080`) to visualize active routers and services.
- **Simplicity**: Removes the need for complex Nginx templates and centralized environment variable management for routing.

## Consequences
- The `core/proxy` directory has been removed.
- `core/traefik` is now a mandatory "always-on" service.
- `core/cloudflared` ingress now points to `lab-traefik:80` for wildcard routing.
- Developers must follow the `docs/networking.md` guide to apply the correct labels to new services.
