# ADR 2: Reverting from Traefik to Nginx for Edge Routing

## Status
Accepted (Supersedes ADR 1)

## Context
In ADR 1, we migrated from Nginx to Traefik v3.0 to gain dynamic service discovery via Docker labels. After operating with Traefik in production, several pain points emerged that outweighed the theoretical benefits:

- **Label verbosity**: Each service required 3-4 Traefik labels for a basic route. For a lab with ~5 HTTP services, this added boilerplate without measurable benefit.
- **Debugging opacity**: Traefik's DEBUG logging is excessively verbose, while error diagnostics are vague. Nginx provides clear, concise error logs.
- **Docker socket exposure**: Mounting `/var/run/docker.sock` into the Traefik container grants it Docker API access — an unnecessary attack surface for a static lab environment.
- **Over-engineered for the scale**: Traefik's dynamic discovery shines with dozens of ephemeral services. Our lab has ~5 long-lived services that rarely change. The "reload" cost of Nginx is a one-second command done a few times a year.

## Decision
We are reverting from Traefik back to **Nginx** as the primary edge router, using a modular `conf.d/` approach.

## Architectural Explanation
Nginx operates as a lightweight static reverse proxy with per-service configuration files:

1. **Configuration**: Each service gets a `server` block in a single `proxy.conf` file inside `conf.d/`, included by the main `nginx.conf`.
2. **Service Discovery**: Nginx uses Docker's embedded DNS resolver (`127.0.0.11`) with the `set` directive to dynamically resolve container names at runtime.
3. **Entrypoint**: Nginx binds to host port 80 (with 443 mapped unused).
4. **Network**: All services remain on `lab-network`; Nginx reaches them via their Docker container names.

## Rationale
- **Simplicity**: One nginx container, one config directory, no Docker socket. Configuration is plain text in a single file.
- **Transparency**: Every routing rule is visible in `core/proxy/conf.d/proxy.conf`. No label injection, no dashboard needed.
- **Security**: No Docker socket mount. The proxy has no special privileges.
- **Maintainability**: Adding a service means adding a `server` block and running `nginx -s reload`. No restart needed.

## Consequences
- The `core/traefik` directory is removed.
- `core/proxy` is reinstated as the mandatory edge router.
- `core/cloudflared` ingress is updated to point to `lab-proxy:80`.
- All Traefik labels are stripped from service `docker-compose.yml` files.
- Services without HTTP needs (azurite, postgres, redis, mongo, mssql) remain IP/port-only.
