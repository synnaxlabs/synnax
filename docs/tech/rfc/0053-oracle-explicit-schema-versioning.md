# 53 Oracle explicit schema versioning

- **Author**: Patrick Dotson
- **Date**: 2026-08-11
- **Related**: [RFC 0033 - Oracle migrations](0033-oracle-migrations.md),
  [RFC 0042 - Core structure refactor](0042-core-structure-refactor.md),
  [RFC 0048 - Oracle predecessor-chain versioning](0048-oracle-predecessor-chain-versioning.md)

## 0 Summary

`oracle snapshot` copies the whole schema tree into `schemas/snapshots/vN/` once per
release. That copy is the baseline for migration diffing and for the RFC 0048 §4.0 alias
split. This RFC deletes the snapshot system. Each versioned resource instead owns
hand-written version files:

```
schemas/synnax/versions/channel/
  v0.oracle
  v1.oracle    # full declarations for types changed at v1; aliases for the rest
```

A version file enumerates the resource's complete persisted namespace at N: changed
types as full declarations, unchanged types as alias lines to their definer
(`Key = v0.Key`), absence meaning removal. Version files are the authority for every
persisted shape. The live schema becomes a generated projection of the chain, which
developers annotate with current-only concerns. Frozen `versions/vN` Go packages and
`migrate.gen.go` files become regenerated, checked outputs of the chain.

## 1 Motivation

The snapshot system is dead weight:

- Every checked-in snapshot (v53-v56) predates the current grammar and layout and no
  longer parses. The alias split already runs on the frozen-generated-source fallback.
- The freeze depends on a release ritual — run `oracle snapshot` at the right moment —
  that no CI step enforces.
- Two baseline mechanisms coexist: snapshots plus the frozen-source fallback, with
  degradation paths whose only job is coping with unusable snapshots.
- `migrate_auto.gen.go` is emitted only at bump time, so `oracle check` cannot verify
  it.
- Whole-tree copies record per-resource changes at the wrong granularity.

The second motivation is authority: each schema fact must have exactly one hand-authored
home. Persisted shapes version, so they live in version files. Output paths, bindings,
and wire-only types do not version, so they live in the live file.

## 2 Design

### 2.0 Version files

Each versioned resource owns `schemas/<domain>/versions/<resource>/`, one hand-owned
`vN.oracle` per version:

- **Full declarations** for types whose shape changed at N, carrying the version-owned
  tags: `@key`, `@doc`, and the `@go` persistence set — `marshal` (type- and
  field-level, including `omit`, `json_only`, and `flex`), `hand`, `migrate`, `pinned`,
  and `imex`.
- **Alias lines** for unchanged types: `Key = v0.Key`, pointing at the definer.
  Same-resource chain references resolve implicitly.
