---
name: add-lab-service
description: Specialized skill for extending the Gemini Lab project with a new service (core, external, or custom).
---

# Instructions

You are an expert at extending the Gemini Lab environment with new services.

## Service Categories
When adding a new service, first determine its correct category:
- **`core/`**: Fundamental infrastructure services (always-on, e.g., databases, proxies).
- **`external/`**: Third-party applications (e.g., n8n, Kibana, Mongo Express).
- **`services/`**: Custom applications developed specifically for this lab.

## Technical Requirements
For every new service, you MUST create a new directory in the appropriate category folder and include the following:

1. **`docker-compose.yml`**:
   - MUST join the `lab-network` external bridge network.
   - MUST use Traefik labels for HTTP routing unless it's a TCP-only service (like a database).
   - **Local Network (Default)**: Route to `<service-name>.rpi.local`. Example label: `"traefik.http.routers.<name>.rule=Host(\`<name>.rpi.local\`)"`.
   - **Production (If explicitly requested)**: Route to `<service-name>.mlovera.dev`. Example label: `"traefik.http.routers.<name>.rule=Host(\`<name>.rpi.local\`) || Host(\`<name>.mlovera.dev\`)"`.
   - TCP services (Databases) should expose their ports directly to the host (e.g., `5432:5432`).

2. **Credentials & Environment Variables**:
   - NEVER hardcode secrets in `docker-compose.yml`.
   - Use an `env_file: - .env` directive.
   - MUST create an `example.env` with default/placeholder values.
   - Create the `.env` file (which will be git-ignored) with actual initial credentials.

3. **Documentation**:
   - Create a `README.md` inside the new service directory. Include sections: "How to Run", "How to Use", "Configuration", and "Networking".

4. **Integration & Updates**:
   - Update `docs/networking.md` to include the new service in the "Current Lab Status" table.
   - Update the `shared/lab` CLI tool help examples (`cmd_help`) to include the new service as an example if appropriate.
   - Commit changes following Conventional Commits (e.g., `feat: add <service>`).

# Resources
- file://GEMINI.md
- file://docs/networking.md
- file://shared/setup-lab.sh
- file://shared/lab
