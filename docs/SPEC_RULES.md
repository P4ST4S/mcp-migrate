# MCP 2026-07-28 Spec Rules

Status: verified against the 2026-07-28 release-candidate draft available on 2026-05-29. The final spec is scheduled for 2026-07-28, so entries backed by Draft, In-Review, or Accepted SEPs are intentionally marked for re-check before a v0.1 release.

Primary sources read:

- Official RC announcement: https://blog.modelcontextprotocol.io/posts/2026-07-28-release-candidate/
- Official 2026 roadmap: https://blog.modelcontextprotocol.io/posts/2026-mcp-roadmap/
- Official draft changelog: https://modelcontextprotocol.io/specification/draft/changelog
- Official draft caching utility page: https://modelcontextprotocol.io/specification/draft/server/utilities/caching
- Full compare URL named by the changelog: https://github.com/modelcontextprotocol/specification/compare/2025-11-25...draft
- SEP index: https://modelcontextprotocol.io/seps
- Deprecated registry: https://modelcontextprotocol.io/specification/draft/deprecated

Severity model used by this project:

- `breaking`: a strict 2026-07-28 peer can reject or fail the behavior.
- `deprecated`: still works in 2026-07-28, but the feature is in the lifecycle-policy Deprecated state.
- `warning`: operational risk or migration concern without direct rejection.
- `info`: useful modernization signal.

## Confirmed Rules

