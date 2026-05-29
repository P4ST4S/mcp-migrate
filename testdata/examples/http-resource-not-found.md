# MCP Migration Report

## Severity Legend

- `breaking`: Incompatible with a strict MCP 2026-07-28 peer. This does not mean the feature stops working on July 28, 2026.

- `deprecated`: Still functional in MCP 2026-07-28, but in the Deprecated lifecycle state. Deprecated features remain functional for at least 12 months before earliest removal eligibility.

- `warning`: Operational risk or minor non-conformance that may affect portability, scaling, or future migration.

- `info`: Informational modernization or interoperability suggestion.

## resource-not-found-code

- Severity: `breaking`
- Spec target: `2026-07-28`
- Enforcement: `report-only`
- SEP: `SEP-2164` (`Draft`, `unverified`)

Resource not found uses a legacy or non-standard JSON-RPC error code.

resources/read returned legacy JSON-RPC error code -32002 for a missing resource.

Remediation: Use -32602 Invalid Params for missing resources after final spec reconciliation.
