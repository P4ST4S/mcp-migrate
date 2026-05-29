# MCP Migration Report

## Severity Legend

- `breaking`: Incompatible with a strict MCP 2026-07-28 peer. This does not mean the feature stops working on July 28, 2026.

- `deprecated`: Still functional in MCP 2026-07-28, but in the Deprecated lifecycle state. Deprecated features remain functional for at least 12 months before earliest removal eligibility.

- `warning`: Operational risk or minor non-conformance that may affect portability, scaling, or future migration.

- `info`: Informational modernization or interoperability suggestion.

## mcp-session-id-removed

- Severity: `breaking`
- Spec target: `2026-07-28`
- Enforcement: `enforced`
- SEP: `SEP-2567` (`Final`, `verified`)

Server depends on the removed Mcp-Session-Id protocol session header.

A response indicated Mcp-Session-Id usage. Header values and body content are redacted.

Remediation: Replace protocol sessions with explicit application handles passed through tool arguments and results.

## initialize-text-heuristic

- Severity: `warning`
- Spec target: `2026-07-28`
- Enforcement: `enforced`
- SEP: `SEP-2575` (`Accepted`, `unverified`)

Response text mentions initialize, which is only a weak legacy signal.

A response mentioned initialize. This is a weak heuristic only; body content is redacted.

Remediation: Confirm with stronger evidence such as server/discover failure plus a successful legacy initialize probe on an isolated stdio process.

## server-discover-required

- Severity: `breaking`
- Spec target: `2026-07-28`
- Enforcement: `enforced`
- SEP: `SEP-2575` (`Accepted`, `unverified`)

Server does not expose the stateless server/discover RPC.

Probe discover returned JSON-RPC error code -32000.

Remediation: Implement server/discover with supported versions, server capabilities, and server identity.
