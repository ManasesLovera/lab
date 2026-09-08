# Planboard

Curriculum planning web application for WELEEC (Bun/Next.js), deployed from `ghcr.io/weleec/planboard` and backed by the lab's Postgres instance.

## How to Run

1. Initialize the lab environment (if not already done):
   ```bash
   source ./shared/setup-lab.sh
   ```
2. Ensure Postgres is running and the `weleec_planboard` database/role exist (already provisioned):
   ```bash
   lab up postgres
   ```
3. Pull access to the private image requires `read:packages` on the ghcr login:
   ```bash
   gh auth refresh -h github.com -s read:packages
   gh auth token | docker login ghcr.io -u <github-user> --password-stdin
   ```
4. Copy `example.env` to `.env` and set the real `PB_DATABASE_URL`:
   ```bash
   cp example.env .env
   ```
5. Start the service using the lab CLI:
   ```bash
   lab up planboard
   ```

## How to Use

- **Web UI**: Access the UI in your browser at `http://planboard.rpi.local` or `https://planboard.mlovera.dev`.
- **Health check**: `GET /api/health` runs a real database query (unauthenticated; reports `driver` and status only).
- **Logs**: `lab logs planboard`

## Configuration

The application is configured using environment variables in the local `.env` file:

- `PB_DB`: Database driver — `postgres` (this deployment) or `sqlite`.
- `PB_DATABASE_URL`: Postgres connection URL. `#` in the password MUST stay percent-encoded as `%23`.
- `PB_LOGIN_MAX_ATTEMPTS`: Optional override of the login throttle (leave unset in production).

## Networking

- **Internal Discovery**: The service is reachable on `lab-network` by its container name `planboard` on port `3000`. There is deliberately **no host port** published — it is only reachable through the proxy.
- **Proxy Ingress**: Configured in Nginx (`core/proxy/conf.d/proxy.conf`) to map domains `planboard.rpi.local` and `planboard.mlovera.dev` to `planboard:3000`. `X-Forwarded-For` is required by the app's login throttle.
- **Remote Ingress**: Routed via `lab-cloudflared` to Nginx on port 80 (public hostname `planboard.mlovera.dev` added in the Cloudflare tunnel dashboard).