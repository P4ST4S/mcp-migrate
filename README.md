# mcp-migrate

[![CI](https://img.shields.io/github/actions/workflow/status/P4ST4S/mcp-migrate/ci.yml?branch=main&label=CI)](https://github.com/P4ST4S/mcp-migrate/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/P4ST4S/mcp-migrate)](https://goreportcard.com/report/github.com/P4ST4S/mcp-migrate)
[![Go version](https://img.shields.io/badge/Go-1.24+-00ADD8?logo=go)](https://go.dev/doc/install)
[![License](https://img.shields.io/github/license/P4ST4S/mcp-migrate)](./LICENSE)

Live conformance analyzer and safe patcher for the MCP `2026-07-28` spec. Probes a running MCP server — over stdio or Streamable HTTP — and reports what needs to change, with severity, remediation, and machine-readable JSONL output.

> **Pre-release — `0.x`**
> Targeting the MCP `2026-07-28` release candidate (locked 2026-05-21, final expected 2026-07-28).
> The CLI interface and JSONL schema may change before v1.

## Contents

- [Install](#install)
- [Quick start](#quick-start)
- [Commands](#commands)
- [Safety](#safety)
- [Output format](#output-format)
- [Severity model](#severity-model)
- [Development](#development)
- [Contributing](#contributing)
- [License](#license)

## Install

```sh
go install github.com/P4ST4S/mcp-migrate/cmd/mcp-migrate@latest
```

Or build from source:

```sh
git clone https://github.com/P4ST4S/mcp-migrate
cd mcp-migrate
go build -o ./bin/mcp-migrate ./cmd/mcp-migrate
```

Docker:

```sh
docker run --rm ghcr.io/p4st4s/mcp-migrate analyze --transport http --url http://host.docker.internal:3000/mcp
```

Requires Go 1.24+.

## Quick start

Analyze a Streamable HTTP server:

```sh
mcp-migrate analyze --transport http --url http://localhost:3000/mcp
```

Analyze a stdio server:

```sh
mcp-migrate analyze --transport stdio --server-command "node ./server.js"
```

Filter to breaking findings:

```sh
mcp-migrate analyze --transport http --url http://localhost:3000/mcp \
  | jq -r 'select(.severity == "breaking") | [.rule, .enforcement, .message] | @tsv'
```

Render a Markdown report:

```sh
mcp-migrate analyze --transport http --url http://localhost:3000/mcp \
  --format markdown > report.md
```

Preview source-code patches without writing:

```sh
mcp-migrate patch --path ./src --allow-pending
```

Apply patches:

```sh
mcp-migrate patch --path ./src --allow-pending --write
```

## Commands

### `analyze`

| Flag | Default | Description |
|---|---|---|
| `--transport` | `http` | `http` or `stdio` |
| `--url` | — | Streamable HTTP endpoint (HTTP only) |
| `--server-command` | — | Server command to spawn (stdio only) |
| `--format` | `jsonl` | `jsonl` or `markdown` |
| `--spec-target` | `2026-07-28` | Target spec version |
| `--allow-resource-read` | off | Include `resources/read` probes |
| `--allow-mutating-probes` | off | Allow probes that may modify state |

### `patch`

Rewrites `–32002` (legacy resource-not-found error code) to `–32602` in Go, JS/TS, and Python source files. Only replaces occurrences where the surrounding code confirms a resource-not-found context.

| Flag | Default | Description |
|---|---|---|
| `--path` | — | File or directory to scan (required) |
| `--write` | off | Write changes to disk (default: dry-run) |
| `--allow-pending` | off | Required: the underlying SEP-2164 is still Draft |

Without `--allow-pending`, `patch` refuses with an explanation. This is intentional: patching based on a non-final spec carries the risk of a second migration.

## Safety

**`analyze` is read-only by default.** Default probes: `server/discover`, `tools/list`, `resources/list`, `prompts/list`. `resources/read` requires `--allow-resource-read`. `tools/call` is never sent.

**Hidden-state detection** repeats the same read-only list probes and compares canonicalized results. Drift caused by explicit handle fields (`stateHandle`, etc.) is normalized and ignored.

**`patch` is dry-run by default.** Writes nothing without `--write`. Substitutions are context-confirmed: `–32002` is replaced only when a resource-not-found signal (`resources/read`, `resource not found`, `ResourceNotFound`) appears within ±2 lines of the occurrence in non-comment code. Distant mentions in comments do not qualify. Patches are idempotent.

**Secret redaction.** JSONL and Markdown output never contains tokens, auth headers, environment variables, stderr, response bodies, raw headers, cookies, URL userinfo, or sensitive query parameters.

## Output format

JSONL: one finding per line, each an independent JSON object.

```json
{
  "schema": "mcp-migrate/finding/v1",
  "rule": "server-discover-required",
  "sep": { "id": "SEP-2575", "status": "Accepted", "verification": "unverified" },
  "severity": "breaking",
  "enforcement": "report-only",
  "spec_target": "2026-07-28",
  "source": { "mode": "live", "ref": "http://localhost:3000/mcp" },
  "message": "Server does not expose the stateless server/discover RPC.",
  "remediation": "Implement server/discover with supported versions, server capabilities, and server identity.",
  "autofix": false,
  "status": "confirmed"
}
```

Empty output is valid and means no findings were produced.

`enforcement` is `"enforced"` when the underlying SEP is `Final` and verified, and the detection method is direct observation. It is `"report-only"` when the SEP is not yet `Final`, the detection relies on differential inference (e.g. list drift), or the rule is pending final-spec reconciliation.

Full schema: [`docs/REPORT_SCHEMA.md`](docs/REPORT_SCHEMA.md).

Example reports from test fixtures:

| Profile | JSONL | Markdown |
|---|---|---|
| HTTP — compliant | [http-compliant.jsonl](testdata/examples/http-compliant.jsonl) | [http-compliant.md](testdata/examples/http-compliant.md) |
| HTTP — legacy | [http-legacy.jsonl](testdata/examples/http-legacy.jsonl) | [http-legacy.md](testdata/examples/http-legacy.md) |
| HTTP — mixed | [http-mixed.jsonl](testdata/examples/http-mixed.jsonl) | [http-mixed.md](testdata/examples/http-mixed.md) |
| HTTP — stateful lists | [http-stateful-lists.jsonl](testdata/examples/http-stateful-lists.jsonl) | [http-stateful-lists.md](testdata/examples/http-stateful-lists.md) |
| stdio — compliant | [stdio-compliant.jsonl](testdata/examples/stdio-compliant.jsonl) | [stdio-compliant.md](testdata/examples/stdio-compliant.md) |
| stdio — legacy | [stdio-legacy.jsonl](testdata/examples/stdio-legacy.jsonl) | [stdio-legacy.md](testdata/examples/stdio-legacy.md) |
| stdio — stateful lists | [stdio-stateful-lists.jsonl](testdata/examples/stdio-stateful-lists.jsonl) | [stdio-stateful-lists.md](testdata/examples/stdio-stateful-lists.md) |

## Severity model

| Severity | Meaning |
|---|---|
| `breaking` | Incompatible with a strict `2026-07-28` peer. Does not mean the server breaks on July 28. |
| `deprecated` | Still functional, but in the Deprecated lifecycle state. Earliest removal: 12 months after deprecation. |
| `warning` | Operational risk or non-conformance that may affect portability, scaling, or future migration. |
| `info` | Modernization or interoperability suggestion. |

## Development

```sh
go test ./...
go build ./...
```

Tests use `httptest.Server` fixtures for HTTP and Go helper processes for stdio. No external dependencies — the module is dependency-free.

To regenerate the `testdata/examples/` JSONL and Markdown files after a rule change:

```sh
UPDATE_EXAMPLES=1 go test ./internal/analyze/live
```

## Contributing

See [`CONTRIBUTING.md`](CONTRIBUTING.md) for contribution guidelines, ground rules for new rules, patch requirements, and the evidence-strength model.

Internal documentation:

- [`docs/SPEC_RULES.md`](docs/SPEC_RULES.md) — rule definitions, SEP references, severity rationale
- [`docs/REPORT_SCHEMA.md`](docs/REPORT_SCHEMA.md) — JSONL schema reference
- [`docs/PLAN.md`](docs/PLAN.md) — implementation plan and phase status

## License

Apache-2.0. See [`LICENSE`](LICENSE).
