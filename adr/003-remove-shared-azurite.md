# ADR 003: Remove Shared Azurite Service

## Status
Accepted

## Context
Azurite was initially added to the `core/` shared infrastructure to provide a centralized Azure Storage API emulator (Blob, Queue, Table) for other lab services.

However, several practical issues emerged:
1. **Tooling Expectations**: Most development tools and SDKs that interact with Azurite (like Azure Storage Explorer or local CLI tools) default to `localhost`. Connecting to a remote IP within the lab network required additional configuration that proved cumbersome.
2. **Impractical Exposure**: Exposing storage emulation over the local network added complexity without significant benefit for the intended use cases.
3. **Resource Usage**: Azurite is extremely lightweight. The overhead of managing it as a centralized core service (with shared volumes and network configuration) outweighs the simplicity of having individual projects spin up their own localized instances when needed.

## Decision
We will remove the `core/azurite` service and its associated data volumes from the shared infrastructure. Moving forward, any service requiring Azure Storage emulation should include its own localized Azurite container within its `docker-compose.yml`.

## Consequences
- The shared `azurite` service is no longer available at `192.168.1.8:10000-10002`.
- `lab up azurite` will no longer work.
- Documentation has been updated to remove references to the shared storage emulator.
- Infrastructure complexity is reduced.
