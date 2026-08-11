# 53 Oracle explicit schema versioning

- **Author**: Patrick Dotson
- **Date**: 2026-08-11
- **Related**: [RFC 0033 - Oracle migrations](0033-oracle-migrations.md),
  [RFC 0042 - Core structure refactor](0042-core-structure-refactor.md),
  [RFC 0048 - Oracle predecessor-chain versioning](0048-oracle-predecessor-chain-versioning.md)

## 0 Summary

Oracle's schema history is a whole-tree copy: once per release, `oracle snapshot`
duplicates every `.oracle` file into `schemas/snapshots/vN/`, and that folder serves as
the diffing baseline for migration generation and for the predecessor-chain alias split
(RFC 0048 §4.0). This RFC deletes the snapshot system and replaces it with hand-owned
per-resource version files:

```
schemas/synnax/versions/channel/
  v0.oracle
  v1.oracle    # full declarations for types changed at v1; aliases for the rest
```

A version file enumerates the resource's complete persisted namespace at that version —
changed types as full declarations, unchanged types as one-line aliases to their definer
(`Key = v0.Key`), absence meaning removal. Version files are the source of truth for
every persisted shape, and developers author them directly: edit the current file for a
version that has not shipped, write `v(N+1).oracle` for one that has.

The live schema file inverts: its versioned content is generated from the chain, and
developers annotate it with current-only concerns — output paths, language bindings,
validation tags, wire-only types. A check gate holds the two representations in
agreement at declaration granularity. Frozen `versions/vN` Go packages and migration
auto-copies are regenerated, checked outputs of the chain, and the two-tier baseline of
RFC 0048 §4.0 collapses to one mechanism.

---

## 1 Motivation

The snapshot system is dead weight, and the reasons are structural, not accidental:

1. **History rots in a dialect nothing maintains.** Every checked-in snapshot (v53–v56)
   predates RFC 0043's optionality syntax and no longer parses; all predate the
   `schemas/{synnax,arc,x}` layout. `versioning.AliasSplit` short-circuits on every one
   of them, and `oracle migrate`'s diffing has been dormant since the grammar moved on.
   The alias split already runs entirely on the frozen-generated-source fallback
   (`frozenAliasSplit`, `oracle/plugin/go/types/frozen.go`). Snapshot bytes are frozen;
   the grammar is not; the two must diverge.
2. **The freeze event is a release ritual.** Whole-tree snapshotting works only if
   someone runs `oracle snapshot` at the right moment on the right layout. The RFC 0033
   §4.5.1 CI check that would have enforced it was never built; nothing in `scripts/` or
   `.github/` references `schemas/snapshots`.
3. **Two baseline mechanisms.** Snapshot-declared baselines plus the frozen-source
   fallback (`frozenAliasSplit`, `chainResolver` walking every `go.mod` in the repo,
   qualifier normalization) exist side by side, with degradation paths
   (`snapshot.ErrAnalysis`, `SnapshotPreVersioning`) whose only job is coping with
   unusable snapshots.
4. **Bump-time-only artifacts escape checking.** `migrate_auto.gen.go` files are emitted
   only at bump time — the old table exists only in that transient moment — so they are
   reconciled by hand and never verified by `oracle check`.
5. **Whole-tree granularity answers no question anyone asks.** Versions are per-resource
   (RFC 0042); a per-release copy of 35 files to record a one-struct change is the wrong
   shape for the data.

Authority direction is the second motivation. A schema fact should have exactly one
hand-authored home. Persisted shapes version, so their home is the version file. Output
paths, language bindings, and wire-only request types do not version, so their home is
the live file. A design that duplicates a fact across both homes must spend machinery
keeping the copies honest, and every fact that lands in the wrong home either pollutes
version history or forces edits to historical files.

---

## 2 Vocabulary

- **Version file**: `schemas/<domain>/versions/<resource>/vN.oracle` — the hand-owned
  record of a resource's complete persisted namespace at version N.
- **Chain**: A resource's ordered version files, v0 upward. Alias lines and doc
  inheritance resolve through it.
- **Definer**: As in RFC 0048 — the version at which a type's shape was last declared.
  Version-file aliases point at the definer directly.
- **Live file**: `schemas/<domain>/<resource>.oracle` — the current surface: versioned
  content projected from the chain, plus hand-authored current-only annotations.
- **Version-owned**: The domains a version file records. The live file must match chain
  resolution on them exactly.
- **Live-owned**: The domains authored directly in the live file.
- **Stored reference**: A cross-resource reference that is part of a record's persisted
  bytes, e.g. the color on a range.
