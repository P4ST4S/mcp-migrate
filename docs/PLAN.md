# Implementation Plan

Target: `mcp-migrate` v0.1, a Go CLI that probes MCP servers live over stdio or Streamable HTTP, reports migration findings for the 2026-07-28 spec, and only applies safe mechanical patches.

This plan is based on `docs/SPEC_RULES.md`. Any rule tagged `status: pending-verification` must be rechecked against the final 2026-07-28 spec before it becomes a hard failure.

## Review Status

Last updated: 2026-05-30.

- Phase 0 is implemented and committed in `6cd1ae8` (`chore: scaffold go cli`).
- Phase 1 is implemented and committed in `4f9b030` (`feat: add report schema and rule registry`).
- Phase 2 is implemented and committed in `bfe33b0` (`feat: add live http analyzer probes`).
- Phase 3 is implemented in the current branch.
- Phase 4 is implemented in the current branch.
- Phase 5 is implemented in the current branch.
- Phase 6 is implemented in the current branch.
- Validation command: `go build ./...` and `go test ./...`. The HTTP integration tests use `httptest.Server`; stdio integration tests launch helper processes. Both may require extra permissions in sandboxed environments.

## Product Decisions

- CLI framework: use the standard library `flag` package for v0.1. The command surface is small (`analyze`, later `patch`), avoiding Cobra keeps the binary light and dependency-free. Revisit Cobra only if nested commands, shell completion, or plugin-style command discovery become real needs.
- Go layout: keep `cmd/mcp-migrate` plus `internal/*`. The Go module layout guide recommends `internal` for non-public implementation packages and `cmd` for installable commands in repositories that may grow supporting packages.
- Report format: JSONL is the primary interface. Each finding is one JSON object with a versioned `schema` field. Markdown is a renderer over the same model, never a separate analysis path.
- Rule storage: rules are declarative metadata in `internal/rules`, with each rule linking to a `SPEC_RULES.md` entry and carrying `id`, `sep`, `severity`, `applies_to`, `autofixable`, and `status`.
- SEP attribution: JSONL renders SEP references as objects. A SEP is `verified` only when its status is `Final` and its source file has been found; Accepted/Draft/In-Review/unindexed entries render as `unverified`.
- Enforcement policy: rules tagged `pending-verification` always produce `enforcement: "report-only"` until final-spec reconciliation. Rules whose `sep.verification` is `unverified` also produce `report-only` by default, unless a rule carries an explicit documented enforcement override. No current rule uses that override.
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

Status: implemented.

Complexity: M

Dependencies: Phase 0.

Work:

- Define `report.Finding` with fields:
  `schema`, `rule`, `sep`, `severity`, `enforcement`, `spec_target`, `source`, `message`, `detail`, `remediation`, `autofix`, `status`.
- Implement JSONL writer and Markdown renderer with a severity legend at the top of every Markdown report.
- Implement declarative rule registry seeded from `SPEC_RULES.md`, without protocol probes in the rules package.
- Add validation that rule IDs are unique, severities/statuses are known, and each rule has a SEP-like reference.
- Document the JSONL schema in `docs/REPORT_SCHEMA.md`.

Done:

- JSONL tests cover structured SEP output, enforcement, and empty output.
- Markdown tests cover empty reports plus the severity legend and deprecation-window wording.
- Registry tests catch duplicate IDs, missing SEP references, seeded rules, SEP verification tagging, pending-verification report-only behavior, unverified-SEP report-only behavior, and Final/verified SEP enforcement.

## Phase 2: Live HTTP Analyzer

Status: implemented.

Complexity: L

Dependencies: Phase 1.

Work:

- Implement HTTP probe client for Streamable HTTP.
- Probe `server/discover` with 2026-07-28 metadata.
- Do not send legacy `initialize`; legacy servers are detected read-only through failed `server/discover`/list probes, `Mcp-Session-Id` signals, and redacted response text mentioning initialize.
- Send representative requests with and without:
  `MCP-Protocol-Version`, `Mcp-Method`, `Mcp-Name`, and per-request `_meta`.
- Validate response behavior for:
  protocol-version mismatch, missing headers, header/body mismatch, `tools/list` cache fields, and visible session requirements.
