# MCP Migration Report

## Severity Legend

- `breaking`: Incompatible with a strict MCP 2026-07-28 peer. This does not mean the feature stops working on July 28, 2026.

- `deprecated`: Still functional in MCP 2026-07-28, but in the Deprecated lifecycle state. Deprecated features remain functional for at least 12 months before earliest removal eligibility.

- `warning`: Operational risk or minor non-conformance that may affect portability, scaling, or future migration.

- `info`: Informational modernization or interoperability suggestion.

## session-dependent-lists-removed

- Severity: `warning`
- Spec target: `2026-07-28`
- Enforcement: `report-only`
- SEP: `SEP-2567` (`Final`, `verified`)

List results appear to vary by connection or hidden session state.

tools/list result changed between read-only probes tools-list and state-tools-list-repeat.

Remediation: Make tools/list, resources/list, and prompts/list independent of connection/session state.

## session-dependent-lists-removed

- Severity: `warning`
- Spec target: `2026-07-28`
- Enforcement: `report-only`
- SEP: `SEP-2567` (`Final`, `verified`)

List results appear to vary by connection or hidden session state.

resources/list result changed between read-only probes resources-list and state-resources-list-repeat.

Remediation: Make tools/list, resources/list, and prompts/list independent of connection/session state.

## session-dependent-lists-removed

- Severity: `warning`
- Spec target: `2026-07-28`
- Enforcement: `report-only`
- SEP: `SEP-2567` (`Final`, `verified`)

List results appear to vary by connection or hidden session state.

prompts/list result changed between read-only probes prompts-list and state-prompts-list-repeat.

Remediation: Make tools/list, resources/list, and prompts/list independent of connection/session state.
