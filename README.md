# mcp-migrate

`mcp-migrate` analyse un serveur MCP, en stdio ou Streamable HTTP, et rapporte ce qui doit évoluer pour la spec MCP `2026-07-28`. La sortie est pensée pour l'automatisation: JSONL par défaut, ou Markdown pour revue humaine.

> **Status**
>
> Pré-release ciblant le release candidate MCP `2026-07-28`.
> Le RC a été verrouillé le 21 mai 2026 et la spec finale est attendue le 28 juillet 2026. Les règles de migration peuvent évoluer avec le RC et la reconciliation finale.
>
> Le projet est versionné `0.x`: l'interface CLI et le schéma JSONL peuvent encore casser.

## What Works Today

- `analyze` live pour Streamable HTTP.
- `analyze` live pour stdio.
- Probes read-only par défaut.
- Sortie JSONL, une finding par ligne.
- Rendu Markdown du même modèle de findings.
- Redaction des secrets dans les sorties.

Roadmap, non encore implémenté:

- Détecteur de state caché.
- Mode `patch`.
- Mode `watch`.
- Scan statique multi-langage.

## Install

Depuis le repo:

```sh
go build -o ./bin/mcp-migrate ./cmd/mcp-migrate
```

Ou pendant le développement:

```sh
go run ./cmd/mcp-migrate help
```

## Quick Start

Analyse d'un serveur Streamable HTTP:

```sh
mcp-migrate analyze --transport http --url http://localhost:3000/mcp
```

Analyse d'un serveur stdio:

```sh
mcp-migrate analyze --transport stdio --server-command "node ./server.js"
```

Filtrer la sortie JSONL avec `jq`:

```sh
mcp-migrate analyze --transport http --url http://localhost:3000/mcp \
  | jq -r 'select(.severity == "breaking") | [.rule, .severity, .message] | @tsv'
```

Obtenir un rapport Markdown:

```sh
mcp-migrate analyze --transport http --url http://localhost:3000/mcp --format markdown > report.md
```

Exemples de rapports générés par les fixtures de test:

- [`testdata/examples/http-compliant.md`](testdata/examples/http-compliant.md)
- [`testdata/examples/http-legacy.md`](testdata/examples/http-legacy.md)
- [`testdata/examples/http-mixed.md`](testdata/examples/http-mixed.md)
- [`testdata/examples/stdio-compliant.md`](testdata/examples/stdio-compliant.md)
- [`testdata/examples/stdio-legacy.md`](testdata/examples/stdio-legacy.md)
- [`testdata/examples/stdio-mixed.md`](testdata/examples/stdio-mixed.md)

## Safety Guarantees

Les probes sont read-only par défaut:

- `server/discover`
- `tools/list`
- `resources/list`
- `prompts/list`

`resources/read` n'est pas exécuté par défaut, car un vrai serveur peut attacher des effets de bord à une lecture: consommation, marquage comme lu, fetch distant, ou autre comportement applicatif. Pour l'autoriser explicitement:

```sh
mcp-migrate analyze --transport http --url http://localhost:3000/mcp --allow-resource-read
```

`mcp-migrate` n'envoie aucun `tools/call` mutant pendant l'analyse live actuelle.

Les secrets ne sont pas écrits dans le JSONL ni dans le Markdown: tokens, headers d'auth, variables d'environnement, stderr, response bodies, headers bruts, cookies, userinfo d'URL et paramètres de query sensibles sont masqués ou omis.

## Severity Legend

Cette légende est reprise du schéma JSONL:

- `breaking`: incompatible with a strict MCP `2026-07-28` peer. This does not mean the feature stops working on July 28, 2026.
- `deprecated`: still functional in MCP `2026-07-28`, but in the Deprecated lifecycle state. Deprecated features remain functional for at least 12 months before earliest removal eligibility.
- `warning`: operational risk or minor non-conformance that may affect portability, scaling, or future migration.
- `info`: informational modernization or interoperability suggestion.

En pratique, `breaking` signifie incompatible avec un peer strict `2026-07-28`, pas "casse le 28 juillet". Les features `deprecated` restent fonctionnelles pendant au moins 12 mois avant leur première éligibilité au retrait.

## Report Model

La sortie JSONL utilise le schéma `mcp-migrate/finding/v1`. Chaque ligne est un objet JSON indépendant:

```json
{"schema":"mcp-migrate/finding/v1","rule":"server-discover-required","sep":{"id":"SEP-2575","status":"Accepted","verification":"unverified","source":"https://github.com/modelcontextprotocol/modelcontextprotocol/blob/main/seps/2575-stateless-mcp.md"},"severity":"breaking","enforcement":"enforced","spec_target":"2026-07-28","source":{"mode":"live","ref":"http://localhost:3000/mcp"},"message":"Server does not expose the stateless server/discover RPC.","remediation":"Implement server/discover with supported versions, server capabilities, and server identity.","autofix":false,"status":"confirmed"}
```

Une sortie vide est valide: elle signifie qu'aucune finding n'a été émise.

Les références SEP non `Final`, ou dont le fichier SEP n'a pas été trouvé pendant la vérification, sont marquées `unverified` dans la sortie. Elles ne font pas autorité tant que la spec finale `2026-07-28` n'a pas été reconciliée.

Voir [`docs/REPORT_SCHEMA.md`](docs/REPORT_SCHEMA.md) pour le schéma complet.

## Positioning

`mcp-migrate` complète la suite de conformance officielle [`modelcontextprotocol/conformance`](https://github.com/modelcontextprotocol/conformance). La conformance répond principalement à une question pass/fail; `mcp-migrate` se concentre sur la migration: severity, remediation, signaux de compatibilité live, et à terme détection de state caché. Il ne remplace pas la suite de conformance officielle.

[`Janix-ai/mcp-validator`](https://github.com/Janix-ai/mcp-validator) est une autre référence utile dans l'écosystème; elle couvre des specs antérieures et n'est pas centrée sur la migration stateless `2026-07-28`.

## Documentation

- [`docs/SPEC_RULES.md`](docs/SPEC_RULES.md): règles MCP `2026-07-28`, sources, severities, questions ouvertes.
- [`docs/REPORT_SCHEMA.md`](docs/REPORT_SCHEMA.md): schéma JSONL et légende des severities.
- [`docs/PLAN.md`](docs/PLAN.md): plan d'implémentation phasé et statut.

## Development

```sh
go test ./...
go build ./...
```

Les tests live utilisent des fixtures locales: `httptest.Server` pour HTTP et des helper processes Go pour stdio.

## Feedback And Contributions

Les retours utiles pour cette phase:

- divergence observée entre une règle et la spec RC;
- sortie JSONL difficile à exploiter dans `jq`;
- finding trop sévère ou pas assez actionnable;
- serveur réel qui expose un cas de migration non couvert par les fixtures.

Gardez les changements focalisés: les règles doivent tracer vers [`docs/SPEC_RULES.md`](docs/SPEC_RULES.md), et toute référence SEP non finale doit rester `unverified`.

## License

Apache-2.0. See [`LICENSE`](LICENSE).