| Rule id | SEP | Status | Before 2026-07-28 | 2026-07-28 behavior | Severity | Rationale |
| --- | --- | --- | --- | --- | --- | --- |
| `initialize-handshake-removed` | SEP-2575 | Accepted | Clients established protocol version, client info, and capabilities through `initialize` followed by `notifications/initialized`. | The handshake is removed. Every request carries `io.modelcontextprotocol/protocolVersion`, `io.modelcontextprotocol/clientInfo`, and `io.modelcontextprotocol/clientCapabilities` in `_meta`; `server/discover` is the stateless capability discovery RPC. | `breaking` | The changelog says the handshake is removed and missing per-request fields are malformed. |
| `server-discover-required` | SEP-2575 | Accepted | Server capabilities were learned from `initialize` response. | Servers MUST implement `server/discover`; clients MAY call it before other requests, and STDIO clients SHOULD use it as a backward-compat probe when supporting old and new versions. | `breaking` | A 2026-07-28 analyzer can treat missing `server/discover` as direct non-compliance for target servers. |
| `protocol-version-per-request` | SEP-2575 | Accepted | Protocol version was negotiated once in `initialize`. | Each request includes `_meta["io.modelcontextprotocol/protocolVersion"]`; HTTP also requires `MCP-Protocol-Version`, matching `_meta`, or the server returns HTTP 400. Unsupported versions return error code `-32004` with supported/requested versions. | `breaking` | Required on every request; mismatch or missing value is rejectable. |
| `client-info-capabilities-per-request` | SEP-2575 | Accepted | Client identity and capabilities were session state. | `_meta["io.modelcontextprotocol/clientInfo"]` and `_meta["io.modelcontextprotocol/clientCapabilities"]` are required on every request. Servers MUST NOT infer capabilities from previous requests. | `breaking` | Missing required fields are invalid params and HTTP 400 for HTTP. |
| `mcp-session-id-removed` | SEP-2567 | Final | Streamable HTTP servers could create a protocol session via `Mcp-Session-Id`, and subsequent requests carried that header. | Protocol-level sessions and the `Mcp-Session-Id` header are removed. Cross-call state must use explicit application handles in normal tool arguments/results. | `breaking` | Strict 2026-07-28 clients and servers should not require protocol sessions. Hidden session dependence is also this project's feature-signature risk. |
| `session-dependent-lists-removed` | SEP-2567, SEP-2549 | Final / Accepted | `tools/list`, `resources/list`, and `prompts/list` could vary over a session or connection lifetime. | List results have no per-session or per-connection scope. They may vary by auth or time, but not as a side effect of prior requests on the connection. | `breaking` | Session-scoped dynamic tool lists are no longer a valid protocol pattern. |
| `explicit-state-handles` | SEP-2567 | Final | Servers often stored carts, browsers, transactions, etc. behind a session/process. | Stateful workflows should mint explicit opaque handles and require them in subsequent tool arguments. | `warning` | This is guidance, not a wire type. It is operationally critical for horizontal scaling but not mechanically enforceable in static code. |
| `http-standard-headers` | SEP-2243 | Final | HTTP intermediaries had to inspect JSON-RPC bodies for method/tool/resource/prompt routing. | Streamable HTTP POST requests require `Mcp-Method` for all requests/notifications and `Mcp-Name` for `tools/call`, `resources/read`, and `prompts/get`. Servers processing bodies MUST reject header/body disagreement. | `breaking` | Required for compliance and strict servers reject mismatches. |
| `x-mcp-header` | SEP-2243 | Final | Tool parameters were only in request bodies. | Tools may annotate primitive parameters with `x-mcp-header`; HTTP clients MUST mirror valid annotated values to `Mcp-Param-{name}` headers and reject invalid tool definitions from `tools/list`. | `breaking` | Required client support for Streamable HTTP when servers use the annotation. |
| `mrtr-input-required` | SEP-2322, SEP-2260 | Accepted / Final | Server-initiated `roots/list`, `sampling/createMessage`, and `elicitation/create` could be sent on an SSE stream while work continued. Association with an originating request was recommended in places but not uniformly required. | Server-to-client requests must be associated with an originating client request and are represented through MRTR: `inputRequests`, `inputResponses`, `requestState`, and a result discriminator for incomplete work. Standalone unsolicited server requests are prohibited. | `breaking` | The old standalone/SSE-driven request pattern is explicitly disallowed. |
| `request-scoped-server-requests-only` | SEP-2260 | Final | Standalone server-initiated sampling, elicitation, and roots requests could appear on independent streams. | `roots/list`, `sampling/createMessage`, and `elicitation/create` MUST only occur in association with an originating client request. | `breaking` | This is a normative MUST NOT for standalone server-initiated requests. |
| `subscriptions-listen` | SEP-2575 | Accepted | Streamable HTTP used an optional GET SSE endpoint; `resources/subscribe` and `resources/unsubscribe` existed. | HTTP GET endpoint, `resources/subscribe`, and `resources/unsubscribe` are replaced by `subscriptions/listen`, a POST response stream with explicit notification opt-ins. | `breaking` | Old endpoints/methods are removed in the changelog and SEP-2575. |
| `ping-removed` | SEP-2575 | Accepted | Either party could use `ping` for MCP-level liveness. | `ping` is removed in both directions; use normal RPCs and transport-level health mechanisms. | `breaking` | Removed RPC. |
| `logging-setlevel-replaced` | SEP-2575, SEP-2577 | Accepted / Final | Clients set log level through `logging/setLevel`; servers emitted `notifications/message` according to session state. | SEP-2575 removes `logging/setLevel`; desired log level is per-request `_meta["io.modelcontextprotocol/logLevel"]`, and servers MUST NOT emit log notifications without it. | `breaking` with `status: pending-verification` | Official changelog lists `logging/setLevel` removal, but the RC blog also says Logging is annotation-only deprecated and continues to work. Re-check final spec before enforcing as breaking. |
| `roots-deprecated` | SEP-2577, SEP-2596 | Final / Draft | Roots were a core feature. | Roots are Deprecated as of 2026-07-28; migration is tool parameters, resource URIs, or server configuration. Earliest removal: first revision on or after 2027-07-28. | `deprecated` | Deprecated registry confirms the feature remains in spec during the deprecation window. |
| `sampling-deprecated` | SEP-2577, SEP-2596 | Final / Draft | Sampling was a core feature. | Sampling is Deprecated as of 2026-07-28; migration is direct integration with LLM provider APIs. Earliest removal: first revision on or after 2027-07-28. | `deprecated` | Annotation-only deprecation; no immediate wire removal for the feature as a whole. |
| `logging-deprecated` | SEP-2577, SEP-2596 | Final / Draft | Logging was a core feature. | Logging is Deprecated as of 2026-07-28; migration is `stderr` for stdio and OpenTelemetry for structured observability. Earliest removal: first revision on or after 2027-07-28. | `deprecated` with `status: pending-verification` | Registry confirms deprecation; `logging/setLevel` removal needs final reconciliation. |
| `http-sse-transport-deprecated` | SEP-2596 | Draft | HTTP+SSE was deprecated before the lifecycle policy but without a precise lifecycle-state registry. | Reclassified as Deprecated; migration target is Streamable HTTP. Earliest removal is three months after SEP-2596 reaches Final. | `deprecated` | Lifecycle registry makes the transition explicit. |
| `includecontext-values-deprecated` | SEP-2596 | Draft | Sampling `includeContext: "thisServer"` / `"allServers"` were soft-deprecated in 2025-11-25. | Reclassified as Deprecated; migration is omit the field or use `"none"`. Removal follows Sampling. | `deprecated` | Registry explicitly tracks the values. |
| `tasks-core-to-extension` | SEP-2663 | Final | Experimental `tasks/*` lived in the 2025-11-25 core protocol, including `tasks/result` and `tasks/list`. | Tasks move to official extension `io.modelcontextprotocol/tasks`; lifecycle uses `tools/call` returning `resultType: "task"`, then `tasks/get`, `tasks/update`, and `tasks/cancel`. `tasks/list` is removed; `tasks/result` is replaced. | `breaking` | Core `tasks/*` callers must migrate to extension negotiation and the new method set. |
| `extensions-capabilities` | SEP-2133 | Final | Extensions had no formal framework. | `ClientCapabilities` and `ServerCapabilities` include an `extensions` map keyed by reverse-DNS extension IDs. | `info` | Important for Tasks/MCP Apps detection; non-support is not by itself an error unless an extension is required. |
| `cacheable-results-required` | SEP-2549 | Accepted | List/read results had no protocol TTL/cache scope. | `tools/list`, `prompts/list`, `resources/list`, `resources/read`, and `resources/templates/list` results require `ttlMs` and `cacheScope`. | `breaking`, `enforcement: report-only` while SEP-2549 is unverified | The official draft caching page says servers MUST include caching hints on affected results and defines `ttlMs` / `cacheScope`; because SEP-2549 is Accepted rather than Final, findings are report-only until final-spec reconciliation. |
| `trace-context-meta` | SEP-414 | Final | Trace propagation through `_meta` was an ecosystem convention. | `_meta` keys `traceparent`, `tracestate`, and `baggage` are documented for W3C Trace Context/Baggage propagation. | `info` | Interop/observability improvement, not a migration blocker. |
| `resource-not-found-code` | SEP-2164 | Draft | Spec recommended `-32002` for missing resources; SDKs varied. | Missing resource errors should use JSON-RPC `-32602` Invalid Params; data SHOULD include the missing `uri`; empty `contents` is not a not-found response. | `breaking` | The official changelog states the code changes to `-32602`; patch can safely replace literal `-32002` when the context is resource-not-found. |
| `json-schema-2020-12-tools` | SEP-2106 | Draft | Tool `inputSchema`/`outputSchema` were restricted shapes; `structuredContent` was effectively object-shaped. | `inputSchema` still requires root `type: "object"` but supports JSON Schema 2020-12 keywords; `outputSchema` is unrestricted JSON Schema; `structuredContent` can be any JSON value. | `warning` | Mostly loosening, but validators that reject composition/array output will produce false negatives. |
| `auth-iss-validation` | SEP-2468 | In-Review | MCP auth did not require clients to validate RFC 9207 `iss` in authorization responses. | Clients record the expected issuer from validated metadata and validate `iss` when present; if metadata advertises `authorization_response_iss_parameter_supported: true`, absence is rejected. Authorization servers SHOULD emit `iss`; future spec may make that MUST. | `breaking` for client validation, `warning` for AS emission | A strict client must reject mismatches now; AS omission is not yet universally rejected unless advertised support says it should be present. |
| `auth-application-type-dcr` | SEP-837 | Merged PR, no indexed SEP file found | Dynamic Client Registration could omit OIDC `application_type`, causing default `"web"` behavior. | MCP clients MUST provide an appropriate `application_type`; native apps/CLI/localhost SHOULD use `"native"`, remote web apps SHOULD use `"web"`. | `breaking` | Draft authorization spec uses MUST for clients during DCR. |
| `auth-server-binding` | SEP-2352 | Merged PR, no indexed SEP file found | Clients could risk reusing persisted DCR/preregistered credentials across authorization servers. | Clients MUST bind such credentials to the issuing AS `issuer` and MUST re-register when protected-resource metadata points to a different AS. Client ID Metadata Document client IDs remain portable. | `breaking` | Security requirement; credential reuse across issuers must be rejected. |
| `auth-step-up-scopes` | SEP-2350 | Merged PR, no indexed SEP file found | Scope challenge semantics around existing grants were unclear. | Clients treat `WWW-Authenticate` challenge scopes as authoritative for the current operation and SHOULD request them alongside previously granted scopes during step-up. | `warning` | Incorrect behavior can drop prior permissions or over-request; not always directly observable from an MCP server probe. |
| `auth-well-known-suffix` | SEP-2351 | Merged PR, no indexed SEP file found | MCP-specific authorization-server well-known suffix behavior was unclear. | MCP uses RFC 8414 `oauth-authorization-server` and OIDC discovery endpoint probing; clients MUST validate discovered metadata issuer identity. | `breaking` | Discovery/issuer validation failures must stop auth use. |
| `auth-refresh-token-guidance` | SEP-2207 | Accepted | Refresh token guidance for OIDC-style ASs was underspecified. | Clients that want refresh tokens SHOULD include `refresh_token` in metadata grant types, MAY request `offline_access` if supported, and MUST NOT assume refresh tokens are issued; protected resources SHOULD NOT advertise `offline_access` as a resource requirement. | `info` | Guidance for auth implementations; not a core server conformance rule for v0.1 live probing. |