- **Absence means removed at N.**
- **`@go pinned`** marks a type versioned despite being unpersisted (hand-method
  entanglement, e.g. telem's `Size`). Pinned declarations always declare fully and are
  exempt from the minimality gate.
- **Docs** live on the definer's declaration; alias lines never carry docs. A doc is not
  shape: editing a definer's doc in an old file is how an unchanged type's documentation
  improves.

### 2.1 The live file

`oracle sync` generates the live file's version-owned content from the resolved chain
and preserves the file's own annotations through a declaration-level merge:

- **Version-owned** (must equal chain resolution; a mismatch is a hard error naming the
  version file as authority): type membership, fields, optionality, docs, `@key`, and
  the `@go` persistence set.
- **Live-owned** (authored in the live file): output paths, `@pb`, `@ts`/`@py`/`@cpp`
  bindings and `hand` markers, validation and index tags, and wire-only types
  (`APIChannel`, task `Command`) that have no version story.

Membership doubles as the persistence marker: a live type is persisted iff the current
version file declares or aliases it. Other live declarations are transient surface and
generate at the package root.

### 2.2 Imports

Imports split on the persistence boundary:

- **Stored references pin.** A reference that is part of a record's persisted bytes (a
  range's color) imports the dependency's version file
  (`import "schemas/x/versions/color/v0"`).
- **Resolved references float.** A reference materialized at read time — the
  `@go marshal omit` fields, e.g. a task's status — imports the dependency's live schema
  and always resolves its current surface. A dependency without a chain is always
  imported live.
- **Live files** import live paths, as today.

The current version file's stored pins must equal each dependency's current version: the
generated current package embeds the dependency's current type, so a stale pin is a
contradiction. When a dependency mints, each resource storing it bumps its pin in place
(unshipped) or mints (shipped). Frozen files keep their historical pins.

### 2.3 Workflow

- **Birth**: hand-write `v0.oracle`; sync emits the live skeleton; annotate it. The
  persistence gate's error — a persisted type with no versions directory — is the
  prompt.
- **Amend**: edit the current version file in place, for versions that have not shipped
  in a release.
- **Mint**: `oracle migrate <resource>` scaffolds `v(N+1).oracle` — alias lines to
  definers, pinned declarations redeclared, imports carried — and the developer converts
  the changed types to full declarations. Sync regenerates the rest.

Whether a version shipped is developer knowledge; Oracle never tracks it. There is no
drift detection: an in-place edit is an amendment, a new file is a bump.

### 2.4 Check gates

The blocking `versions` gate verifies:

1. **Live-file consistency**: on-disk live files equal the merged projection.
2. **Minimality**: a redeclaration structurally identical to its resolved predecessor
   errors with "use an alias"; enum member sets compare exactly.
3. **Import placement**: stored references pin, resolved references float, and a live
   file never imports another resource's versions directory.
4. **Pin currency**: the current surface's stored pins equal each dependency's current
   version, transitively.
5. **Persistence**: a persisted type must belong to a versioned resource.
6. **Checked outputs**: frozen `.gen.go`, `migrate.gen.go`, and the live projection are
   regenerate-and-compare artifacts.

### 2.5 Generation

Current packages in every language generate from the live file exactly as today — the
live file carries the complete current surface, so the pipeline's dataflow is untouched.
The chain feeds the Go history machinery:

- Frozen `versions/vN` packages regenerate from version files; hand-written files
  (`migrate.go`, method files) coexist. Define-vs-alias is read from the file, and Go
  emission keeps the RFC 0048 §6.1 predecessor-chain form.
- Resolved fields in frozen packages reference the dependency's live package.
- `migrate.gen.go` is a pure function of the two adjacent version files. Hand transforms
  stay in `migrate.go` per RFC 0048 §4.3; compile errors against the regenerated
  auto-copies guide them.
- Codecs are explicit: a struct or union gets one iff its declaration carries
  `@go marshal`; references never pull a codec in. Frozen codecs regenerate from version
  files.

The `versions/vN` layout stays Go-only (RFC 0048 §8). Wire formats, imex envelope
versions, and gorp migration keys are untouched.

### 2.6 Backfill and kill list

All existing frozen versions are transcribed into version files: current versions from
the live schemas, predecessors from the frozen `types.gen.go` on disk. Acceptance:
current generated output is byte-identical before and after; frozen directories churn
freely as long as everything compiles and no persisted shape changes.

Deleted: `schemas/snapshots/`, `oracle/snapshot/`, `oracle/cmd/snapshot.go`, the
`plugin.Request` snapshot plumbing, the frozen-source baseline (`frozenAliasSplit`,
`chainResolver`), the `@go version` tag, and the interim projection model's apparatus:
the byte-identical drift gate, `oracle migrate --amend`, `versions.Annotate`'s
field-marshal injection, and the analyzer ban on marshal tags in live schemas.

### 2.7 Amendments to prior RFCs

- **RFC 0033**: §3.6, §4.5.1, and §4.5.2's snapshot behavior are superseded by version
  files and the §2.4 gates.
- **RFC 0042**: §5.6 is superseded. Cross-resource compatibility is written into each
  version file as explicit pins, propagated by the pin-currency gate.
- **RFC 0048**: §4.0's two-tier baseline collapses into the chain, and §6.2a's rejection
  of transcribed history is superseded by §2.6. Everything else stands.

## 3 Implementation phases

- **Phase 1: Chain machinery, additive.** Version-file analysis, the versions resolver,
  frozen and `migrate.gen.go` regeneration, the minimality gate.
- **Phase 2: Backfill.** Transcribe every frozen version; delete `@go version` and the
  snapshot machinery.
- **Phase 3: Hybrid cutover, atomic.** Invert authority: live-file merge and consistency
  gate, the import split and pin-currency gate, `oracle migrate` as scaffolder,
  drift-gate deletion.

## 4 Resolved decisions

1. **Version files over pure deletion**: Keeping only the frozen Go as baseline was
   rejected — history stops being reviewable as schemas.
2. **Hand-owned files over machine-frozen emission**: The first implementation froze
   canonical emissions behind a byte drift gate. Rejected after living with it: the
   hand-editable copy was ambiguous, docs silently diverged, and non-version concerns
   kept leaking in. Rails and Alembic settled the same question the same way.
3. **Hybrid live file over a fully generated view**: A read-only projection forces
   wire-only types and binding edits into history files. The hybrid costs the
   declaration-level merge — the most intricate machinery this RFC adds.
4. **Stored pins, resolved floats**: Uniform pinning cascades mint churn into resources
   that only resolve the dependency; uniform floating silently re-interprets shipped
   bytes.
5. **Mint versus amend stays developer knowledge**: A per-release stamp command
   recreates the ritual failure that produced four dead snapshots.
6. **Full-namespace enumeration over pure delta**: Absence would mean both "unchanged"
   and "removed"; alias lines keep every file readable alone.
7. **Frozen Go regenerated from version files**: RFC 0048's "history is immutable"
   narrows to shape semantics — bytes may re-emit from a record that cannot change
   persisted meaning. Hand edits to frozen generated files are forbidden.
8. **Declaration-level gate over byte identity**: The live file interleaves hand and
   derived content, so the gate compares per declaration and domain.
9. **No field-level `extends`**: The shape at N must read in one place; full
   redeclaration on change.
10. **Backfill over grandfathering**: Grandfathering keeps the gnarlier baseline
    mechanism alive forever.

## 5 Open questions

1. **Persisted-reference cycles**: Mutual stored references would cycle imports. None
   exist — mutual references are omit-only and float. A real cycle is a schema design
   bug, not machinery to build.
2. **A dependency both stored and resolved in one file**: Needs a pinned and a live
   import of the same resource. Decide when the analyzer meets one.
3. **Removal flow in the live file**: After a mint removes a type, the stale live
   declaration reads as a new wire-only type to the merge. Manual delete is acceptable
   meanwhile.
4. **Chain compaction** (RFC 0048 §7.1): still deferred.
