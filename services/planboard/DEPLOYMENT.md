# Planboard — deploying production and staging

Two services, one image, one tag apart. This is the operational side: what to run on the Pi
and in what order. The application repository's `docs/reference/deployment.md` explains *why*
the arrangement is shaped this way; this file assumes you are standing in front of it.

| | Production | Staging |
| --- | --- | --- |
| Hostname | `planboard.mlovera.dev` | `planboard-dev.mlovera.dev` |
| LAN | `planboard.rpi.local` | `planboard-dev.rpi.local` |
| Compose service | `planboard` | `planboard-dev` (+ `planboard-dev-mcp`) |
| Image tag | `ghcr.io/weleec/planboard:X.Y.Z`, pinned in `docker-compose.override.yml` (`latest` without it) | `ghcr.io/weleec/planboard:dev` |
| Moved by | a release tag `vX.Y.Z`, cut and deployed by CI when a release pull request is merged into `main` | a pre-release tag `vX.Y.Z-dev.N`, cut by CI on every merge to `development` |
| Database | lab Postgres, container `postgres` | SQLite on the `planboard-dev-data` volume |
| Environment file | `.env` (required) | `.env.dev` (optional) |

**Both environments deploy themselves, on different merges.** A merge to `development` that
changes more than docs is tagged `vX.Y.Z-dev.N`, built, and deployed to staging by the GitHub
Actions runner on this Pi (see [README.md](README.md#the-github-actions-runner)). Production
moves when a `release/vX.Y.Z` (or `hotfix/*`) pull request is merged into `main` in the
application repository: the same runner then runs `scripts/deploy-production.sh` here. The
merge is the only approval, so nothing in this directory needs a hand-run command for a
release.

## Deploying to staging

**Automatic.** Merge to `development`; the merge's CI run does the rest:

```bash
gh run watch $(gh run list -R weleec/planboard --branch development --limit 1 \
  --json databaseId --jq '.[0].databaseId') -R weleec/planboard --exit-status
```

`Staging tag` prints the tag it cut (or `none` for a docs-only merge), `Build image` publishes
`:dev`, and `Deploy staging` runs `scripts/deploy-staging.sh` on this Pi: back up staging's
SQLite into `~/backups` (newest 30 kept), pull and recreate both staging services, check the
revision label equals the commit, wait for healthy. A red `Deploy staging` is a stop — read its
log before doing anything by hand.

Numbering is SemVer pre-releases **of the version being worked towards**, not of the one in
production, computed from the Conventional Commit subjects since the last release. With 1.7.0
released and a `feat` merged, staging builds are `v1.8.0-dev.1`, `-dev.2`, …; `N` restarts at
1 for each new `X.Y.Z`. Do not cut dev tags by hand: two tags for one version collide on `N`.

**By hand**, only when the runner is down — the same script, from the app checkout:

```bash
~/dev/planboard/scripts/deploy-staging.sh <commit-sha>
```

Or the individual steps, in this directory:

```bash
docker compose pull planboard-dev planboard-dev-mcp
docker compose up -d planboard-dev planboard-dev-mcp
docker compose logs --tail=100 planboard-dev
```

**Name the services.** A bare `docker compose up -d` or `lab up planboard` restarts production
as well, which is not what a staging test should cost.

Verify it came up against the database it should have:

```bash
docker exec planboard-dev bun -e \
  'console.log(await (await fetch("http://localhost:3000/api/health")).text())'
# expect {"status":"ok","driver":"sqlite",...}
```

`driver` is the value to read. If it says `postgres`, stop — staging has reached the school's
database and the compose `environment:` block has been edited.

### What staging is and is not

- **It cannot reach production data.** `PB_DB: sqlite` sits in the compose `environment:`
  block, which overrides `env_file:`, and `.env.dev` carries no `PB_DATABASE_URL`. Do not add
  one. To test against realistic content, restore a dump into a scratch database.
- **It cannot mail real teachers.** No `PB_MAIL_TRANSPORT`, so the app falls back to a
  transport that sends nothing while the whole path still runs and the UI still reports
  success.
- **It is not a Postgres rehearsal.** It exercises SQLite, the driver CI already covers, so a
  green staging run says nothing about the Postgres path. The hand-verification before a
  release does not go away — if anything staging is the tempting reason to skip it.
- **The volume starts empty.** First boot migrates a fresh database and serves a login page
  with no users in it. Seed a scratch admin rather than a copy of the school's data.

## Deploying to production

**Automatic.** The release procedure (*Prepare release*, the release pull request, the merge)
lives in the application repository's `docs/operations/release-pipeline.md` — do not
improvise it from here. On the merge, the runner runs `scripts/deploy-production.sh` in this
directory, which: records `schema_migrations`' high-water mark; takes a `pg_dump` into
`~/backups` (`weleec_planboard-<stamp>-pre-vX.Y.Z.dump`, stopping if it is under 10 KB);
writes the version into `docker-compose.override.yml`; pulls and recreates `planboard` **and**
`planboard-mcp`; checks both revision labels, the `[migrate] applied` lines in both logs,
`/api/health` (`driver: postgres`) and the MCP `/health`; and on any failure pins the previous
version back and recreates both. It never restores the database and never names a staging
service.

**The image pin.** `docker-compose.override.yml` sits beside `docker-compose.yml`, and
`docker compose` (and `lab up planboard`) load it automatically. It holds only the image
version of the two production services; it is written by the deploy script, gitignored here,
and editing its version is how to pin by hand. Without it, production follows `latest`.

**By hand**, only when the pipeline cannot run — the same script, from the app checkout
(`<new-migration-ids>` is e.g. `44,45` or `none`):

```bash
~/dev/planboard/scripts/deploy-production.sh X.Y.Z <commit-sha> <new-migration-ids>
```

Or the individual steps. The database step is the one that has actually bitten. In order:

```bash
# 1. record where the schema is
docker exec postgres psql -U weleec_planboard -d weleec_planboard -tA \
  -c 'SELECT id FROM schema_migrations ORDER BY id;'

# 2. back up BEFORE any DDL runs
mkdir -p ~/backups
docker exec postgres pg_dump -U weleec_planboard -Fc weleec_planboard \
  > ~/backups/weleec_planboard-$(date +%F-%H%M).dump

# 3. pull and start — both production services, by name (set the version in
#    docker-compose.override.yml first if it exists)
docker compose pull planboard planboard-mcp && docker compose up -d planboard planboard-mcp

# 4. watch it migrate itself — in either container: whichever connects first runs them
docker compose logs --tail=200 planboard planboard-mcp | grep -F '[migrate]'

# 5. prove it reached the database (the image has bun, not curl or wget)
docker exec planboard bun -e \
  'console.log(await (await fetch("http://localhost:3000/api/health")).text())'
# expect {"status":"ok","driver":"postgres",...}
```

Read both logs for `[migrate] applied …` lines: one per new migration, once across the two,
and nothing on subsequent starts. Migrations run automatically at startup and a failure **stops the
container** rather than serving against a half-built schema — that is the intended behaviour,
not an outage to work around. Never write SQL by hand to get past a migration error.

`has no Postgres body` in that log means the image predates the Postgres half of a migration.
That is a code problem; roll back and fix it in the repository.

### Rolling back

Roll back the **image tag**, not the database — the automated deploy does this itself on a
failed check. By hand, in `docker-compose.override.yml`:

```yaml
services:
  planboard:
    image: ghcr.io/weleec/planboard:1.3.0
  planboard-mcp:
    image: ghcr.io/weleec/planboard:1.3.0
```

then `docker compose up -d planboard planboard-mcp`. An older version ignores tables it does not know about,
so the extra tables a newer migration created are harmless to it and no restore is needed.
Restore the dump only if a migration left the schema itself broken.

> **1.3.0 is a one-way door.** Migration 24 turned `users.id` into a UUID, and every image
> before 1.3.0 reads user ids as integers — a 1.2.x container on a migrated database cannot
> resolve a session. Rolling back across it means restoring the pre-upgrade dump into an
> emptied database *and* pinning the old tag, losing everything written since. This is why
> step 2 is taken immediately before the pull, not the night before.

## Configuration

Compose's `environment:` block **overrides** `env_file:`. When a setting appears to be
ignored, check the compose file first.

| Variable | Where | Notes |
| --- | --- | --- |
| `PB_DB` | compose `environment:` | `postgres` / `sqlite`. Not settable from an env file |
| `PB_DATABASE_URL` | `.env` only | Carries the password. **Production only.** `#` must stay percent-encoded as `%23` |
| `PB_SQLITE_PATH` | compose, staging | `/app/var/weleec.db`. Never `/app/data` — that holds the export templates baked into the image, and a volume there breaks every export |
| `PB_MCP_URL` / `PB_MCP_ISSUER` | compose, staging | The web app and the separate MCP process must agree on both |
| `PB_LOGIN_MAX_ATTEMPTS` | env file | Raises the 5-per-15-minutes throttle while poking at a build. Leave unset in production |
| `PB_MAIL_*` | env file | All-or-nothing: naming a provider without its key and sender throws at startup |
| `PB_DRIVE_*` | env file | All-or-nothing, see below |

Templates are tracked as `example.env` and `example.dev.env`; the real `.env` and `.env.dev`
are git-ignored and are the only copies of their credentials.

### Google Drive

The four Drive variables are all-or-nothing. Any one missing leaves the feature unconfigured:
Settings shows "Not configured yet" and `/api/drive/*` returns 404. There is no degraded mode,
deliberately — the thing that must not happen is a refresh token stored in plaintext by a
deployment that forgot `PB_DRIVE_TOKEN_KEY`.

**Current state: Drive is configured on staging only.** `.env.dev` carries the client id,
secret, redirect URI and token key; production's `.env` has none of them, so Drive is switched
off at `planboard.mlovera.dev` until those four are added there.

The redirect URI differs per environment and must be registered in Google Cloud **character
for character** — a wrong port or a trailing slash fails at the consent screen with
`redirect_uri_mismatch`:

| Environment | Value |
| --- | --- |
| Staging | `https://planboard-dev.mlovera.dev/api/drive/callback` |
| Production | `https://planboard.mlovera.dev/api/drive/callback` |

`PB_DRIVE_TOKEN_KEY` is 32 random bytes, base64, and **is not recoverable**. It decrypts every
stored Google refresh token; losing or rotating it does not break the app, but every user must
reconnect. Back it up with the deployment's other secrets.

```bash
openssl rand -base64 32          # generate one
```

## Troubleshooting

**Which image is actually running:**

```bash
docker compose ps
docker inspect --format '{{.Config.Image}} {{.Image}}' planboard planboard-dev
```

A `docker compose pull` that reports up to date while you expect a new build means CI has not
finished, or the tag you pushed was not the channel you meant. Check with `gh run list`.

**The app is up but the database is not.** `GET /api/health` runs a real query rather than
returning a constant, which is why it catches exactly this. It publishes no host, database
name or error text; those go to stderr, so read `docker compose logs`.

**A Drive connection fails.** The callback writes exactly one line naming the step, HTTP
status and Google's machine reason — no credentials, by construction:

```bash
docker compose logs planboard-dev | grep '\[drive\]'
```

| Line | Meaning |
| --- | --- |
| `step=exchange …` | Failed before identity: client secret or redirect URI |
| `step=identity status=403 reason=accessNotConfigured` | Drive API not enabled on the Google Cloud project — an operator fix, and the page says so |
| `step=identity status=401 …` | Google refused the token at `drive/v3/about` |
| `step=identity status=200 reason=incompleteIdentity` | Google answered but did not name the account |
| `step=store …` | Google was fine; saving the connection failed — token key or the `drive_accounts` insert |

Available from `v1.4.0-dev.4` onward. An older image writes nothing at all, which is itself
the diagnosis: the banner is generic because the build predates this.

**Nothing reaches the site at all.** Ingress is `Cloudflare edge → lab-cloudflared →
lab-proxy (nginx:80) → planboard:3000`. Neither service publishes a host port, so a direct
`curl` to the Pi is expected to fail. nginx dispatches by `server_name` from
`core/proxy/conf.d/proxy.conf`; after editing it:

```bash
docker exec lab-proxy nginx -s reload
```

`container_name` is fixed in the compose file on purpose — nginx resolves both services by
name, so letting Compose derive a name from the directory turns a renamed folder into a 502.

`X-Forwarded-For` is not optional in those server blocks. The login throttle keys on the first
address in it and falls back to one shared bucket when the header is absent — without it, five
failed logins by anyone lock out the whole school.

**Image pull denied.** The package is private; the ghcr login needs `read:packages`:

```bash
gh auth refresh -h github.com -s read:packages
gh auth token | docker login ghcr.io -u ManasesLovera --password-stdin
```
