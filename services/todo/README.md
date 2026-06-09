# Todo App & MCP Server

A lightweight, high-performance Go/Fiber service running a web-based Todo list and exposing a Model Context Protocol (MCP) server for AI agent interactions.

## How to Run

1. Initialize the lab environment (if not already done):
   ```bash
   source ./shared/setup-lab.sh
   ```
2. Provision the Postgres database user and database:
   ```bash
   secrets db create-user postgres todo_db todo_user <password>
   ```
3. Copy `example.env` to `.env` and fill in the database URL and authorization key:
   ```bash
   cp example.env .env
   ```
4. Start the service using the lab CLI:
   ```bash
   lab up todo
   ```

## How to Use

- **Web UI**: Access the UI in your browser at `http://todo.rpi.local` or `https://todo.mlovera.dev`.
- **REST API**: Integrates standard JSON endpoints at `/api/todos`.
- **MCP Server**: Exposes an SSE protocol listener at `/mcp/sse` for AI agents.

## Configuration

The application is configured using environment variables in the local `.env` file:

- `PORT`: Port the Fiber application runs on (default: `3000`).
- `DATABASE_URL`: Connection URL to Postgres.
- `API_KEY`: Secret Bearer token AI agents must supply to authenticate MCP commands.

## Networking

- **Internal Discovery**: The service is reachable on `lab-network` by its container name `todo` on port `3000`.
- **Proxy Ingress**: Configured in Nginx (`core/proxy/conf.d/proxy.conf`) to map domains `todo.rpi.local` and `todo.mlovera.dev` to `todo:3000`.
- **Remote Ingress**: Routed via `lab-cloudflared` to Nginx on port 80.
