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

## Staging: planboard-dev

`planboard-dev.mlovera.dev` runs the same image from the same Dockerfile, one tag apart, so an
unreleased build can be tried before a release tag exists.

| | Production | Staging |
| --- | --- | --- |
| Hostname | `planboard.mlovera.dev` | `planboard-dev.mlovera.dev` |
| Container | `planboard` | `planboard-dev` |
| Image tag | `ghcr.io/weleec/planboard:latest` | `ghcr.io/weleec/planboard:dev` |
| Moved by | a release tag `vX.Y.Z` | a pre-release tag `vX.Y.Z-dev.N` |
| Database | lab Postgres | SQLite on the `planboard-dev-data` volume, always |

Cutting a staging build is done from the app repo (`git tag -a v1.2.0-dev.1 && git push origin
v1.2.0-dev.1`); CI publishes it and moves `dev` only. Then here:

```bash
docker compose pull planboard-dev && docker compose up -d planboard-dev
lab logs planboard          # both services; expect the [migrate] applied … lines
```

Name the service. A bare `lab up planboard` or `docker compose up -d` restarts production too,
which is not what a staging test should cost.

The volume starts empty, so the first boot migrates a fresh database and serves a login page
with no users in it. Seed it with a scratch admin rather than a copy of the school's data.

Two things that are deliberate, not gaps:

- **Staging never gets `PB_DATABASE_URL`.** `PB_DB: sqlite` is in the compose `environment:`
  block, which overrides `env_file:`, and `.env.dev` is documented as holding no database
  credential. That is the whole reason a wrong migration in an unreleased build cannot reach
  the school's data.
- **A green staging run says nothing about Postgres.** It exercises SQLite, the driver CI
  already covers. The hand-verification of the Postgres path before a release does not go away
  — if anything staging is the tempting reason to skip it.

## How to Use

- **Web UI**: Access the UI in your browser at `http://planboard.rpi.local` or `https://planboard.mlovera.dev`. Staging answers at `planboard-dev.rpi.local` / `https://planboard-dev.mlovera.dev`.
- **Health check**: `GET /api/health` runs a real database query (unauthenticated; reports `driver` and status only).
- **Logs**: `lab logs planboard` (both services), or `docker compose logs -f planboard-dev` for staging alone

## Configuration

The application is configured using environment variables in the local `.env` file:

- `PB_DB`: Database driver — `postgres` (this deployment) or `sqlite`.
- `PB_DATABASE_URL`: Postgres connection URL. `#` in the password MUST stay percent-encoded as `%23`.
- `PB_LOGIN_MAX_ATTEMPTS`: Optional override of the login throttle (leave unset in production).

## Networking

- **Internal Discovery**: The service is reachable on `lab-network` by its container name `planboard` on port `3000`. There is deliberately **no host port** published — it is only reachable through the proxy.
- **Proxy Ingress**: Configured in Nginx (`core/proxy/conf.d/proxy.conf`) to map domains `planboard.rpi.local` and `planboard.mlovera.dev` to `planboard:3000`. `X-Forwarded-For` is required by the app's login throttle.
- **Remote Ingress**: Routed via `lab-cloudflared` to Nginx on port 80 (public hostname `planboard.mlovera.dev` added in the Cloudflare tunnel dashboard).