# paintress docs

## Architecture

- [conformance.md](conformance.md) — What/Why/How conformance table (single source)
- [expedition-directory.md](expedition-directory.md) — `.expedition/` directory structure specification
- [policies.md](policies.md) — Event → Policy mapping (WHEN event THEN command)
- [otel-backends.md](otel-backends.md) — OpenTelemetry backend configuration (Jaeger, Weave)
- Claude subprocess isolation: `mcp-config generate` creates MCP allowlist, `--strict-mcp-config` enforces it
- Claude log persistence: raw NDJSON saved to `.run/claude-logs/` after each invocation

- [approval-contract.md](approval-contract.md) — Three-way approval contract design
- [dmail-protocol-conventions.md](dmail-protocol-conventions.md) — D-Mail filename uniqueness and archive retention conventions
- [dmail-protocol.md](dmail-protocol.md) — D-Mail protocol implementation details
- [testing.md](testing.md) — Test strategy and conventions

## CLI Reference

- [paintress](cli/paintress.md) — Root command
- [paintress init](cli/paintress_init.md) — Initialize project configuration
- [paintress run](cli/paintress_run.md) — Run the expedition loop
- [paintress issues](cli/paintress_issues.md) — List Linear issues via Claude MCP
- [paintress status](cli/paintress_status.md) — Show operational status
- [paintress config](cli/paintress_config.md) — View or update project configuration
- [paintress config show](cli/paintress_config_show.md) — Show current configuration
- [paintress config set](cli/paintress_config_set.md) — Update a configuration value
- [paintress doctor](cli/paintress_doctor.md) — Run health checks
- [paintress clean](cli/paintress_clean.md) — Remove state directory (.expedition/)
- [paintress rebuild](cli/paintress_rebuild.md) — Rebuild projections from event store
- [paintress archive-prune](cli/paintress_archive-prune.md) — Prune old archived d-mails
- [paintress version](cli/paintress_version.md) — Print version, commit, and build information
- [paintress update](cli/paintress_update.md) — Self-update paintress to the latest release

## Architecture Decision Records

- [adr/](adr/README.md) — Tool-specific ADRs
- [shared-adr/](shared-adr/README.md) — Cross-tool shared ADRs (S0001–S0035)
