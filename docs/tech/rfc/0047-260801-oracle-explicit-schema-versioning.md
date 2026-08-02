# 0047 - Oracle Explicit Schema Versioning

**Feature Name:** Explicit Versioned Schema Files Replacing Whole-Tree Snapshots

**Status:** Draft

**Related:** [RFC 0033](./0033-260320-oracle-migrations.md),
[RFC 0041](./0041-260527-core-structure-refactor.md),
[RFC 0046](./0046-260720-oracle-predecessor-chain-versioning.md)

---

# 0 - Summary

Oracle's schema history is a whole-tree copy: once per release, `oracle snapshot`
duplicates every `.oracle` file into `schemas/snapshots/vN/`, and that folder serves as
the diffing baseline for migration generation and for the predecessor-chain alias split
(RFC 0046 §4.0). This RFC deletes the snapshot system and replaces it with explicit
per-resource version files:

```
schemas/synnax/versions/channel/
  v0.oracle
  v1.oracle    # full declarations for types changed at v1; aliases for the rest
```

A version file enumerates the resource's complete persisted namespace at that version —
changed types as full declarations, unchanged types as one-line aliases to their
definer (`Key = v0.Key`), absence meaning removal. The versions directory becomes the
sole version authority: the per-type `@go version` tag is deleted, the current version
is the highest `vN` file, and membership in that file is what marks a type as
persisted. Frozen `versions/vN` Go packages become regenerated, checked outputs of
these files, and the two-tier baseline of RFC 0046 §4.0 collapses to one mechanism.

---

# 1 - Motivation

The snapshot system is dead weight, and the reasons are structural, not accidental:

1. **History rots in a dialect nothing maintains.** Every checked-in snapshot
   (v53–v56) predates RFC 0043's optionality syntax and no longer parses; all predate
   `@go version` and the `schemas/{synnax,arc,x}` layout. `versioning.AliasSplit`
   short-circuits on every one of them, and `oracle migrate`'s diffing has been
   dormant since the grammar moved on. The alias split already runs entirely on the
   frozen-generated-source fallback (`frozenAliasSplit`,
   `oracle/plugin/go/types/frozen.go`). Snapshot bytes are frozen; the grammar is not;
   the two must diverge.
2. **The freeze event is a release ritual.** Whole-tree snapshotting works only if
   someone runs `oracle snapshot` at the right moment on the right layout. The RFC
   0033 §4.5.1 CI check that would have enforced it was never built; nothing in
   `scripts/` or `.github/` references `schemas/snapshots`.
3. **Two baseline mechanisms.** Snapshot-declared baselines plus the frozen-source
   fallback (`frozenAliasSplit`, `chainResolver` walking every `go.mod` in the repo,
   qualifier normalization) exist side by side, with degradation paths
   (`snapshot.ErrAnalysis`, `SnapshotPreVersioning`) whose only job is coping with
   unusable snapshots.
4. **Bump-time-only artifacts escape checking.** `migrate_auto.gen.go` files are
   emitted only at bump time — the old table exists only in that transient moment — so
   they are reconciled by hand and never verified by `oracle check`.
5. **Whole-tree granularity answers no question anyone asks.** Versions are
   per-resource and dense (RFC 0041); a per-release copy of 35 files to record a
   one-struct change is the wrong shape for the data.

---

# 2 - Vocabulary

- **Version file** — `schemas/<domain>/versions/<resource>/vN.oracle`: the complete
  persisted namespace of a resource at version N.
- **Chain** — a resource's ordered version files, v0 upward. Alias lines and doc
  inheritance resolve through it.
- **Definer** — as in RFC 0046: the version at which a type's shape was last declared.
  Version-file aliases point at the definer directly.
- **Mint** — creating `v(N+1).oracle` from the live schema's persisted closure: the
  moment version N freezes.
- **Amend** — rewriting the current version file in place, for versions that have not
  shipped in a release.
- **Drift** — any byte difference between the current version file and the canonical
  emission Oracle would produce from the live schema.
