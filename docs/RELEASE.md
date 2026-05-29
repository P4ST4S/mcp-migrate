# Release Readiness — `mcp-migrate`

Deux pistes distinctes. **Ne pas confondre « phases du PLAN cochées » avec « releasable ».** Les phases finies = le code compile, teste, tourne. Releasable = les garde-fous sont livrés, testé contre de vrais serveurs, et le périmètre est figé.

Principe directeur : **la maturité du tool suit la maturité de la spec.** On cible un RC qui peut bouger jusqu'au 28 juillet 2026 → tant que la spec n'est pas finale, on ne sort que des pré-releases honnêtes.

---

## Piste A — `0.1.0-rc.1` (pré-release, à sortir PENDANT la fenêtre RC)

Objectif : premier arrivé, feedback réel des implémenteurs, avant les codemods officiels. À sortir **dès que les cases ci-dessous sont vertes**, sans attendre le 28 juillet.

### Bloquant — qualité de base
- [ ] `go build ./...` et `go test ./...` verts en CI.
- [ ] `analyze` fonctionne en **HTTP** et en **stdio** (pas seulement l'un des deux).
- [ ] Sortie JSONL valide et pipe-able (`mcp-migrate analyze ... | jq` ne casse pas, y compris sur 0 finding).
- [ ] Rendu Markdown stable (ordre déterministe, groupé par severity).

### Bloquant — garde-fous (les deux conditions du go)
- [ ] **Légende severity** rendue en tête de chaque rapport Markdown et documentée dans le schéma JSONL : `breaking` = incompatible avec un peer strict 2026-07-28, **pas** « casse le 28 juillet » ; les features `deprecated` restent fonctionnelles ≥ 12 mois.
- [ ] **Tag `unverified`** sur tout numéro de SEP dont le `status` ≠ `Final` ou dont le fichier SEP n'a pas été trouvé. Aucun SEP non vérifié surfacé comme autoritatif.
- [ ] Toute règle `pending-verification` est **non-fatale (report-only)** : elle ne produit pas de verdict pass/fail définitif tant que la spec n'est pas finale.

### Bloquant — test hors fixtures
- [ ] Testé contre **au moins 2 serveurs SDK officiels** réels (HTTP + stdio), pas seulement les fixtures `httptest`.
- [ ] Testé contre **1–2 serveurs publics** existants.
- [ ] Hidden-state detector : faux positifs identifiés et documentés ; un handle explicite retourné n'est pas flaggé à tort.
- [ ] **Probes read-only confirmées non mutantes** sur un vrai serveur (le défaut read/list/discover tient en conditions réelles, l'opt-in tool-call mutant est bien explicite).

### Bloquant — communication
- [ ] README dit noir sur blanc : « cible le **RC** 2026-07-28, les règles évoluent avec le RC, pré-release ».
- [ ] Version taguée `0.1.0-rc.1` (bump `-rc.N` à chaque mouvement du RC).
- [ ] Périmètre annoncé = analyze live + hidden-state + patch sûr. Non-goals explicites : pas de `watch`, pas de scan multi-langage, pas de refacto sémantique du state. (Pour ne pas se voir reprocher des absences assumées.)

### Souhaitable (peut glisser en `rc.2`)
- [ ] Binaires GoReleaser (snapshot) téléchargeables.
- [ ] Image Docker qui tourne (`mcp-migrate analyze --help`).
- [ ] Quelques `--help` propres et un exemple de bout en bout dans le README.

---

## Piste B — `0.1.0` (stable, APRÈS le 28 juillet 2026)

Objectif : caler la release stable sur la spec finale. À ne sortir qu'une fois la spec ratifiée **et** la passe de réconciliation faite.

### Pré-requis spec
- [ ] Spec finale 2026-07-28 publiée.
- [ ] **Réconciliation complète** : chaque règle `pending-verification` repassée contre le changelog final ; statut mis à jour (`Final` / supprimée / corrigée).
- [ ] Spot-check terminé des numéros SEP douteux (`SEP-414` trace context, et les SEP auth « no indexed SEP file found »). Tag `unverified` retiré uniquement pour ceux confirmés.
- [ ] Cas de divergence connus tranchés contre le texte final :
  - [ ] `logging/setLevel` retiré vs Logging déprécié (breaking vs deprecated).
  - [ ] Discriminateur MRTR : `inputRequired` vs `input_required`.
  - [ ] `cacheable-results-required` réellement MUST (breaking) ou SHOULD (warning).
  - [ ] `x-mcp-header` réellement MUST côté client.
  - [ ] Dérive de date SEP-2663 (`2026-06-30`) corrigée ou confirmée éditoriale.

### Pré-requis produit
- [ ] Plus aucune règle `breaking` ne s'appuie sur une source non finale.
- [ ] Suite de tests d'intégration verte contre serveurs réels mis à jour vers la spec finale.
- [ ] Packaging finalisé : GoReleaser release (pas seulement snapshot), image Docker publiée.
- [ ] CHANGELOG décrivant ce qui passe de `rc` à stable.
- [ ] Promesse de `0.1.0` figée et documentée (ce qui est couvert / ce qui ne l'est pas).

### Garde rappel SemVer
- [ ] On reste en `0.x` : l'API CLI et le schéma JSONL peuvent encore casser entre mineures. Documenter que la stabilité d'interface n'est promise qu'à partir de `1.0.0`.

---

## Definition of Done vs Releasable (résumé)

| | Phases PLAN cochées | `0.1.0-rc.1` | `0.1.0` stable |
|---|---|---|---|
| Compile + teste + tourne | ✅ | ✅ | ✅ |
| Garde-fous (légende, `unverified`, report-only) | — | ✅ | ✅ |
| Testé contre vrais serveurs | — | ✅ | ✅ (spec finale) |
| Spec finale + réconciliation `pending` | — | — | ✅ |
| Packaging release + Docker publié | — | partiel | ✅ |

**Règle simple :** sors le `0.1.0-rc.x` dès que la colonne du milieu est verte — ne l'attends pas. Garde le `0.1.0` propre pour l'après-28-juillet.