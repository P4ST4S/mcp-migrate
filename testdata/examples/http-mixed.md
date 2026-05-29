# MCP Migration Report

## Severity Legend

- `breaking`: Incompatible with a strict MCP 2026-07-28 peer. This does not mean the feature stops working on July 28, 2026.

- `deprecated`: Still functional in MCP 2026-07-28, but in the Deprecated lifecycle state. Deprecated features remain functional for at least 12 months before earliest removal eligibility.

- `warning`: Operational risk or minor non-conformance that may affect portability, scaling, or future migration.

- `info`: Informational modernization or interoperability suggestion.

## protocol-version-per-request

- Severity: `breaking`
- Spec target: `2026-07-28`
- Enforcement: `enforced`
- SEP: `SEP-2575` (`Accepted`, `unverified`)

Request is missing per-request protocol version metadata or HTTP version header.

Server accepted a read-only server/discover probe where MCP-Protocol-Version did not match request _meta.

Remediation: Send io.modelcontextprotocol/protocolVersion in _meta on every request and MCP-Protocol-Version on HTTP.

## client-info-capabilities-per-request

- Severity: `breaking`
- Spec target: `2026-07-28`
- Enforcement: `enforced`
- SEP: `SEP-2575` (`Accepted`, `unverified`)

Request is missing per-request client identity or capabilities.

Server accepted a read-only tools/list probe with no per-request _meta.

Remediation: Send io.modelcontextprotocol/clientInfo and io.modelcontextprotocol/clientCapabilities in request _meta.

## http-standard-headers

- Severity: `breaking`
- Spec target: `2026-07-28`
- Enforcement: `enforced`
- SEP: `SEP-2243` (`Final`, `verified`)

Streamable HTTP request is missing required MCP routing headers or accepts header/body mismatch.

Server accepted read-only tools/list without Mcp-Method.

Remediation: Send and validate Mcp-Method on all POSTs and Mcp-Name for tools/call, resources/read, and prompts/get.

## http-standard-headers

- Severity: `breaking`
- Spec target: `2026-07-28`
- Enforcement: `enforced`
- SEP: `SEP-2243` (`Final`, `verified`)

Streamable HTTP request is missing required MCP routing headers or accepts header/body mismatch.

Server accepted read-only tools/list with Mcp-Method that did not match the JSON-RPC method.

Remediation: Send and validate Mcp-Method on all POSTs and Mcp-Name for tools/call, resources/read, and prompts/get.

## cacheable-results-required

- Severity: `breaking`
- Spec target: `2026-07-28`
- Enforcement: `enforced`
- SEP: `SEP-2549` (`Accepted`, `unverified`)

Cacheable result is missing ttlMs or cacheScope.

tools/list response was accepted but did not include both ttlMs and cacheScope.

Remediation: Return ttlMs and cacheScope on list/read results covered by the 2026-07-28 changelog.

## cacheable-results-required

- Severity: `breaking`
- Spec target: `2026-07-28`
- Enforcement: `enforced`
- SEP: `SEP-2549` (`Accepted`, `unverified`)

Cacheable result is missing ttlMs or cacheScope.

resources/list response was accepted but did not include both ttlMs and cacheScope.

Remediation: Return ttlMs and cacheScope on list/read results covered by the 2026-07-28 changelog.

## cacheable-results-required

- Severity: `breaking`
- Spec target: `2026-07-28`
- Enforcement: `enforced`
- SEP: `SEP-2549` (`Accepted`, `unverified`)

Cacheable result is missing ttlMs or cacheScope.

prompts/list response was accepted but did not include both ttlMs and cacheScope.

Remediation: Return ttlMs and cacheScope on list/read results covered by the 2026-07-28 changelog.