- **Persisted shape** — a type's fields excluding `@go marshal omit` and other
  non-persisted surface, per the persisted-closure rules already used by `schemadiff`.

---

# 3 - Principles

1. **History is living code.** Version files are first-class `.oracle` files: parsed,
   formatted, LSP-served, and migrated with every grammar change. A shape record that
   cannot rot replaces a byte archive that already did.
2. **The versions directory is the version declaration.** One authority: the highest
   `vN` file is the current version, and membership in it marks a type persisted.
   There is no tag to drift against the directory.
3. **Frozen never references live — as an import rule, not a convention.** Inside a
   `versions/` directory, imports may target only other `versions/` directories.
   Outside, a live schema may import only its own resource's `versions/` directory.
4. **Below current, persisted shape only.** Version files and frozen Go packages
   record exactly what storage can contain. Transient surface (omit fields) exists
   only at current.
5. **Explicit over inferred.** The full namespace is enumerated — definitions,
   aliases, and (by absence) removals. Freezing is a deliberate developer action, not
   a release byproduct; whether a version has shipped is the developer's knowledge,
   not Oracle's.
6. **Generated output is a pure function of the schema tree.** With the chain
   permanently available in schema form, frozen packages and migration auto-copies
   become deterministic, checked artifacts.

---

# 4 - Design

## 4.0 - Version Files

Each versioned resource owns `schemas/<domain>/versions/<resource>/`, one `vN.oracle`
per version. The live tree is untouched; `pipeline.DiscoverSchemas` excludes
`versions/` subtrees from live analysis the way it excludes `snapshots/` today.

A version file enumerates the complete persisted namespace at N:

- **Full declarations** for types whose persisted shape changed at N, in ordinary
  schema syntax, carrying persistence-relevant tags (`@key`, `@go marshal`, codec
  tags). Any omit or otherwise non-persisted field is an analyzer error.
- **Alias lines** for unchanged types: `Key = v0.Key`, pointing at the definer.
  Same-resource chain references (`v0.`) resolve implicitly — no import needed;
  imports appear only for cross-resource pins.