## Pending Verification Notes

- The final URL for the 2026-07-28 changelog is currently the draft URL: `https://modelcontextprotocol.io/specification/draft/changelog`. Re-check after 2026-07-28 for a versioned `/specification/2026-07-28/changelog` page.
- SEP-2575, SEP-2549, and SEP-2322 are Accepted rather than Final in the current SEP index; their normative text can still change before final release.
- Cacheability MUST vs SHOULD was re-checked on 2026-05-29 against the official draft caching utility page. The page says servers MUST include caching hints on affected results, while client freshness behavior around positive, absent, or negative `ttlMs` is mostly SHOULD/MAY guidance. Therefore the detector remains valid, but enforcement remains `report-only` while SEP-2549 is not Final.
- SEP-2164 and SEP-2106 are Draft, and SEP-2468 is In-Review, despite the RC blog/changelog presenting their effects as part of the RC. Treat exact detector behavior as pending until final spec freeze.
- The RC blog example uses `resultType: "inputRequired"`, while SEP-2322 uses `resultType: "input_required"`. The analyzer must support both while reporting the mismatch as `status: pending-verification` until the final schema resolves it.
- SEP-2322 names the request type `ElicitRequest` and method `elicitation/create`; the blog describes an `InputRequiredResult` object with an item `"type": "elicitation"`. Prefer the schema/SEP for implementation, but retain a note in findings when servers use the blog shape.
- The blog says Roots, Sampling, and Logging are annotation-only deprecations that continue to work, while SEP-2575 and the changelog remove `logging/setLevel`, top-level `roots/list`, and several related flows. The rule engine should split "feature deprecated" from "specific method removed" and mark logging method enforcement pending final verification.
- SEP-2663 says it updates Tasks for a `2026-06-30` specification, while the RC announcement and changelog target `2026-07-28`. Treat that date as editorial drift unless corrected in the final SEP/spec.
- SEP-414 was spot-checked on 2026-05-29: despite its low number relative to the 2026 SEPs, `seps/414-request-meta.md` exists and the SEP index marks it `Final`.