- **Resolved reference**: A reference materialized at read time and never persisted with
  the record — the `@go marshal omit` fields, e.g. the status of a task or the labels on
  a range.
- **Mint**: Creating `v(N+1).oracle` — the moment version N freezes.
- **Amend**: Editing the current version file in place, for versions that have not
  shipped in a release.
- **Persisted shape**: A type's fields excluding omit and other non-persisted surface,
  per the persisted-closure rules already used by `schemadiff`.

---

## 3 Principles

1. **History is living code.** Version files are first-class `.oracle` files: parsed,
   formatted, LSP-served, and migrated with every grammar change. A shape record that
   cannot rot replaces a byte archive that already did.
2. **Version files hold the version story, and only the version story.** Everything that
   determines persisted bytes — membership, fields, optionality, marshal tags, docs — is
   authored there. Nothing else is.
3. **Authority follows persistence.** A stored reference pins the dep's version file:
   old bytes must be interpreted against the dep's shape at that version. A resolved
   reference floats on the dep's live surface: a read-time join always materializes the
   current shape.
4. **The live file is a projection, then an annotation surface.** Version-owned content
   in it is derived and gated; live-owned content is authored in place. The generation
   pipeline keeps reading live files; the gate, not the pipeline, carries the authority.
5. **Explicit over inferred.** The full namespace is enumerated — definitions, aliases,
   and (by absence) removals. Whether a version has shipped is the developer's
   knowledge, not Oracle's; the choice is expressed by which file the developer edits.
6. **Generated output is a pure function of the schema tree.** Frozen packages,
   migration auto-copies, and the live file's versioned content are deterministic,
   checked artifacts of the chain.

---

## 4 Design

### 4.0 Version files

Each versioned resource owns `schemas/<domain>/versions/<resource>/`, one hand-owned
`vN.oracle` per version. A version file enumerates the complete persisted namespace at
N:

- **Full declarations** for types whose shape changed at N, in ordinary schema syntax,
  carrying the version-owned tags: `@key`, `@doc`, and the `@go` persistence set —
  `marshal` (type- and field-level, including `omit`, `json_only`, and `flex`), `hand`,
  `migrate`, `pinned`, and `imex`. Omit transitions are part of the version story: a
  version that stops persisting a field records the `omit`.
- **Alias lines** for unchanged types: `Key = v0.Key`, pointing at the definer.
  Same-resource chain references (`v0.`) resolve implicitly — no import needed.