- Record all raw probe observations in an internal trace structure, then convert to findings through rules.
- Keep probes read-only by default. Implemented default HTTP probes are `server/discover`, `tools/list`, `resources/list`, and `prompts/list`.
- `resources/read` is no longer part of the default HTTP probe set because real servers may attach side effects such as consume, mark-as-read, or remote fetch. It is available only with `--allow-resource-read`.
- Add `--allow-mutating-probes` as an explicit opt-in flag, disabled by default. No mutating HTTP probes are implemented in Phase 2.
- Redact secrets from output: URL userinfo and sensitive query parameters are masked; response bodies, header values, network errors, and authorization material are not emitted.

Done:

- Detects servers that require `Mcp-Session-Id`.
- Detects missing `server/discover`.
- Detects missing/ignored required HTTP headers.
- Detects per-request `_meta` acceptance gaps where probeable.
- Detects protocol-version header/_meta mismatch acceptance where probeable.
- Detects `tools/list`, `resources/list`, and `prompts/list` cache metadata gaps by default. Also checks `resources/read` when `--allow-resource-read` is set.
- Integration tests use `httptest.Server` fixtures for compliant, legacy, and mixed-behavior servers.
- Tests assert no `tools/call` is sent and no fixture secrets leak into findings.
- Tests cover false-positive and false-negative cases for HTTP response text that mentions initialize. That signal is `initialize-text-heuristic` with `warning` severity, not proof of a legacy initialize requirement.

Current Phase 2 Limitations:

- The analyzer does not probe `Mcp-Name` mismatch for `resources/read` by default; `resources/read` is opt-in.
- Session-dependent list drift is now covered by Phase 4 read-only repeated list probes.
- The analyzer does not yet inspect `x-mcp-header` tool schema behavior.
- The analyzer does not yet perform OAuth/auth flow probes.
- A real-server smoke test is intentionally deferred to a release gate; Phase 2 only requires arbitrary `--url` support and fixture coverage.

## Phase 3: Live STDIO Analyzer

Status: implemented.

Complexity: L

Dependencies: Phase 1.

Work:

- Spawn a server command with controlled stdin/stdout/stderr.
- Implement JSON-RPC request/response correlation with timeouts.
- Probe `server/discover` first for 2026-07-28 behavior.
- Fall back to legacy `initialize` probe to classify older servers.
- Do not send `tools/call` or any other mutating probe.
- Detect legacy initialize-only flow where observable.
- Capture stderr with a fixed bound and never emit stderr content.
- Redact command arguments and never emit environment variables.
- Keep raw stdio observations in `STDIOTrace`/`STDIOObservation`, then convert to findings through the rule engine.

Decision: stdio sends a legacy `initialize` probe after `server/discover` fails. This intentionally differs from HTTP. Stdio analysis launches an isolated process owned by the analyzer and tears it down after probing, so the legacy handshake mutates only process-scoped state in a disposable child process. HTTP targets may be shared remote servers, so HTTP avoids `initialize` and reports only stronger protocol signals plus weak text heuristics.

Done:

- Handles clean process shutdown and timeout cleanup.
- Tests use Go helper processes for compliant, legacy, mixed, and timeout profiles.
- Findings include source mode and redacted command reference without leaking command secrets, environment values, or stderr content.
- Tests assert no `tools/call` is sent, helper stderr secrets do not leak, and timeout probes return promptly.
- `testdata/examples/` contains JSONL and Markdown examples for HTTP compliant/legacy/mixed and stdio compliant/legacy/mixed profiles.

Current Phase 3 Limitations:

- Stdio does not yet probe `ping`, legacy resource subscription methods, or MRTR shapes. Those are still planned follow-up rules for the live stdio analyzer.
- Stdio command parsing supports simple quoting/escaping but is not a full shell. This is intentional: commands are executed directly without invoking a shell.

## Phase 4: Hidden State Detector

Status: implemented.

Complexity: XL

Dependencies: Phase 2 and Phase 3.

Work:

- Define live state signatures:
  session header required, repeated list results changing without explicit handles, cross-request state not represented by tool arguments, and retry to a fresh connection losing required state. Signatures that require a stateful `tools/call` setup are deferred until a dedicated opt-in probe exists.