## Implementation Coverage

Last updated: 2026-05-29.

Output model and rule registry:

- Implemented: each finding carries `schema`, `rule`, structured `sep`, `severity`, `enforcement`, `spec_target`, `source`, `message`, optional `detail`, optional `remediation`, `autofix`, and `status`.
- Implemented: `sep.verification` is `verified` only when the SEP status is `Final` and the SEP file has been found; otherwise it is `unverified`.
- Implemented: rules with `status: pending-verification` evaluate as `enforcement: "report-only"`.
- Implemented: rules whose `sep.verification` is `unverified` evaluate as `enforcement: "report-only"` by default, unless a rule carries an explicit documented enforcement override. No current rule uses that override.
- Implemented: Markdown reports always render the severity legend before findings.
- Documented in `docs/REPORT_SCHEMA.md`.

Live HTTP analyzer coverage:

| Rule id | Phase 2 status | Notes |
| --- | --- | --- |
| `server-discover-required` | implemented | Probes `server/discover`; missing, errored, or malformed discovery maps to a finding. |
| `initialize-handshake-removed` | partial | HTTP does not send `initialize`. Text mentions are downgraded to `initialize-text-heuristic` warning; stdio sends `initialize` only in an isolated child process after `server/discover` fails. |
| `protocol-version-per-request` | partial | Sends a read-only `server/discover` request with mismatched HTTP protocol header vs `_meta`; acceptance maps to a finding. |
| `client-info-capabilities-per-request` | implemented | Sends read-only `tools/list` without `_meta`; acceptance maps to a finding. |
| `mcp-session-id-removed` | implemented | Detects `Mcp-Session-Id` response header or redacted response-body mention. |
| `http-standard-headers` | implemented | Sends read-only `tools/list` without `Mcp-Method` and with mismatched `Mcp-Method`; acceptance maps to findings. |
| `initialize-text-heuristic` | implemented | HTTP-only weak signal for response text mentioning initialize; warning severity only. |
| `cacheable-results-required` | implemented | Checks accepted `tools/list`, `resources/list`, and `prompts/list` by default. `resources/read` is checked only when explicitly opted in. |
| `resource-not-found-code` | implemented for opt-in `resources/read` | When `--allow-resource-read` is set, maps legacy `-32002` from `resources/read` to a pending-verification report-only finding. |
| `session-dependent-lists-removed` | implemented for read-only list drift | Repeats list probes on the existing HTTP client and a fresh client/connection, canonicalizes results, and reports drift not explained by explicit handle fields. |
| `explicit-state-handles` | implemented for stdio process-lifetime drift | Repeats list probes in the analyzer-owned stdio process and reports process-lifetime drift as a warning-style finding. |
| `x-mcp-header` | not started | Deferred; requires tool schema inspection and header mirroring validation. |

