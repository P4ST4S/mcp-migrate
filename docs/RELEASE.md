# Release Readiness — `mcp-migrate`

Two distinct tracks. **Do not confuse “PLAN phases checked” with “releasable.”** Finished phases = the code builds, tests, and runs. Releasable = the guardrails are shipped, tested against real servers, and the scope is frozen.

Guiding principle: **the tool’s maturity follows the spec’s maturity.** We are targeting an RC that can move until July 28, 2026 → as long as the spec is not final, we only ship honest pre-releases.

---

## Track A — `0.1.0-rc.1` (pre-release, to ship DURING the RC window)

Goal: first out, real implementer feedback, before the official codemods. Ship **as soon as the boxes below are green**, without waiting for July 28.

### Blocking — baseline quality
- [ ] `go build ./...` and `go test ./...` are green in CI.
- [ ] `analyze` works over **HTTP** and **stdio** (not just one of them).
- [ ] Valid, pipeable JSONL output (`mcp-migrate analyze ... | jq` does not break, including on 0 findings).
- [ ] Stable Markdown rendering (deterministic order, grouped by severity).

### Blocking — guardrails (the two go conditions)
- [ ] **Severity legend** rendered at the top of every Markdown report and documented in the JSONL schema: `breaking` = incompatible with a strict 2026-07-28 peer, **not** “breaks on July 28”; `deprecated` features remain functional for ≥ 12 months.
- [ ] **`unverified` tag** on any SEP number whose `status` ≠ `Final` or whose SEP file was not found. No unverified SEP is surfaced as authoritative.
- [ ] Any `pending-verification` rule is **non-fatal (report-only)**: it does not produce a definitive pass/fail verdict while the spec is not final.

### Blocking — testing beyond fixtures
- [ ] Tested against **at least 2 real official SDK servers** (HTTP + stdio), not just `httptest` fixtures.
- [ ] Tested against **1–2 existing public servers**.
- [ ] Hidden-state detector: false positives identified and documented; an explicit returned handle is not flagged by mistake.
- [ ] **Read-only probes confirmed non-mutating** on a real server (the default read/list/discover holds in real conditions, the opt-in mutant tool call is clearly explicit).

### Blocking — communication
- [ ] README states plainly: “targets the **RC** 2026-07-28, rules evolve with the RC, pre-release”.
- [ ] Version tagged `0.1.0-rc.1` (bump `-rc.N` with every RC movement).
- [ ] Announced scope = live analyze + hidden-state + safe patch. Explicit non-goals: no `watch`, no multi-language scan, no semantic state refactor. (So nobody can complain about deliberate omissions.)

### Nice to have (can slip to `rc.2`)
- [ ] Downloadable GoReleaser binaries (snapshot).
- [ ] Working Docker image (`mcp-migrate analyze --help`).
- [ ] A few clean `--help` outputs and an end-to-end example in the README.

---

## Track B — `0.1.0` (stable, AFTER July 28, 2026)

Goal: align the stable release with the final spec. Ship only once the spec is ratified **and** reconciliation is done.

### Spec prerequisites
- [ ] Final 2026-07-28 spec published.
- [ ] **Complete reconciliation**: every `pending-verification` rule rechecked against the final changelog; status updated (`Final` / removed / corrected).
- [ ] Spot-check completed for doubtful SEP numbers (`SEP-414` trace context, and the auth SEPs with “no indexed SEP file found”). Remove the `unverified` tag only for those confirmed.
- [ ] Known divergence cases resolved against the final text:
  - [ ] `logging/setLevel` removed vs Logging deprecated (breaking vs deprecated).
  - [ ] MRTR discriminator: `inputRequired` vs `input_required`.
  - [ ] `cacheable-results-required` actually MUST (breaking) or SHOULD (warning).
  - [ ] `x-mcp-header` actually MUST on the client side.
  - [ ] SEP-2663 date drift (`2026-06-30`) corrected or confirmed editorial.

### Product prerequisites
- [ ] No `breaking` rule depends on a non-final source anymore.
- [ ] Integration test suite green against updated real servers aligned to the final spec.
- [ ] Packaging finalized: GoReleaser release (not just snapshot), published Docker image.
- [ ] CHANGELOG describing what moves from `rc` to stable.
- [ ] `0.1.0` promise frozen and documented (what is covered / what is not).

### SemVer reminder
- [ ] We remain on `0.x`: the CLI API and JSONL schema may still break between minors. Document that interface stability is only promised starting at `1.0.0`.

---

## Definition of Done vs Releasable (summary)

| | PLAN phases checked | `0.1.0-rc.1` | `0.1.0` stable |
|---|---|---|---|
| Build + test + run | ✅ | ✅ | ✅ |
| Guardrails (legend, `unverified`, report-only) | — | ✅ | ✅ |
| Tested against real servers | — | ✅ | ✅ (final spec) |
| Final spec + `pending` reconciliation | — | — | ✅ |
| Release packaging + published Docker | — | partial | ✅ |

**Simple rule:** ship `0.1.0-rc.x` as soon as the middle column is green — do not wait for it. Keep `0.1.0` clean for after July 28.