- **Absence means removed at N.** Deletion is explicit, reviewable history.
- **`@go pinned`** on a declaration marks a type versioned despite being unpersisted
  (hand-written-method entanglement, e.g. telem's `Size`/`Rate`), with the same gate
  semantics the old `@go version N pinned` carried: skip the persistence gate, warn if
  the type is actually persisted.
- **Docs** on full declarations inherit from the predecessor version's doc for that
  type; a `@doc` tag on the declaration overrides when the meaning changed at N. A
  brand-new type seeds its doc from the live schema at mint; version-file docs are
  history-owned thereafter. Alias lines never carry docs — generated aliases transpose
  the definer's resolved doc.

Cross-resource references in full declarations import the dep's version file
(`import "schemas/x/versions/telem/v0"`) — pins computed at mint time from the dep's
chain position, visible in the file, enforced by the analyzer.

## 4.1 - Import Rules and Their Consequences

The two rules in Principle 3 have teeth:

- **The persistence gate becomes blocking.** A frozen declaration cannot reference an
  unversioned type — there is no `versions/` directory to import. Freezing a shape
  whose persisted closure includes an unversioned type is an error, as is (at check
  time) a persisted type in a resource with no versions directory. The old
  non-blocking warning becomes the error path.
- **Chain-aware resolution.** A file's namespace at N is the union along its chain:
  full declarations at N, alias targets resolved at their definers. The analyzer
  presents this the way the generated Go alias chain does.

## 4.2 - Version Authority

The per-type `@go version` tag is deleted from live schemas. A resource is versioned
iff its `versions/` directory exists; the current version is the highest `vN` file;
a live type is persisted iff the current version file declares or aliases it. Live
siblings absent from the file are transient and generate at the package root, exactly
as unversioned siblings do today.

The type-level `@go marshal` tag moves with it: it declares a codec root — a
persistence fact — and already requires versioning
(`oracle/plugin/go/marshal/marshal.go`). It lives on version-file declarations;
type-level `@go marshal` in a live schema becomes an analyzer error at cutover, and
current-package codec generation consults the current version file. The field-level
`@go marshal omit` form stays in live schemas: it marks live-only surface, which
version files reject by construction.

## 4.3 - The Freeze Model

Freezing is mint-time, with no release-time step:

- **Mint** (`oracle migrate`): writes `v(N+1).oracle` from the live persisted closure
  — full declarations for changed shapes, aliases for the rest, dep pins at current
  chain positions — then scaffolds the Go migration and syncs.
- **Amend** (`oracle migrate --amend <resource>`): rewrites the current version file
  in place and re-syncs, with no migration scaffold. For versions that have never
  shipped in a release; Oracle does not track shipped-ness, the developer does.
- **Drift** is detected by `oracle check` as a blocking gate: the current version file
  must be byte-identical to the canonical emission derived from the live schema —
  formatting, ordering, docs, everything. The error offers the two exits: bump
  (`oracle migrate <resource>`) or amend.

`oracle migrate [resource...]` bumps the named resources; bare `oracle migrate` bumps
every drifted resource. `oracle snapshot` is deleted with no replacement. A new
resource enters versioning by hand-writing its `v0.oracle`; the persistence gate's
error is the prompt. No birth tooling ships with this RFC.

## 4.4 - Check Gates

`oracle check` gains, all blocking:

1. **Drift** — current version file byte-identical to canonical emission (§4.3).
2. **Minimality** — every full declaration in a version file must differ structurally
   (`schemadiff`, persisted fields) from its resolved predecessor; a redundant
   redeclaration errors with "use an alias." An alias must point at a version that
   actually declares the type. This inherits `detectBumps`' "bump with no change"
   validation: an all-alias mint is an error.
3. **Import rules** — §4.1's two rules, plus the no-non-persisted-fields error.
4. **Persistence gate** — blocking, per §4.1.
5. **Frozen outputs** — frozen `versions/vN` `.gen.go` files and `migrate.gen.go`
   files join the checked surface (§4.5).

## 4.5 - Go Generation

Frozen `versions/vN` Go packages become regenerated outputs of the version files. Sync
re-emits their `.gen.go` files deterministically; hand-written files (`migrate.go`,
method files) coexist untouched, since sync never writes non-generated files.

- **Persisted-only frozen packages.** Regenerated frozen types carry persisted shape
  only. Omit fields exist solely in the current package, generated from the live
  schema as today. The documented "frozen dirs gain a compile-time dependency on
  latest names via omit fields" caveat dies: frozen packages become hermetic in
  meaning as well as bytes.
- **Define-vs-alias is read, not computed.** The version file enumerates the split;
  `versioning.AliasSplit`'s snapshot walk and `schemadiff` comparisons for aliasing
  disappear. Go emission keeps the RFC 0046 §6.1 predecessor-chain form
  (`v2.T = v1.T`) regardless of the file's definer-targeted alias notation — the two
  denote the same type.
- **Omit-carrying types always define in current.** A current type with omit fields
  cannot equal its omit-free frozen counterpart, so the generator materializes a full
  definition even when the version file records a persisted-identity alias. The file
  remains the truth for history and migrations; the Go materialization is an emission
  detail. Alias-based Go type identity therefore narrows to omit-free types, and
  auto-copy emits field-wise conversions where identity is lost.
- **`migrate_auto.gen.go` → `migrate.gen.go`**, now a pure function of the
  (`v(N-1)`, `vN`) version files, regenerated by sync and verified by check. Hand
  reconciliation inside auto-copy files ends; the generator emits only reachable
  helpers. Hand-written transforms stay in `migrate.go`, and the RFC 0046 §4.3
  conventions (incoming-package placement, one exported entry point, `MigrateX`
  wrappers) are unchanged.
- Codecs and stringers are emitted only for defined types, as today; a frozen
  package's codecs regenerate from its version file.

## 4.6 - Backfill

All existing frozen versions are transcribed into version files: current versions
minted from the live schemas, predecessors derived from the frozen `types.gen.go` on
disk. The legacy payload-version directories (lineplot/log/schematic/table `legacy/`)
stay out of scope, as always. After backfill the frozen-source fallback has no
remaining caller and the entire apparatus is deleted (§4.7).

Acceptance for the cutover: **current generated output is byte-identical** before and
after — any diff is a defect in the backfill or the split. **Frozen directories churn
freely**: regeneration from version files through today's templates supersedes
historical bytes (doc-template vintage, hand-reconciled files), and the gate is that
everything compiles, all suites stay green, and no persisted shape changes.

## 4.7 - Kill List

- `schemas/snapshots/` (all 141 files).
- `oracle/snapshot/` (`Create`, `LatestVersion`, `FileLoader`, `Files`,
  `TableLoader`, `ErrAnalysis`) and its tests.
- `oracle/cmd/snapshot.go`; the snapshot step of `oracle migrate`.
- `plugin.Request.SnapshotVersion` / `LoadSnapshot` / `OldResolutions` as
  snapshot-fed inputs — replaced by chain access derived from version files.
- `versioning.AliasSplit`'s snapshot tier; `SnapshotPreVersioning` and the
  pre-versioning/unparseable degradation branches in `oracle/cmd/migrate.go`.
- `frozenAliasSplit`, `chainResolver`, `normalizeQualifiers`, and the rest of the
  frozen-source baseline in `oracle/plugin/go/types/frozen.go`.
- `readCoreVersion`'s role in versioning — core-release numbering no longer keys
  anything in Oracle.
- The `@go version` tag, its analyzer handling, and its documentation.
- Type-level `@go marshal` in live schemas (§4.2) — the field-level `omit` form
  stays.

## 4.8 - Amendments to Prior RFCs

- **RFC 0033** §3.6, §4.5.1, and §4.5.2's snapshot behavior are superseded: the
  authoritative record moves from snapshot copies to version files, and the
  never-built CI snapshot diff is replaced by the §4.4 gates.
- **RFC 0041** §5.6 is superseded. The snapshot folder as compatibility group dies
  with release coupling; cross-resource compatibility is now written into each
  version file as explicit dep pins — ironically the shape §5.6 rejected as
  speculative, now carrying none of the rejected cascade machinery because pins are
  recorded at mint time rather than propagated by scaffolding.
- **RFC 0046** §4.0's two-tier baseline collapses to one: the version-file chain.
  §6.2a's rejection of transcribed history is superseded by §4.6's backfill — the
  target then was dead bytes in a rotting grammar; the target now is living files.
  Everything else in RFC 0046 (backward aliasing, predecessor chain, migration
  placement, method-file placement) stands.

---

# 5 - Implementation Phases

**Phase 1 — machinery, additive.** Analyzer support for version files (chain-aware
resolution, alias lines, `@doc`, `@go pinned`, import rules, non-persisted-field
errors); the §4.4 gates; generator reads the chain for the alias split, regenerates
frozen packages persisted-only, and emits `migrate.gen.go` as a checked artifact;
`oracle migrate` gains resource arguments and `--amend`. All dormant where no
`versions/` directory exists, so the tree stays green with zero version files.

**Phase 2 — backfill and cutover, atomic.** Write every version file (§4.6), delete
`@go version` tags, delete `schemas/snapshots/` and the §4.7 machinery, rename the
auto-copy files, and run one sync. Verified against the §4.6 acceptance bar.

A boundary between the phases buys a green intermediate state and splits mechanical
generator work from the repo-wide cutover for bisection; no further splitting earns
its keep.

---

# 6 - Resolved Decisions

**6.0 - Version files over pure deletion.** The frozen generated Go already on disk
could have been the only baseline — delete snapshots and keep `frozenAliasSplit`
forever. Rejected: schema-level bump detection ("you changed a persisted shape
without bumping") dies permanently, and the baseline lives in generated output rather
than schema language. The trade is real: version files are a second representation
that must be kept consistent, which is exactly what the byte-identity drift gate and
checked frozen outputs are for.

**6.1 - Mint-time freeze over a release-time step.** A per-release stamp command was
rejected: it recreates the ritual failure mode that produced four dead snapshots.
The trade: iterating on an unreleased version's shape requires an explicit `--amend`
instead of being silently absorbed — accepted as the honest workflow.

**6.2 - Explicit full-namespace enumeration over pure delta.** Pure
absence-means-alias delta files were the first lock, then rejected on inspection:
absence would be overloaded between "unchanged" and "removed," and files would be
unreadable in isolation. Alias lines cost one line per unchanged type and buy an
unambiguous removal representation and a readable namespace at every version.
Full-copy version files were rejected outright as the snapshot disease at smaller
scale.

**6.3 - The versions directory as sole authority over keeping `@go version`.**
Keeping the tag means two encodings of the same fact and a drift gate between them.
Deleting it costs a live-analysis dependency on the versions directory — accepted;
the persistence gate's semantics carry over as membership checks.

**6.4 - Frozen Go regenerated from version files over immutable-on-disk.** RFC 0046's
"history is immutable" principle is narrowed to *meaning*: frozen `.gen.go` bytes may
be re-emitted by newer generators, but only from a shape record that cannot change
persisted semantics. This buys checked frozen outputs and checkable auto-copies; it
costs cutover churn in frozen directories (accepted, §4.6) and forbids hand edits to
frozen generated files (previously tolerated silently, now caught).

**6.5 - Byte-identity drift gate over structural comparison.** Structural equality at
the gate would tolerate formatting, ordering, and doc divergence between the file and
its canonical emission, inviting slow rot. Byte identity makes the gate a
regenerate-and-compare, the same trust model as the rest of `oracle check`.
Structural equality survives only where it is the point: the minimality gate.

**6.6 - No field-level `extends`.** `Channel extends v2.Channel { +created_at }` was
rejected: the shape at N stops being readable in one place, and removal syntax and
ordering rules complicate the analyzer for modest brevity. Full redeclaration on
change; the door stays open.

**6.7 - Backfill over grandfathering.** Grandfathering keeps two baseline mechanisms
alive indefinitely, with the gnarlier one immortal. Backfill is semi-mechanical (the
shape source is the frozen Go on disk) and makes "explicitly managed" true for the
whole history, not just future versions. Supersedes RFC 0046 §6.2a.

**6.8 - No birth tooling.** Minting a new resource's `v0.oracle` is hand-authoring;
check's blocking gate is the prompt. Tooling can come later if the friction is real.

**6.9 - Accepted limitation: no retro-documentation.** Doc overrides ride only on
full declarations, and the minimality gate requires structural change, so an
unchanged type's frozen doc cannot be updated at a new version. Relaxable later by
allowing `@doc` on an alias line.

---

# 7 - Open Questions

1. **Persisted-reference cycles.** Mutual persisted references between resources
   would produce file-level import cycles. The Go layer survived because mutual refs
   are omit-only, which persisted-only files strip — verified during Phase 2; a real
   cycle is a schema design bug to fix, not machinery to build.
2. **`@doc` on alias lines** (§6.9), if retro-documentation ever hurts.
3. **Chain compaction** (RFC 0046 §7.1) — unchanged by this RFC, still deferred.

---

# 8 - What This RFC Does Not Cover

- TypeScript, Python, C++, and protobuf outputs: the `versions/vN` layout is Go-only
  (RFC 0046 §8), and version files feed only the Go generator.
- Wire formats, imex envelope versions, and gorp migration keys — untouched; migration
  chains and their string keys are unchanged.
- The advisory (`continue-on-error`) status of the Oracle CI check — a separate
  concern from the gates this RFC adds.
- Runtime migration dispatch (RFC 0041's deferred `decode.go`).
