<!-- What does this PR do, and why? One paragraph is enough. -->

<!-- If this closes an issue: "Closes #N" anywhere in the description triggers auto-close. -->

<!-- For rule changes: which SEP or spec passage justifies this? -->

## Checklist

- [ ] `go test ./...` passes
- [ ] Rule change traces to a SEP or spec passage in `docs/SPEC_RULES.md`
- [ ] Inferential detection (comparing two probes) uses `InferentialEvidence: true`
- [ ] Non-Final SEP uses `Status: StatusPendingVerification`
- [ ] New fixture added for both the positive and the no-false-positive case
- [ ] `UPDATE_EXAMPLES=1 go test ./internal/analyze/live` run and examples committed (if output changed)
