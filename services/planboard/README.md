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
| Moved by | a release tag `vX.Y.Z`, pushed by a human | a pre-release tag `vX.Y.Z-dev.N`, cut and deployed automatically on every merge to `development` |
| Database | lab Postgres | SQLite on the `planboard-dev-data` volume, always |

**Staging deploys itself.** Every merge to `development` in the app repo that changes more
than docs gets the next `vX.Y.Z-dev.N` tag, publishes `:dev`, and is deployed here by the
GitHub Actions runner described below. Nobody cuts dev tags or pulls staging by hand any more.
If the runner is down, deploy by hand with the same script it runs:

```bash
~/dev/planboard/scripts/deploy-staging.sh <commit-sha>
lab logs planboard          # both services; expect the [migrate] applied … lines
```

If you ever do run compose by hand, name the services. A bare `lab up planboard` or
`docker compose up -d` restarts production too, which is not what a staging test should cost.

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

## The GitHub Actions runner

Staging deploys itself, and the thing that does it runs on this Pi: a GitHub Actions
self-hosted runner registered on `weleec/planboard`.

| | |
| --- | --- |
| Directory | `~/runners/planboard` — **outside this repository, and outside every other** |
| Runner name | `pi5-planboard`, labels `self-hosted` and `pi` |
| Runs as | `mlovera`, systemd unit `actions.runner.weleec-planboard.pi5-planboard.service` |
| Takes | `Deploy staging` only, on every merge to `development` that changes more than docs |
| Touches | `planboard-dev` and `planboard-dev-mcp` in this directory, and `~/backups` |

It keeps an outbound connection open to GitHub and receives the job over it; nothing connects
in, and neither nginx nor the tunnel is involved. The job runs `scripts/deploy-staging.sh`
from the application repository: back up staging's SQLite into `~/backups`, pull and recreate
both staging services, check the revision label equals the commit, wait for healthy. It never
names a production service.

**Why it is not in this repository.** This repository is public. The runner's directory holds
the credentials that let it take jobs (`.credentials`, `.runner`) and, under `_work/`,
checkouts of the application's private source. `.gitignore` covers neither, so keeping it
here would be one `git add -A` away from publishing both. It is also a self-updating binary,
not source, and it is rebuilt with a fresh registration rather than restored — nothing in it is
worth versioning.

**Why it runs as `mlovera`.** `docker compose` reads every service's `env_file` when it loads
this project, so even `docker compose pull planboard-dev` fails without a readable `.env`; the
image is private and pulled with this user's ghcr login; and `docker` group membership is root
here in all but name, so a separate user would isolate nothing.

**One runner per repository.** A runner serves one repository or one organisation, and a
personal account has no account-wide runners, so it cannot also serve this repository or any
other `ManasesLovera/*` one. Another repository that deploys on this Pi gets its own runner in
`~/runners/<repo>/`.

**Rebuilding it** (new SD card, lost registration) — the prerequisites above (ghcr login,
`.env`, `.env.dev`) plus `mkdir -p ~/backups`, then:

```bash
V=2.337.0   # current: https://github.com/actions/runner/releases
mkdir -p ~/runners/planboard && cd ~/runners/planboard
curl -sfLO https://github.com/actions/runner/releases/download/v$V/actions-runner-linux-arm64-$V.tar.gz
tar xzf actions-runner-linux-arm64-$V.tar.gz && rm actions-runner-linux-arm64-$V.tar.gz
./config.sh --unattended --url https://github.com/weleec/planboard \
  --token "$(gh api -X POST repos/weleec/planboard/actions/runners/registration-token --jq .token)" \
  --name pi5-planboard --labels pi     # add --replace if the old registration is still listed
sudo ./svc.sh install mlovera && sudo ./svc.sh start
gh api repos/weleec/planboard/actions/runners --jq '.runners[] | "\(.name) \(.status)"'
```

The full guide — operating, pausing and removing it, and the rules that keep it safe — is the
application repository's `docs/operations/staging-runner.md`.

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