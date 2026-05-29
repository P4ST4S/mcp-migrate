# Implementation Plan

Target: `mcp-migrate` v0.1, a Go CLI that probes MCP servers live over stdio or Streamable HTTP, reports migration findings for the 2026-07-28 spec, and only applies safe mechanical patches.

This plan is based on `docs/SPEC_RULES.md`. Any rule tagged `status: pending-verification` must be rechecked against the final 2026-07-28 spec before it becomes a hard failure.

## Product Decisions

- CLI framework: use the standard library `flag` package for v0.1. The command surface is small (`analyze`, later `patch`), avoiding Cobra keeps the binary light and dependency-free. Revisit Cobra only if nested commands, shell completion, or plugin-style command discovery become real needs.
- Go layout: keep `cmd/mcp-migrate` plus `internal/*`. The Go module layout guide recommends `internal` for non-public implementation packages and `cmd` for installable commands in repositories that may grow supporting packages.
- Report format: JSONL is the primary interface. Each finding is one JSON object with a versioned `schema` field. Markdown is a renderer over the same model, never a separate analysis path.
- Rule storage: rules are declarative metadata in `internal/rules`, with each rule linking to a `SPEC_RULES.md` entry and carrying `id`, `sep`, `severity`, `applies_to`, `autofixable`, and `status`.
- Competitive posture: official SDKs may ship their own 2026-07-28 migrations or codemods during the RC validation window. `patch` is a demo hook, not the durable moat. The durable value is cross-language live conformance, hidden state detection, and later `watch`.

## Phase 0: Repo Bootstrap

Complexity: S

Dependencies: none.

Work:

- Create Go module `github.com/P4ST4S/mcp-migrate`.
- Add `cmd/mcp-migrate/main.go`.
- Add `internal/cli` with root command dispatch and `analyze` flag parsing.
- Add minimal package skeletons for `spec`, `rules`, `analyze/live`, `analyze/static`, `state`, `report`, and `patch`.
- Add standard table-driven tests for CLI parsing and report emitters.
- Add CI, Dockerfile, and GoReleaser draft files with TODO markers where release metadata is not final.

Done:

- `go build ./...` passes.
- `go test ./...` passes.
- `mcp-migrate analyze --transport stdio --server-command "..."` accepts flags but performs no probing yet.
- `mcp-migrate analyze --transport http --url http://localhost:3000/mcp` accepts flags but performs no probing yet.
- Empty analysis emits valid JSONL: no findings means zero finding lines and no invalid placeholder JSON.

## Phase 1: Report and Rule Engine

Complexity: M

Dependencies: Phase 0.

Work:

- Define `report.Finding` with fields:
  `schema`, `rule`, `sep`, `severity`, `spec_target`, `source`, `message`, `detail`, `remediation`, `autofix`, `status`.
- Implement JSONL writer and Markdown renderer.
- Implement declarative rule registry seeded from `SPEC_RULES.md`, but do not run protocol probes yet.
- Add validation that rule IDs are unique and severities/statuses are known.

Done:

- JSONL tests cover escaping, multiple findings, and empty output.
- Markdown tests cover grouping by severity and stable ordering.
- Registry tests catch duplicate IDs and missing spec references.

## Phase 2: Live HTTP Analyzer

Complexity: L

Dependencies: Phase 1.

Work:

- Implement HTTP probe client for Streamable HTTP.
- Probe `server/discover` with 2026-07-28 metadata.
- Probe legacy `initialize` behavior only as a compatibility signal, not as the desired path.
- Send representative requests with and without:
  `MCP-Protocol-Version`, `Mcp-Method`, `Mcp-Name`, and per-request `_meta`.
- Validate response behavior for:
  protocol-version mismatch, missing headers, header/body mismatch, `tools/list` cache fields, and visible session requirements.
- Record all raw probe observations in an internal trace structure, then convert to findings through rules.

Done:

- Detects servers that require `Mcp-Session-Id`.
- Detects missing `server/discover`.
- Detects missing/ignored required HTTP headers.
- Detects `tools/list`/resource list cache metadata gaps where probeable.
- Integration tests use `httptest.Server` fixtures for compliant, legacy, and mixed-behavior servers.

## Phase 3: Live STDIO Analyzer