- **Absence means removed at N.** Deletion is explicit, reviewable history.
- **`@go pinned`** on a declaration marks a type versioned despite being unpersisted
  (hand-written-method entanglement, e.g. telem's `Size`/`Rate`), with the same gate
  semantics the old form carried: skip the persistence gate, warn if the type is
  actually persisted.
- **Docs** live on the definer's declaration; alias lines never carry docs, and the
  generated Go aliases transpose the definer's resolved doc. A doc is not shape: editing
  the definer's doc in an old file is legitimate and is how an unchanged type's
  documentation improves after the fact.

### 4.1 The live file

The live file is generated from the resolved chain, then hand-annotated. `oracle sync`
re-emits its version-owned content whenever the chain changes, preserving live-owned
content through a declaration-level merge — a hand validation tag and a version-owned
doc share one field block, so the merge operates per field, per domain.

The ownership split:

- **Version-owned** (must match chain resolution exactly; mismatch is a hard error
  naming the version file as authority): type membership, fields and their types,
  optionality, docs, `@key`, and the `@go` persistence set from §4.0.
- **Live-owned** (authored in the live file, free to add and edit): output paths, `@pb`,
  `@ts`/`@py`/`@cpp` bindings and `hand` markers, validation and index tags, and
  wire-only types — request and command shapes (`APIChannel`, task `Command`, ranger
  `New`) that are never persisted and have no version story.

Membership doubles as the persistence marker: a live type is persisted iff the current
version file declares or aliases it. Live declarations absent from the current version
file are transient surface and generate at the package root, exactly as unversioned
siblings do today.

### 4.2 Imports

Import placement follows Principle 3:

- **Stored references pin.** A version-file declaration whose persisted bytes embed a
  dep imports the dep's version file (`import "schemas/x/versions/color/v0"`). Pins are
  visible in the file and enforced by the analyzer.
- **Resolved references float.** A type referenced only by omit fields imports the dep's
  live schema (`import "schemas/synnax/status"`) and always resolves to the dep's
  current surface. This generalizes the existing rule for unversioned resources
  (`arc/program`); frozen packages already reference live shapes for resolved fields.
- **Live files** import live paths, as today.

The current version file's stored pins must equal each dep's current version — the
generated current package embeds the dep's current type, so a stale pin is a
contradiction. When a dep mints, every resource storing it fails the pin-currency gate;
the fix is the same developer-knowledge split as every shape change: bump the pin in
place if the dependent's current version has not shipped, mint the dependent if it has.
Frozen files keep their historical pins untouched.

### 4.3 Workflow

- **Birth**: hand-write `v0.oracle`; `oracle sync` emits the live file's versioned
  skeleton; annotate it with outputs and bindings. The persistence gate's blocking error
  — a persisted type in a resource with no versions directory — is the prompt. No
  further birth tooling ships with this RFC.
- **Amend**: edit the current version file in place. For versions that have never
  shipped in a release.
- **Mint**: `oracle migrate <resource>` scaffolds `v(N+1).oracle` — every type as an
  alias line to its definer, pinned declarations redeclared, imports carried forward —
  and the developer converts the changed types to full declarations. Sync then
  regenerates `migrate.gen.go`, the frozen packages, and the live file's versioned
  content. The hand-written `migrate.go` entry points of RFC 0048 §4.3 stay
  hand-authored; compile errors against the regenerated auto-copies guide them.

There is no drift detection and no bare `oracle migrate`: shape edits happen in the
record itself, so there is nothing to drift. The old model's tripwire — "you changed a
persisted shape without bumping" — is replaced by the file choice: an in-place edit is
an amendment, a new file is a bump.

### 4.4 Check gates

`oracle check` verifies, all blocking:

1. **Live-file consistency**: every version-owned domain of every versioned declaration
   in the live file structurally equals chain resolution. The error names the version
   file as the place to fix.
2. **Minimality**: every full declaration in a version file must differ structurally
   (`schemadiff`, persisted fields) from its resolved predecessor; a redundant
   redeclaration errors with "use an alias." An alias must point at a version that
   actually declares the type, and an all-alias mint is an error.
3. **Import placement**: stored references resolve through pins, resolved references
   through live imports, live files never import `versions/` directories of other
   resources.
4. **Pin currency**: the current version file's stored pins equal each dep's current
   version (§4.2).
5. **Persistence**: a persisted type must belong to a versioned resource, and a frozen
   declaration's stored closure may reference only versioned types.
6. **Checked outputs**: frozen `versions/vN` `.gen.go` files, `migrate.gen.go` files,
   and the live file's version-owned content are all regenerate-and-compare artifacts.

### 4.5 Generation

Current packages in every language — Go, TypeScript, Python, C++, protobuf — generate
from the live file exactly as they do today. This is the hybrid's central property: the
live file contains the complete current surface (versioned content emitted from the
chain, annotations authored in place), so the generation pipeline's dataflow is
untouched and the gate carries the authority.

The chain feeds the Go history machinery:

- **Frozen `versions/vN` packages** are regenerated outputs of the version files. Sync
  re-emits their `.gen.go` files deterministically; hand-written files (`migrate.go`,
  method files) coexist untouched. Define-vs-alias is read from the file, not computed:
  `versioning.AliasSplit`'s snapshot walk disappears, and Go emission keeps the RFC 0048
  §6.1 predecessor-chain form (`v2.T = v1.T`) regardless of the file's definer-targeted
  notation — the two denote the same type.
- **Resolved fields in frozen packages** reference the dep's live package (§4.2), so a
  frozen shape that kept an omit field compiles against the current resolved type.
- **`migrate.gen.go`** is a pure function of the (`v(N-1)`, `vN`) version files,
  regenerated by sync and verified by check. Hand reconciliation inside auto-copy files
  ends; hand-written transforms stay in `migrate.go` per RFC 0048 §4.3.
- **Codecs** for the current package derive from the live file's emitted marshal tags; a
  frozen package's codecs regenerate from its version file.

### 4.6 Backfill

All existing frozen versions are transcribed into version files: current versions from
the live schemas, predecessors from the frozen `types.gen.go` on disk. The legacy
payload-version directories (lineplot/log/schematic/table `legacy/`) stay out of scope.
After backfill the frozen-source fallback has no remaining caller and the entire
apparatus is deleted (§4.7).

Acceptance for the cutover: **current generated output is byte-identical** before and
after — any diff is a defect in the backfill or the split. **Frozen directories churn
freely**: regeneration from version files through today's templates supersedes
historical bytes, and the gate is that everything compiles, all suites stay green, and
no persisted shape changes.

### 4.7 Kill list

- `schemas/snapshots/` (all 141 files).
- `oracle/snapshot/` (`Create`, `LatestVersion`, `FileLoader`, `Files`, `TableLoader`,
  `ErrAnalysis`) and its tests.
- `oracle/cmd/snapshot.go`; the snapshot step of `oracle migrate`.
- `plugin.Request.SnapshotVersion` / `LoadSnapshot` / `OldResolutions` as snapshot-fed
  inputs — replaced by chain access derived from version files.
- `versioning.AliasSplit`'s snapshot tier; `SnapshotPreVersioning` and the
  pre-versioning degradation branches in `oracle/cmd/migrate.go`.
- `frozenAliasSplit`, `chainResolver`, `normalizeQualifiers`, and the rest of the
  frozen-source baseline in `oracle/plugin/go/types/frozen.go`.
- `readCoreVersion`'s role in versioning — core-release numbering no longer keys
  anything in Oracle.
- The `@go version` tag, its analyzer handling, and its documentation.
- The byte-identical drift gate, canonical version-file emission as its truth, and
  `oracle migrate --amend` — amendment is editing the file. The version-file renderer
  survives as the mint scaffolder and live-file emitter.
- The live-projection apparatus of the interim model: `versions.Annotate`, the
  field-marshal injection onto live tables, and the analyzer error banning marshal tags
  in live schemas — marshal tags are again present in live files, as emitted
  version-owned content.

### 4.8 Amendments to prior RFCs

- **RFC 0033** §3.6, §4.5.1, and §4.5.2's snapshot behavior are superseded: the
  authoritative record moves from snapshot copies to version files, and the never-built
  CI snapshot diff is replaced by the §4.4 gates.
- **RFC 0042** §5.6 is superseded. The snapshot folder as compatibility group dies with
  release coupling; cross-resource compatibility is written into each version file as
  explicit dep pins, with the pin-currency gate (§4.2) as the propagation mechanism §5.6
  feared — scoped to stored references only.
- **RFC 0048** §4.0's two-tier baseline collapses to one: the version-file chain.
  §6.2a's rejection of transcribed history is superseded by §4.6's backfill — the target
  then was dead bytes in a rotting grammar; the target now is living files. Everything
  else in RFC 0048 (backward aliasing, predecessor chain, migration placement,
  method-file placement) stands.

---

## 5 Implementation phases

**Phase 1 — chain machinery, additive.** Analyzer support for version files (chain-aware
resolution, alias lines, `@go pinned`, import rules); the versions resolver and
surfaces; frozen-package and `migrate.gen.go` regeneration from the chain; the
minimality gate. All dormant where no `versions/` directory exists.

**Phase 2 — backfill.** Transcribe every frozen version into version files (§4.6);
delete `@go version` tags, `schemas/snapshots/`, and the snapshot machinery.

**Phase 3 — the hybrid cutover, atomic.** Invert authority: live-file emission and the
declaration-level merge; the live-file consistency, import-placement, and pin-currency
gates; the stored/resolved import split; `oracle migrate` becomes the scaffolder; delete
the drift gate and the live-projection apparatus (§4.7). Verified against the §4.6
acceptance bar.

Phases 1 and 2 buy a green intermediate state and split mechanical generator work from
the repo-wide backfill. Phase 3 is a single cutover: coexistence between the drift model
and the hybrid model would mean two authorities, which is the disease this RFC treats.

---

## 6 Resolved decisions

**6.0 - Version files over pure deletion.** The frozen generated Go already on disk
could have been the only baseline — delete snapshots and keep `frozenAliasSplit`
forever. Rejected: the baseline lives in generated output rather than schema language,
and history stops being reviewable as schemas. The trade is real: version files are a
second representation of current shapes, which is exactly what the §4.4 gates hold
honest.

**6.1 - Hand-owned version files over machine-frozen emission.** The first implemented
design kept the live schema authoritative: version files were canonical emissions of the
live persisted closure, held byte-identical by a drift gate, with
`oracle migrate --amend` as the regeneration lever. Rejected after living with it: the
shape existed twice with the hand-editable copy ambiguous; type docs silently diverged
(the file's doc was an unconditional override, so a live doc edit neither drifted nor
propagated); and version files steadily absorbed non-version concerns (marshal tags
moved in; hand markers were next). The database-migration ecosystem (Rails, Alembic,
Flyway) settled the same question the same way: hand-written migrations are the
authority, the resolved schema is derived. The trade: the drift gate's tripwire on live
edits dies, replaced by the file-choice convention of §4.3.

**6.2 - Hybrid live file over a fully generated view.** With version files
authoritative, the live file could be a read-only resolved projection, with every
current-only tag moving into version files. Rejected: version files stop being pure
version story, wire-only types acquire fake version history, and every output-path or
binding edit touches a history file. The hybrid keeps both files single-purpose and
costs the declaration-level merge and gate — the most intricate machinery this RFC adds,
but bounded: the parser, renderer, and formatter exist, and the version-owned domain set
is already codified.

**6.3 - Stored pins, resolved floats.** Uniform pinning was rejected: a dep's mint would
cascade pin churn into resources that only resolve it, and a resolved field pinned to
history is wrong — a task retrieved today materializes today's status shape. Uniform
floating was rejected: a shipped version's stored bytes would silently re-interpret when
the dep changed. The persistence boundary is the pin boundary.

**6.4 - Mint versus amend stays developer knowledge.** A per-release stamp command was
rejected: it recreates the ritual failure mode that produced four dead snapshots. Oracle
never learns what shipped; the developer expresses it by editing in place or minting.

**6.5 - Explicit full-namespace enumeration over pure delta.** Pure absence-means-alias
delta files were rejected: absence would be overloaded between "unchanged" and
"removed," and files would be unreadable in isolation. Alias lines cost one line per
unchanged type and buy an unambiguous removal representation and a readable namespace at
every version. Full-copy version files were rejected outright as the snapshot disease at
smaller scale.

**6.6 - The versions directory as sole version authority.** Keeping `@go version` means
two encodings of the same fact and a gate between them. Deleting it costs a
live-analysis dependency on the versions directory — accepted; membership checks carry
the persistence semantics.

**6.7 - Frozen Go regenerated from version files over immutable-on-disk.** RFC 0048's
"history is immutable" principle is narrowed to meaning: frozen `.gen.go` bytes may be
re-emitted by newer generators, but only from a shape record that cannot change
persisted semantics. This buys checked frozen outputs and checkable auto-copies; it
costs cutover churn in frozen directories (accepted, §4.6) and forbids hand edits to
frozen generated files.

**6.8 - Declaration-level consistency gate over byte identity.** Byte identity was the
right trust model when the version file was a pure emission. The live file interleaves
hand annotations with derived content, so the gate compares per declaration, per
version-owned domain; the formatter still canonicalizes bytes separately. Structural
equality also remains the point of the minimality gate.

**6.9 - No field-level `extends`.** `Channel extends v2.Channel { +created_at }` was
rejected: the shape at N stops being readable in one place, and removal syntax and
ordering rules complicate the analyzer for modest brevity. Full redeclaration on change;
the door stays open.

**6.10 - Backfill over grandfathering.** Grandfathering keeps two baseline mechanisms
alive indefinitely, with the gnarlier one immortal. Backfill is semi-mechanical (the
shape source is the frozen Go on disk) and makes "explicitly managed" true for the whole
history. Supersedes RFC 0048 §6.2a.

---

## 7 Open questions

1. **Persisted-reference cycles.** Mutual stored references between resources would
   produce file-level import cycles. The Go layer survived because mutual refs are
   omit-only, which float on live imports — verified during Phase 2; a real cycle is a
   schema design bug to fix, not machinery to build.
2. **A dep both stored and resolved in one file.** A version file that stores one of a
   dep's types and resolves another needs both a pinned and a live import of the same
   resource, and a disambiguation rule between them. Believed rare to nonexistent today;
   decide when the analyzer meets one.
3. **Removal flow in the live file.** When a mint removes a type, the live file's stale
   declaration reads as a new wire-only type to the merge. The likely answer is a
   scaffold hint or a warning when a former member survives as an annotation; the manual
   delete is acceptable meanwhile.
4. **Chain compaction** (RFC 0048 §7.1) — unchanged by this RFC, still deferred.

---

## 8 What this RFC does not cover

- TypeScript, Python, C++, and protobuf outputs beyond §4.5's no-change guarantee: the
  `versions/vN` layout is Go-only (RFC 0048 §8), and version files feed only the Go
  history machinery.
- Wire formats, imex envelope versions, and gorp migration keys — untouched; migration
  chains and their string keys are unchanged.
- The advisory (`continue-on-error`) status of the Oracle CI check — a separate concern
  from the gates this RFC adds.
- Runtime migration dispatch (RFC 0042's deferred `decode.go`).
