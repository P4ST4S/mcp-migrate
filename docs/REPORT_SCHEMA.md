# JSONL Report Schema

Schema id: `mcp-migrate/finding/v1`

Each JSONL line is one finding object. Empty output is valid JSONL and means no findings were emitted.

## Fields

| Field | Type | Required | Notes |
| --- | --- | --- | --- |
| `schema` | string | yes | Always `mcp-migrate/finding/v1` for this schema. |
| `rule` | string | yes | Stable rule id from `internal/rules` and `docs/SPEC_RULES.md`. |
| `sep` | object | no | SEP reference. Present only when the rule maps to a SEP-like identifier. |
| `sep.id` | string | yes, when `sep` exists | Example: `SEP-2567`. |
| `sep.status` | string | no | SEP index/file status observed during verification, such as `Final`, `Accepted`, `Draft`, `In-Review`, or `unindexed`. |
| `sep.verification` | string | yes, when `sep` exists | `verified` only when the SEP status is `Final` and the SEP file was found. Otherwise `unverified`. Do not treat unverified SEP ids as authoritative. |
| `sep.source` | string | no | URL of the SEP file when found. Empty for unindexed ids. |
| `severity` | string | yes | One of `breaking`, `deprecated`, `warning`, `info`. See legend below. |
| `enforcement` | string | yes | `enforced` or `report-only`. Rules with `status: pending-verification` are always `report-only` until reconciled with the final spec. |
| `spec_target` | string | yes | Target MCP spec version, currently `2026-07-28`. |
| `source` | object | yes | Where the finding came from. |
| `source.mode` | string | yes | `live` or `static`. |
| `source.ref` | string | no | Endpoint, command, file path, or fixture reference. |
| `message` | string | yes | Human-readable summary. |
| `detail` | string | no | Probe evidence or additional context. |
| `remediation` | string | no | Suggested fix. |
| `autofix` | boolean | yes | Whether `patch` can safely fix this finding. |
| `status` | string | no | Rule verification status, usually `confirmed` or `pending-verification`. |

## Severity Legend

- `breaking`: incompatible with a strict MCP `2026-07-28` peer. This does not mean the feature stops working on July 28, 2026.
- `deprecated`: still functional in MCP `2026-07-28`, but in the Deprecated lifecycle state. Deprecated features remain functional for at least 12 months before earliest removal eligibility.
- `warning`: operational risk or minor non-conformance that may affect portability, scaling, or future migration.
- `info`: informational modernization or interoperability suggestion.

## Example

```json
{"schema":"mcp-migrate/finding/v1","rule":"resource-not-found-code","sep":{"id":"SEP-2164","status":"Draft","verification":"unverified","source":"https://github.com/modelcontextprotocol/modelcontextprotocol/blob/main/seps/2164-resource-not-found-error.md"},"severity":"breaking","enforcement":"report-only","spec_target":"2026-07-28","source":{"mode":"live","ref":"http://localhost:3000/mcp"},"message":"Resource not found uses a legacy or non-standard JSON-RPC error code.","remediation":"Use -32602 Invalid Params for missing resources after final spec reconciliation.","autofix":true,"status":"pending-verification"}
```