Complexity: L

Dependencies: Phase 1.

Work:

- Spawn a server command with controlled stdin/stdout/stderr.
- Implement JSON-RPC request/response correlation with timeouts.
- Probe `server/discover` first for 2026-07-28 behavior.
- Fall back to legacy `initialize` probe to classify older servers.
- Detect use of removed methods (`ping`, legacy resources subscriptions, `initialize`-only flow) and MRTR result shape where observable.

Done:

- Handles clean process shutdown and timeout cleanup.
- Tests use small fake stdio server binaries or Go test helper processes.
- Findings include exact source mode and command reference without leaking environment secrets.

## Phase 4: Hidden State Detector

Complexity: XL

Dependencies: Phase 2 and Phase 3.

Work:

- Define live state signatures:
  session header required, list results change after a stateful call, tool call succeeds only after same-connection setup, cross-request state not represented by tool arguments, and retry to a fresh connection loses required state.
- For HTTP, run paired probes against independent clients/connections and compare behavior.
- For STDIO, classify process-lifetime state separately: useful warning, not necessarily direct HTTP non-compliance.
- Add explicit remediation hints: state handles, auth-principal keyed storage, documented TTL, cleanup/list tools.

Done:

- Detects at least one session-keyed HTTP fixture.
- Detects one stateful stdio fixture as `warning`, not `breaking`.
- Avoids false positives where state is represented by explicit returned handles.
- Produces concise findings suitable for `jq` filtering.

## Phase 5: Safe Patch Engine

Complexity: M

Dependencies: Phase 1.

Work:

- Implement dry-run-first patch command.
- Support only safe transformations:
  resource-not-found error literal `-32002` to `-32602` in clearly scoped contexts, and HTTP request header injection in simple recognized client call sites.
- Require `--write` for filesystem writes.
- Make patches idempotent.
- Emit patch findings as JSONL and optional unified diff.

Done:

- Dry-run is default and writes nothing.
- `--write` updates files only when the transformation is context-confirmed.
- Tests cover idempotency and refusal for ambiguous code.
- No automatic refactor of server state is implemented.

## Phase 6: Packaging and CI Hardening

Complexity: M

Dependencies: Phases 0-5 as relevant.

Work:

- Finalize GitHub Actions for test/build.
- Add `goreleaser` config for local snapshot builds.
- Finalize Dockerfile for scratch/distroless-style runtime if no CGO dependency appears.
- Add README usage section without overwriting existing human-owned content.

Done:

- CI runs `go test ./...` and `go build ./...`.
- Snapshot release builds produce local artifacts.
- Docker image runs `mcp-migrate analyze --help`.

## v0.1 Rule Priorities

Implement first:

- `initialize-handshake-removed`
- `server-discover-required`
- `protocol-version-per-request`
- `client-info-capabilities-per-request`
- `mcp-session-id-removed`
- `session-dependent-lists-removed`
- `http-standard-headers`
- `cacheable-results-required`
- `resource-not-found-code`

Implement as warnings/deprecations:

- `roots-deprecated`
- `sampling-deprecated`
- `logging-deprecated`
- `http-sse-transport-deprecated`
- `tasks-core-to-extension`
- hidden state signatures from `explicit-state-handles`

Defer or mark pending until final spec:

- Exact MRTR discriminator spelling.
- Strict enforcement around `logging/setLevel` versus Logging deprecation.
- Auth flows that require a browser/OAuth round trip unless fixtures are provided.

## Non-Goals for v0.1

- `watch` mode.
- Static multi-language scanning.
- Automatic semantic refactor of state/session handling.
- Full OAuth authorization flow automation against arbitrary real providers.
- Replacing the official conformance suite.

## Risk Register

- Spec drift before 2026-07-28: keep pending statuses in rule metadata and add a final-spec verification phase before v0.1.
- False positives in hidden-state detection: classify uncertain state as `warning` with probe evidence, not `breaking`.
- SDK codemods reduce `patch` differentiation: keep patch narrow and invest in live conformance plus state signatures.
- Live probes can mutate real servers: default to read/list/discover probes; require explicit opt-in for tool calls that may change state.
- STDIO command execution can hang: enforce timeouts, cancellation, and stderr capture limits.