- For HTTP, run paired read-only list probes against the existing client and a fresh client/connection, then compare behavior.
- For STDIO, classify process-lifetime list drift separately: useful warning, not necessarily direct HTTP non-compliance.
- Add explicit remediation hints: state handles, auth-principal keyed storage, documented TTL, cleanup/list tools.

Done:

- Detects at least one session-keyed HTTP fixture through repeated read-only list drift and reports `session-dependent-lists-removed`.
- Detects one stateful stdio fixture as `explicit-state-handles` with `warning` severity, not as a breaking HTTP/session finding.
- Avoids false positives where drift is limited to explicit returned handle fields such as `stateHandle`.
- Produces concise findings suitable for `jq` filtering.

Current Phase 4 Limitations:

- No `tools/call` state probes are sent by default, so signatures that require a stateful setup call remain out of scope until a dedicated opt-in probe is designed.
- HTTP state drift is detected from canonicalized list result changes; auth-principal or time-based variation can still require human interpretation.
- Stdio state drift is intentionally classified as a warning-style process-lifetime signal.

## Phase 5: Safe Patch Engine

Status: implemented.

Complexity: M

Dependencies: Phase 1.

Work:

- Implement dry-run-first patch command.
- Support only safe transformations:
  resource-not-found error literal `-32002` to `-32602` in clearly scoped contexts.
- Require `--write` for filesystem writes.
- Make patches idempotent.
- Emit patch findings as unified diff.

Done:

- `mcp-migrate patch --path <file|dir>` scans Go, JS/TS, and Python source files.
- Detection strategy: regex match on `-32002` as a standalone token; confirmation requires
  a resource-not-found context signal (`resources/read`, `resource not found`, `resource_not_found`,
  or `ResourceNotFound`) within 12 lines of the occurrence. Occurrences with no context signal
  are counted as `Skipped` and never touched.
- Dry-run is default: diffs are produced and printed, no file is written.
- `--write` rewrites files in-place; permissions are preserved.
- Patches are idempotent: a second pass on an already-patched file produces zero changes.
- Evidence-strength principle mirrors Phase 4: the patcher only acts on context-confirmed
  occurrences and reports ambiguous ones rather than silently skipping or guessing.
- Tests cover: dry-run file safety, write correctness for Go/JS/Python, idempotency,
  ambiguous context refusal, directory walk, unsupported extension skip, error conditions,
  and diff content sanity.
- Fixtures in `testdata/patch/` provide input/expected pairs for Go, JS, Python, and an
  ambiguous case that must produce zero changes.
- HTTP request header injection deferred: requires AST-level analysis per language SDK
  and is out of scope for v0.1.
- No automatic refactor of server state is implemented.

## Phase 6: Packaging and CI Hardening

Status: implemented.

Complexity: M

Dependencies: Phases 0-5 as relevant.

Work:

- Finalize GitHub Actions for test/build.
- Add `goreleaser` config for local snapshot builds.
- Finalize Dockerfile for scratch/distroless-style runtime if no CGO dependency appears.
- Add README usage section without overwriting existing human-owned content.

Done:

- `go.mod` declares `go 1.21` — the minimum version that covers all language features and
  standard-library packages used in production and test code (`slices` package, introduced
  in 1.21, is used in test files only).
- CI (`.github/workflows/ci.yml`) uses `go-version-file: go.mod` so the runner always
  matches the declared minimum. A separate `docker` job builds the image and smoke-tests it
  with `docker run --rm mcp-migrate:ci analyze --help` after `test` passes.
- Dockerfile uses `golang:1.21-alpine` as the build stage (matches `go.mod`) and
  `gcr.io/distroless/static-debian12` as the runtime stage (no shell, no libc). Build flags:
  `CGO_ENABLED=0 -trimpath` for a reproducible, dependency-free binary.
- GoReleaser (`.goreleaser.yml`) produces `linux/amd64`, `linux/arm64`, `darwin/amd64`,
  `darwin/arm64` archives with checksums. `CGO_ENABLED=0` and `-trimpath` are set.
- Docker image verified locally: `docker run --rm mcp-migrate:local analyze --help` prints
  the flag usage (exit 2 is normal for `flag.ContinueOnError` on `-h`/`--help`).

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
