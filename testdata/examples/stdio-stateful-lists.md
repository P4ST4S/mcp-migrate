# MCP Migration Report

## Severity Legend

- `breaking`: Incompatible with a strict MCP 2026-07-28 peer. This does not mean the feature stops working on July 28, 2026.

- `deprecated`: Still functional in MCP 2026-07-28, but in the Deprecated lifecycle state. Deprecated features remain functional for at least 12 months before earliest removal eligibility.

- `warning`: Operational risk or minor non-conformance that may affect portability, scaling, or future migration.

- `info`: Informational modernization or interoperability suggestion.

## explicit-state-handles

- Severity: `warning`
- Spec target: `2026-07-28`
- Enforcement: `enforced`
- SEP: `SEP-2567` (`Final`, `verified`)

Stateful workflow is not represented by explicit handles.

tools/list stdio result changed within one analyzer-owned process between read-only probes tools-list and state-tools-list-repeat.

Remediation: Mint opaque handles in tool results and require them as ordinary arguments on subsequent calls.

## explicit-state-handles

- Severity: `warning`
- Spec target: `2026-07-28`
- Enforcement: `enforced`
- SEP: `SEP-2567` (`Final`, `verified`)

Stateful workflow is not represented by explicit handles.

resources/list stdio result changed within one analyzer-owned process between read-only probes resources-list and state-resources-list-repeat.

Remediation: Mint opaque handles in tool results and require them as ordinary arguments on subsequent calls.

## explicit-state-handles

- Severity: `warning`
- Spec target: `2026-07-28`
- Enforcement: `enforced`
- SEP: `SEP-2567` (`Final`, `verified`)

Stateful workflow is not represented by explicit handles.

prompts/list stdio result changed within one analyzer-owned process between read-only probes prompts-list and state-prompts-list-repeat.

Remediation: Mint opaque handles in tool results and require them as ordinary arguments on subsequent calls.
