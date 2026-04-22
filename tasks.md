# EPIC: Centralized Lab Management & CLI

## Tasks

- [x] **COMPLETED** Task 1: Build the `lab` CLI Core
  - Create a new bash script at `shared/lab`.
  - Implement project discovery: dynamically find any directory containing a `docker-compose.yml` within `core/`, `external/`, and `services/`.
  - Implement the base commands: `list`, `ps`, `up`, `down`, `logs`.
  - Make the script executable.

- [x] **COMPLETED** Task 2: Inject the CLI into the Environment
  - Update `shared/setup-lab.sh` to add an alias for the `lab` CLI so it can be run from anywhere.
  - Ensure this injection is idempotent.



- [/] **IN PROGRESS** Task 6: Secure External Access (SSH & Web)
  - [x] Update project `config.yml` for `rpi.mlovera.dev` SSH routing.
  - [x] Configure SSH Hostname in Cloudflare Zero Trust Dashboard.
  - [ ] Configure Cloudflare Access Application for Browser-based SSH.
  - [ ] Verify local SSH client connectivity via `cloudflared` proxy command.
