# paintress docs

## Architecture

- [conformance.md](conformance.md) — What/Why/How conformance table (single source)
- [expedition-directory.md](expedition-directory.md) — `.expedition/` directory structure specification
- [policies.md](policies.md) — Event → Policy mapping (WHEN event THEN command)
- [otel-backends.md](otel-backends.md) — OpenTelemetry backend configuration (Jaeger, Weave)
- [approval-contract.md](approval-contract.md) — Three-way approval contract design
- [dmail-protocol-conventions.md](dmail-protocol-conventions.md) — D-Mail filename uniqueness and archive retention conventions
- [dmail-protocol.md](dmail-protocol.md) — D-Mail protocol implementation details
- [testing.md](testing.md) — Test strategy and conventions

## CLI Reference

- [paintress](cli/paintress.md) — Root command
- [paintress init](cli/paintress_init.md) — Initialize a project (`--force` to regenerate)
- [paintress run](cli/paintress_run.md) — Run the autonomous expedition loop
- [paintress issues](cli/paintress_issues.md) — Query Linear issues via Claude MCP
- [paintress status](cli/paintress_status.md) — Show expedition status
- [paintress config show](cli/paintress_config.md) — Display project configuration
- [paintress config set](cli/paintress_config.md) — Update a project configuration value
- [paintress doctor](cli/paintress_doctor.md) — Diagnose configuration issues (git-remote, deprecated kind detection)
- [paintress clean](cli/paintress_clean.md) — Clean state files
- [paintress rebuild](cli/paintress_rebuild.md) — Rebuild state from events
- [paintress archive-prune](cli/paintress_archive-prune.md) — Prune archived data
- [paintress version](cli/paintress_version.md) — Show version
- [paintress update](cli/paintress_update.md) — Self-update

## Architecture Decision Records

- [adr/](adr/README.md) — Tool-specific ADRs
- [shared-adr/](shared-adr/README.md) — Cross-tool shared ADRs (S0001–S0031)