Phase 2 safety posture:

- Probes are read-only by default: `server/discover` and list methods only.
- Hidden-state detection repeats read-only list methods and compares canonicalized results; it does not send `tools/call`.
- `resources/read` is opt-in (`--allow-resource-read`) because real servers may attach side effects to reads.
- `tools/call` is not sent in Phase 2.
- `--allow-mutating-probes` exists as an explicit opt-in, but no mutating probes are implemented yet.
- Raw probe observations are kept in internal `HTTPTrace`/`HTTPObservation` structs; rules convert observations to findings and do not perform network I/O.
- Output masks URL userinfo and sensitive query values, and never emits authorization headers, raw response headers, raw response bodies, or network error strings.

Live STDIO analyzer coverage:

| Rule id | Phase 3 status | Notes |
| --- | --- | --- |
| `server-discover-required` | implemented | Probes `server/discover`; missing, errored, timeout, or malformed discovery maps to a finding. |
| `initialize-handshake-removed` | implemented | If `server/discover` fails, stdio sends a legacy `initialize` probe to the isolated child process; success maps to a finding. |
| `client-info-capabilities-per-request` | implemented | Sends read-only `tools/list` without `_meta`; acceptance maps to a finding. |
| `cacheable-results-required` | implemented | Checks accepted `tools/list`, `resources/list`, and `prompts/list` results for `ttlMs` and `cacheScope`. |
| `explicit-state-handles` | implemented for process-lifetime drift | Repeats list probes in the same stdio process and reports canonical result drift as process-lifetime hidden state. |

