# MCP Migration Report

## Severity Legend

- `breaking`: Incompatible with a strict MCP 2026-07-28 peer. This does not mean the feature stops working on July 28, 2026.

- `deprecated`: Still functional in MCP 2026-07-28, but in the Deprecated lifecycle state. Deprecated features remain functional for at least 12 months before earliest removal eligibility.

- `warning`: Operational risk or minor non-conformance that may affect portability, scaling, or future migration.

- `info`: Informational modernization or interoperability suggestion.

## server-discover-required

- Severity: `breaking`
- Spec target: `2026-07-28`
- Enforcement: `enforced`
- SEP: `SEP-2575` (`Accepted`, `unverified`)

Server does not expose the stateless server/discover RPC.

Probe discover returned JSON-RPC error code -32601.

Remediation: Implement server/discover with supported versions, server capabilities, and server identity.

## initialize-handshake-removed

- Severity: `breaking`
- Spec target: `2026-07-28`
- Enforcement: `enforced`
- SEP: `SEP-2575` (`Accepted`, `unverified`)

Legacy initialize handshake is not the MCP 2026-07-28 stateless path.

Legacy initialize succeeded on an isolated stdio process after server/discover failed.

Remediation: Carry protocol version, client info, and client capabilities in per-request _meta and expose server/discover.

## client-info-capabilities-per-request

- Severity: `breaking`
- Spec target: `2026-07-28`
- Enforcement: `enforced`
- SEP: `SEP-2575` (`Accepted`, `unverified`)

Request is missing per-request client identity or capabilities.

Server accepted read-only stdio tools/list with no per-request _meta.

Remediation: Send io.modelcontextprotocol/clientInfo and io.modelcontextprotocol/clientCapabilities in request _meta.

## cacheable-results-required

- Severity: `breaking`
- Spec target: `2026-07-28`
- Enforcement: `enforced`
- SEP: `SEP-2549` (`Accepted`, `unverified`)

Cacheable result is missing ttlMs or cacheScope.

tools/list stdio response was accepted but did not include both ttlMs and cacheScope.

Remediation: Return ttlMs and cacheScope on list/read results covered by the 2026-07-28 changelog.

## cacheable-results-required

- Severity: `breaking`
- Spec target: `2026-07-28`
- Enforcement: `enforced`
- SEP: `SEP-2549` (`Accepted`, `unverified`)

Cacheable result is missing ttlMs or cacheScope.

resources/list stdio response was accepted but did not include both ttlMs and cacheScope.

Remediation: Return ttlMs and cacheScope on list/read results covered by the 2026-07-28 changelog.

## cacheable-results-required

- Severity: `breaking`
- Spec target: `2026-07-28`
- Enforcement: `enforced`
- SEP: `SEP-2549` (`Accepted`, `unverified`)

Cacheable result is missing ttlMs or cacheScope.

prompts/list stdio response was accepted but did not include both ttlMs and cacheScope.

Remediation: Return ttlMs and cacheScope on list/read results covered by the 2026-07-28 changelog.
