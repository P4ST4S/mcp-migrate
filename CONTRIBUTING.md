# Contributing to mcp-migrate

`mcp-migrate` is a live conformance analyzer for the MCP `2026-07-28` spec. The most valuable contributions are those grounded in the spec text: new rule detections, corrections to existing severity or enforcement calls, and real-server evidence that exposes gaps in the current probe set.

## What is useful right now

**Rule corrections.** If a rule's severity, enforcement, or SEP reference diverges from the spec RC text, open an issue with the relevant spec passage. Every rule in `internal/rules/rules.go` traces to a row in `docs/SPEC_RULES.md`. Changes require updating both.

**Real-server findings.** If you run `mcp-migrate analyze` against a production MCP server and get a false positive, a false negative, or a finding that is misleading, that is exactly the feedback this project needs. Include the anonymized JSONL output and a description of the server's actual behavior.

**New probe coverage.** Rules that are in `docs/SPEC_RULES.md` but marked as not yet implemented in the Implementation Coverage section are the clearest next targets. The unimplemented rules include `x-mcp-header`, `mrtr-input-required`, `subscriptions-listen`, and several auth rules.

**Pending-verification reconciliation.** Several rules carry `status: pending-verification` because their underlying SEP is Accepted or Draft rather than Final. Once the final `2026-07-28` spec ships (2026-07-28), these rules need to be re-checked against the final text and either confirmed or corrected. This is time-sensitive work with direct user impact.

## What is not useful right now

- Refactoring for its own sake.
- Additions that are not traceable to a SEP or spec passage.
- New patch transformations for rules whose SEP is not yet Final — `patch` blocks these at runtime for a reason.
- Performance optimizations for the probe path — correctness is the only constraint at this stage.

## Ground rules for rules

Every rule must satisfy all of the following before being merged:

1. **Spec citation.** The rule traces to a specific SEP and/or a passage in the official spec changelog or draft. Record the primary source in `docs/SPEC_RULES.md`.

2. **Correct SEP status.** The `sep()` call in `internal/rules/rules.go` must reflect the actual SEP index status (`Final`, `Accepted`, `Draft`, `In-Review`). Do not mark a SEP as Final unless it is. Verification is `"verified"` only when status is `Final` and the SEP file exists in the repo.

3. **Correct enforcement.** Rules whose SEP is not `Final`, whose detection relies on differential inference rather than direct observation, or whose spec text is pending final reconciliation must have `Status: StatusPendingVerification` or `InferentialEvidence: true`. This is not optional — it directly affects what users can rely on.

4. **Fixture coverage.** New detections need a fixture that triggers the finding and, where relevant, a fixture that must not trigger it. Use `httptest.Server` for HTTP and Go helper processes for stdio. See `internal/analyze/live/http_test.go` for the pattern.

5. **Example regeneration.** If you add or change a rule that affects the output fixtures, regenerate them:

   ```sh
   UPDATE_EXAMPLES=1 go test ./internal/analyze/live
   ```

   Commit the updated `testdata/examples/` files with the rule change.

## Evidence strength

`mcp-migrate` distinguishes two categories of detection:

- **Direct observation**: a protocol artifact is present in a response — a header, an error code, a missing method. These findings can be `enforcement: "enforced"` when the underlying SEP is Final.
- **Differential inference**: a property is inferred by comparing two observations — list drift between probes, absence of a signal across multiple requests. These findings are always `enforcement: "report-only"` regardless of SEP status, because the inference is fallible (time-based content, pagination, eventually-consistent backends).

Set `InferentialEvidence: true` on any rule whose detection compares observations rather than directly reading a protocol artifact.

## Patch contributions

`patch` applies only to rules where `Autofixable: true` in the rule registry. Before adding a new transformation:

- Confirm the underlying SEP is Final. `patch` gates on `Status: StatusPendingVerification` at runtime and will refuse to apply the transformation otherwise.
- The substitution must be context-confirmed: a local signal (±2 lines, non-comment code) must confirm the code is in the right semantic context. The `distant_comment.go` fixture in `testdata/patch/ambiguous/` documents the class of false positive this prevents.
- Add input/expected fixture pairs in `testdata/patch/` for each supported language.
- The transformation must be idempotent.

## Development workflow

```sh
go test ./...       # run all tests
go build ./...      # confirm the binary builds
```

No external dependencies. The module is intentionally dependency-free — do not add `go.sum` entries.

For the patch package specifically, tests run against filesystem fixtures and a live registry lookup. The `opts()` helper in `patch_test.go` sets `AllowPending: true` for tests that exercise behavior rather than the pending-verification gate — use it consistently.

## Submitting changes

- Keep commits focused: one logical change per commit.
- Reference the relevant SEP or spec section in the commit message or PR description.
- If you are correcting a rule based on the final `2026-07-28` spec (post-2026-07-28), note the specific passage that changed and update `docs/SPEC_RULES.md` accordingly.
- Do not modify `LICENSE` or `.gitignore`.

## Asking questions

Open an issue. Tag it with the relevant rule ID if applicable.