Phase 3 safety posture:

- The analyzer launches and owns the stdio process, applies a timeout to each RPC, cancels/kills the process on timeout, and waits for process exit.
- Stderr is drained with a fixed memory bound; stderr content is never emitted.
- Environment variables are inherited for process execution but never emitted in JSONL or Markdown.
- Command args are redacted in `source.ref` when they look sensitive.
- No `tools/call` is sent.
- Hidden-state detection repeats read-only list methods in the analyzer-owned process; it does not send `tools/call`.
- The stdio `initialize` probe is allowed because it only affects the disposable child process; this is intentionally different from HTTP remote probing.
- Raw probe observations are kept in internal `STDIOTrace`/`STDIOObservation` structs; rules convert observations to findings and do not perform process I/O.

Phase 5 patch engine coverage:

| Transformation | Status | Notes |
| --- | --- | --- |
| `resource-not-found-code` (`-32002` → `-32602`) | implemented | Context-confirmed replacement in Go, JS/TS, Python. Occurrences without a `resources/read` or `resource not found` signal within 12 lines are skipped and reported. Idempotent. |
| HTTP header injection (`Mcp-Method`, `Mcp-Name`) | not started | Requires per-language AST analysis of SDK call sites; deferred to a later patch revision. |

Patch safety posture:

- Dry-run is default; `--write` required for filesystem mutations.
- Context window detection prevents false positives in generic error-handler code.
- Evidence-strength principle: only context-confirmed occurrences are rewritten; ambiguous ones are reported as `Skipped`, never silently dropped or guessed.
- Patches are idempotent: a second pass on an already-patched file produces zero changes.
- Supported file extensions: `.go`, `.js`, `.ts`, `.jsx`, `.tsx`, `.py`.

Review artifacts:

- `testdata/examples/http-compliant.{jsonl,md}`
- `testdata/examples/http-legacy.{jsonl,md}`
- `testdata/examples/http-mixed.{jsonl,md}`
- `testdata/examples/http-resource-not-found.{jsonl,md}`
- `testdata/examples/http-stateful-lists.{jsonl,md}`
- `testdata/examples/http-explicit-handle-lists.{jsonl,md}`
- `testdata/examples/stdio-compliant.{jsonl,md}`
- `testdata/examples/stdio-legacy.{jsonl,md}`
- `testdata/examples/stdio-mixed.{jsonl,md}`
- `testdata/examples/stdio-stateful-lists.{jsonl,md}`
- `testdata/examples/stdio-explicit-handle-lists.{jsonl,md}`
- `testdata/patch/go/resource_handler/` — Go input/expected pair
- `testdata/patch/js/` — JS input/expected pair
- `testdata/patch/python/` — Python input/expected pair
- `testdata/patch/ambiguous/` — must produce zero changes (no context signal)

## Competitive/Community Observations Read

- Janix `mcp-validator` focuses on protocol compliance test suites for STDIO and HTTP and currently advertises coverage through 2025-era protocol behavior such as initialization, sessions, ping, and 2025-06-18 features. It is a useful baseline, but it does not center 2026-07-28 stateless migration or hidden session-state detection.
- Community migration writeups consistently frame the durable migration work as moving session/process state into explicit handles, adding per-request metadata and headers, and designing live conformance probes instead of relying only on static code patterns.
- The official C# SDK guidance already recommends explicitly choosing stateless mode when sessions are not needed; this supports making hidden-state detection the key product differentiator rather than treating codemods as the core moat.
