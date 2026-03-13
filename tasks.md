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

- [x] **COMPLETED** Task 3: Automate "Always-On" Infrastructure
  - Update `shared/setup-lab.sh` to automatically trigger `lab up proxy` and `lab up dockge` at the end of the script.
  - Ensure other projects like `n8n`, `ollama`, `openclaw`, and `openproject` remain optional.

- [x] **COMPLETED** Task 4: Scaffold OpenProject (Optional)
  - Create a `services/openproject/` directory with a minimal `docker-compose.yml` so that it appears in the `lab` CLI and Dockge.

- [x] **COMPLETED** Task 5: Configure OpenProject with Official Multi-Container Setup
  - Update `external/openproject/docker-compose.yml` with the official configuration.
  - Set up necessary environment variables in `.env`.
  - Confirm service is running and accessible.
