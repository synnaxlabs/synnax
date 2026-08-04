# 47 Oracle predecessor-chain type versioning

- **Author**: Patrick Dotson
- **Date**: 2026-07-27
- **Related**: [RFC 0033](./0033-oracle-migrations.md),
  [RFC 0042](./0042-core-structure-refactor.md),
  [RFC 0044](./0044-oracle-optionality-defaults-mutation.md)

## 0 Summary

Under the full-copy layout every `versions/vN/` package was a complete copy: when a
resource bumped its `@go version`, Oracle re-emitted every type at the path into the new
version directory, and the freeze step re-emitted every type into the outgoing one.
Between arc's `types/versions/v0` and `v1`, two types changed shape; roughly ten were
duplicated, along with their codecs and enum stringers, and the hand-written method
files were ported by hand.

This RFC replaces full copies with a predecessor chain. A version package defines only
the types whose shape changed at that version; every other type is a Go type alias to
the previous version's package:

```go
// arc/go/types/versions/v1/types.gen.go
type Param = v0.Param

type FunctionProperties struct { ... } // changed at v1: defined here
```

Because a Go alias is the same type, methods, codecs, and interface satisfactions travel
with it. Historical version directories are never regenerated — a bump adds a new
package and touches nothing behind it. Migration copies between versions of an unchanged
type collapse into direct assignment, because the two names denote one type.

---

## 1 Motivation

The full-copy layout had three concrete costs, all visible in the repo before the
cutover:

1. **Duplication scaled with package size, not change size.** A one-field change to one
   struct re-emitted the whole package twice (outgoing freeze + incoming current).
   `closureForPath` did this deliberately: "if any type from a package is needed,
   include ALL types from that package" (deleted with this RFC).
2. **Hand-written files ported by hand.** `moveHelpers` (also deleted) carried the
   version package's method file forward wholesale and deleted the outgoing copy, and
   files like arc's `type.go` and `msgpack.go` (~1,300 lines with tests) had to follow
   the current version by manual copy. Each port was an opportunity for drift.
3. **Migration auto-copies were busywork.** `v0.Param` and `v1.Param` were distinct
   structs even when byte-identical, so `migrate_auto.gen.go` generated field-by-field
   copies for types that never changed.

---

## 2 Vocabulary

- **Definer**: the version package where a type's current shape is defined as a real
  declaration. A type's definer is the version at which its shape last changed (or v0,
  where everything is defined).
- **Alias surface**: the `T = vPrev.T` declarations (plus enum/union const
  re-declarations) that complete a version package's namespace.
- **Predecessor chain**: each version package imports only `v(N-1)`. An alias may
  resolve through several hops to its definer; the Go compiler and gopls pierce the
  chain, and `grep "type T struct"` finds definers directly.
- **Changed**: structurally unequal per `schemadiff.SchemasEqual`
  (`oracle/plugin/go/internal/schemadiff`), which is transitive: a type whose own
  declaration is untouched but whose referenced types changed shape is changed.

---

## 3 Principles

1. **History is immutable.** Once a version is frozen by a bump, its directory is never
   regenerated. This extends the frozen-snapshot rule from schemas to generated code.
2. **Every version presents a complete namespace.** `vN` exposes every type at the path
   — by definition or by alias. Consumers (root re-export, migrations, frozen
   dependents) never need to know which.
3. **Transitive shape equality decides define-vs-alias.** Aliasing a type whose nested
   types changed would pin stale shapes; `schemadiff.SchemasEqual`'s recursive
   comparison is the gate, and `schemadiff.SchemaDiff`'s `TypeDescendantChanged` already
   classifies it.
4. **Code lives with the definer.** Go forbids methods on aliased non-local types, so
   hand-written method files (`<resource>.go`, arc's `type.go`) sit in the defining
   package permanently. When a type is redefined at a bump, its methods are ported to
   the new definer; omissions surface as compile errors at root call sites.

---

## 4 Design

### 4.0 The emission rule

When `go/types` generates a version-laid-out current package `vM`:

- **Baseline**: the latest snapshot in `schemas/snapshots/` whose resolution declares
  version `M-1` for the path. Bumps are validated against the latest snapshot
  (`detectBumps`, `oracle/plugin/go/migrate/migrate.go`), so once per-resource versions
  appear in snapshots this baseline always exists; snapshots are append-only and checked
  in, so the rule is stateless and deterministic across syncs.
- **Frozen-source fallback**: when no snapshot declares `M-1` (all history predates
  `@go version`, and the pre-versioning snapshots no longer parse under the current
  grammar), the frozen predecessor package itself is the baseline: the generator renders
  the would-be define-all file, parses it and the frozen `types.gen.go` with comments
  stripped, and aliases exactly the types whose declarations are identical — struct tags
  included — computed to a fixpoint so a type referencing a re-defined local type is
  re-defined too (`frozenAliasSplit`, `oracle/plugin/go/types/frozen.go`). Before
  comparing, both sides are canonicalized through alias chains — local and cross-package
  (`chainResolver`): a frozen predecessor that is itself an alias surface, or that names
  a dependency's older version directory, still compares equal to a candidate naming the
  definer, so chained history keeps aliasing instead of spuriously redefining. An alias
  is emitted only when it denotes literally the same code, which makes the retrofit
  immune to generator-vintage drift (a frozen struct missing today's `omitzero` tags
  simply stays defined).
- **Define** a type when it has no baseline counterpart (brand-new types fold into the
  current version), or when `schemadiff.SchemasEqual` against the baseline reports a
  change (own shape or transitive).
- **Alias** everything else: `type T = vPrev.T`, with enum and union discriminator
  consts re-declared beside the alias. The emission machinery lives in
  `aliasFileGenerator` (`oracle/plugin/go/types/alias.go`), including generic type
  parameters (`Status[Details any] = vPrev.Status[Details]`). The same machinery emits
  the re-export surfaces above the version directories — the `versions/types.gen.go`
  selector aliasing the latest version and the package-root `types.gen.go` aliasing the
  selector — and both include hand-written (`@go omit`) types, so the full namespace at
  a path is presented by generated aliases rather than hand-maintained re-export files.
- **v0** and paths with neither a declared baseline nor a frozen predecessor on disk
  define everything.

### 4.1 Codecs and methods

`codec.gen.go` and enum `_string.go` files are emitted only for types defined at that
version. Aliased types carry their codec methods and stringers through the alias from
their definer. The same holds for `gorp.Entry` methods: they live in the definer's
`<resource>.go` method file (the existing placement,
`core/pkg/service/status/versions/v1/status.go`), and the freeze-time
`generateGorpEntryMethods` append is deleted — a frozen package already has whatever it
needs, because it was current when its content was written.

### 4.2 The bump

`freezePath`'s re-emission disappears. The outgoing package is already in its final form
on disk: its defined types reference dependency versions that are themselves immutable,
so imports are pinned by construction. A bump now:

1. Emits the incoming `v(N+1)` package under the emission rule (mostly aliases).
2. Scaffolds migration files into the incoming package (§4.3).
3. Does **not** move hand-written method files. The developer ports methods for
   redefined types only, guided by compile errors; methods on still-aliased types stay
   put.

`detectBumps`' discipline is unchanged: shape change without a bump, skipped versions,
and decreases remain errors.

### 4.3 Migrations

Migration files for **all** paths live in the incoming version package: `vN` holds
`migrate.go` / `migrate_auto.gen.go` transforming `v(N-1)` types into `vN` types. The
generator already scaffolds both Gorp-entry and value-type paths this way
(`scaffoldIncoming`, `oracle/plugin/go/migrate/migrate.go`). Legacy reverse-direction
files (migrations living in the frozen old package, importing the new version) form an
import cycle the moment the current package aliases backward, so the retrofit moved them
into the incoming packages for the affected paths (`arc/types` v0→v1, `arc/ir` v0→v1)
and retargeted their callers; paths whose current packages emitted no aliases keep their
legacy files until they next bump.

Each version package exports a single entry point — `Migration`, or `Migrations` when
one bump carries several steps (arc v1's shape lift plus its `set_status` rename) — and
a hand-written `versions/migrations.go` concatenates them into the ordered chain the
service registers with gorp. Codec re-encodings (`gorp.CodecMigration`) are pinned in
the version package whose stored bytes they rewrite (arc v0's `msgpack_to_orc`), not in
the version that followed them.

Auto-copy generation simplifies: when a nested type is `TypeUnchanged`, old and new
names denote the same Go type, and the generated copy is direct assignment. Only
`TypeChanged` and `TypeDescendantChanged` types need per-field copies.

### 4.4 Cross-path dependencies

Current packages keep referencing their dependencies' current versions via
`versioning.RewriteCurrent`. Two cases:

- **Dep change reaches this path's types**: transitivity marks them changed, this path
  must bump, and the redefined types point at the dep's new version.
- **Dep change does not reach them**: this path's regenerated current package repoints
  imports at the dep's new version directory, but every referenced dep type is unchanged
  there and therefore an alias; type identity is preserved through the chain and the
  emitted code is byte-stable.

### 4.5 Amendments to RFC 0042

RFC 0042 §4.3.1 ("Each Version Is Self-Contained") is amended: a version package
presents a complete _namespace_, not a self-contained copy. `vN` imports `v(N-1)` by
design — the RFC 0033 §3.6 rule that a version imports nothing outside itself already
bent when incoming-package migrations imported their predecessor; the chain makes that
the norm. The consequence is that historical version directories cannot be individually
deleted while newer versions alias into them (§6.4).

Two mechanical pieces of RFC 0042 are superseded with the freeze-flow deletion (§4.2):
§4.6.0's bump steps that re-emit the outgoing package and move the method file forward,
and §5.3's Oracle-automated `helpers.go` move — method files stay with their definer,
and the version package's method file is named for the resource (`status.go`), not
`helpers.go`. RFC 0042's layout sections also predate the Phase 2 rename of the version
directory from `types/` to `versions/`; this RFC uses the landed naming.

---

## 5 Implementation phases

**Phase 1 — generator.** Teach `go/types` the emission rule (baseline resolution,
define-vs-alias split, alias + const emission for version packages); restrict codec and
stringer emission to defined types; delete `freezePath` re-emission, `closureForPath`'s
package expansion, `generateGorpEntryMethods`, and `moveHelpers`; move value-type
migration scaffolding to the incoming package. Oracle's own test suite covers the split
(unchanged → alias, own-change → define, descendant-change → define, new type → define
at current, pre-versioning baseline → define all).

**Phase 2 — retrofit cutover.** The pre-versioning snapshots no longer parse under the
current grammar, so the frozen-source fallback drives the retrofit: one sync regenerated
every current version package against its frozen predecessor, collapsing byte-identical
declarations into aliases. Reconciliation moved hand-written methods whose receiver
types became aliases to their definer packages (arc's `dimensions.go` and
`ChanDirection` methods, rack/task `Key` methods, arc service's `StatusDetails`
decoder), deleted the now-duplicate enum `_string.go` files from current packages, and
relocated the cycle-inducing legacy migrations (§4.3). The cutover also renamed the
version directory from `types/` to `versions/`, routed package-root re-exports through
the `versions` selector, renumbered `arc/ir` and `arc/graph` onto dense v0/v1 (removing
a spurious version link between them), and consolidated migration exports onto the
one-entry-point-per-version shape (§4.3). Frozen packages gained only files; no frozen
declaration changed. Wire formats are untouched throughout — the change is purely
source-level.

---

## 6 Resolved decisions

**6.0 - Backward aliasing (new → old), not forward materialization.** The alternative
kept definitions in the current package and materialized frozen copies into historical
packages at divergence time. Rejected: it rewrites history on every divergence,
contradicting the frozen-snapshot principle, and the current packages churn anyway. The
trade is real: under backward aliasing, a type's definition and methods stay in the
version folder where it last changed, so current packages become thin alias surfaces
over definitions scattered by last-change time.

**6.1 - Predecessor chain, not direct-to-definer.** `v2.T = v1.T` even when the definer
is v0. Both compile to the identical type; the chain needs only the predecessor baseline
(what `plugin.Request` carries), keeps each package to one version import, and makes a
bump's diff read as the delta against its predecessor. The trade: a human reading
`v4/types.gen.go` may hop files to reach a struct body — mitigated by gopls and by
grepping for the definition.

**6.2 - Existing full-copy history is grandfathered.** Frozen full-copy directories stay
as they are; an alias-form current version chains into them exactly as it would into
alias-form history. Rewriting history for uniformity was rejected as pure churn against
immutable packages.

**6.2a - Frozen-source baseline over transcribed or annotated history.** Two retrofit
alternatives were rejected: transcribing the pre-versioning snapshots into current
grammar (hand-authoring ~30 schemas of history, and the frozen snapshots are frozen),
and a per-type `@go changed` annotation (relies on humans reconstructing which types
changed, and cannot see tag-level drift the schema does not express). Comparing against
the frozen generated source is the only baseline that is both authoritative and already
machine-readable. The trade is real: the generator reads its own prior output, and a
future generator-style change makes affected types fall back to definitions rather than
aliases — safe, but churny; the snapshot-declared baseline takes over from the first
post-versioning snapshot onward.

**6.3 - Migrations live in the incoming package for all paths.** The generator already
scaffolds this way; the RFC ratifies it, and the retrofit moved the legacy
reverse-direction files that would otherwise cycle (§4.3).

**6.4 - Version directories become permanently pinned.** Full copies allowed, in
principle, deleting ancient version packages wholesale. The chain gives that up: every
version is reachable from current through aliases. Accepted — nothing prunes these
directories today, and a future compaction mechanism (§7) can restore the ability if it
is ever needed.

---

## 7 Open questions

1. **Chain compaction.** A future `@go` marker could force a bump to define every type,
   cutting the chain and making everything older prunable. Deferred until chain length
   is a demonstrated problem.
2. **Baseline resolution cost.** Resolving the latest snapshot declaring `M-1` loads
   historical snapshot resolutions per sync. If measurable, cache per-path baselines;
   not expected to matter at current snapshot counts.

---

## 8 What this RFC does not cover

- TypeScript, Python, C++, and protobuf outputs: the `versions/vN` layout is Go-only
  (RFC 0042 §4.3), and this RFC changes nothing outside the Go generator.
- Snapshot cadence, `oracle snapshot`/`oracle migrate` CLI behavior, and the bump
  discipline in `detectBumps` — all unchanged.
- The runtime migration dispatch (RFC 0042's deferred `decode.go`) and Gorp integration.
