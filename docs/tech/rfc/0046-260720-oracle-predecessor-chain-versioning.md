# 0046 - Oracle Predecessor-Chain Type Versioning

**Feature Name:** Predecessor-Chain Aliasing for Versioned Go Type Packages

**Status:** Draft

**Related:** [RFC 0033](./0033-260320-oracle-migrations.md),
[RFC 0041](./0041-260527-core-structure-refactor.md),
[RFC 0043](./0043-260615-oracle-optionality-defaults-mutation.md)

---

# 0 - Summary

Today every `types/vN/` package is a full copy: when a resource bumps its `@go
version`, Oracle re-emits every type at the path into the new version directory, and
the freeze step re-emits every type into the outgoing one. Between arc's
`types/types/v0` and `v1`, two types changed shape; roughly ten were duplicated,
along with their codecs and enum stringers, and the hand-written method files were
ported by hand.

This RFC replaces full copies with a predecessor chain. A version package defines
only the types whose shape changed at that version; every other type is a Go type
alias to the previous version's package:

```go
// arc/go/types/types/v1/types.gen.go
type Param = v0.Param

type FunctionProperties struct { ... } // changed at v1: defined here
```

Because a Go alias is the same type, methods, codecs, and interface satisfactions
travel with it. Historical version directories are never regenerated — a bump adds a
new package and touches nothing behind it. Migration copies between versions of an
unchanged type collapse into direct assignment, because the two names denote one
type.

---

# 1 - Motivation

The full-copy layout has three concrete costs, all visible in the repo today:

1. **Duplication scales with package size, not change size.** A one-field change to
   one struct re-emits the whole package twice (outgoing freeze + incoming current).
   `closureForPath` does this deliberately: "if any type from a package is needed,
   include ALL types from that package" (`oracle/plugin/go/migrate/migrate.go:544`).
2. **Hand-written files port by hand.** `moveHelpers`
   (`oracle/plugin/go/migrate/migrate.go:427`) carries `helpers.go` forward
   wholesale and deletes the outgoing copy, and files like arc's `type.go` and
   `msgpack.go` (~1,300 lines with tests) must follow the current version by manual
   copy. Each port is an opportunity for drift.
3. **Migration auto-copies are busywork.** `v0.Param` and `v1.Param` are distinct
   structs even when byte-identical, so `migrate_auto.gen.go` generates
   field-by-field copies for types that never changed.

---

# 2 - Vocabulary

- **Definer** — the version package where a type's current shape is defined as a
  real declaration. A type's definer is the version at which its shape last changed
  (or v0, where everything is defined).
- **Alias surface** — the `T = vPrev.T` declarations (plus enum/union const
  re-declarations) that complete a version package's namespace.
- **Predecessor chain** — each version package imports only `v(N-1)`. An alias may
  resolve through several hops to its definer; the Go compiler and gopls pierce the
  chain, and `grep "type T struct"` finds definers directly.
- **Changed** — structurally unequal per `schemasEqual`
  (`oracle/plugin/go/migrate/schema.go:23`), which is transitive: a type whose own
  declaration is untouched but whose referenced types changed shape is changed.

---

# 3 - Principles

1. **History is immutable.** Once a version is frozen by a bump, its directory is
   never regenerated. This extends the frozen-snapshot rule from schemas to
   generated code.
2. **Every version presents a complete namespace.** `vN` exposes every type at the
   path — by definition or by alias. Consumers (root re-export, migrations, frozen
   dependents) never need to know which.
3. **Transitive shape equality decides define-vs-alias.** Aliasing a type whose
   nested types changed would pin stale shapes; `schemasEqual`'s recursive
   comparison is the gate, and `SchemaDiff`'s `TypeDescendantChanged`
   (`oracle/plugin/go/migrate/schema.go:168`) already classifies it.
4. **Code lives with the definer.** Go forbids methods on aliased non-local types,
   so hand-written method files (`helpers.go`, arc's `type.go`) sit in the defining
   package permanently. When a type is redefined at a bump, its methods are ported
   to the new definer; omissions surface as compile errors at root call sites.

---

# 4 - Design

## 4.0 - The Emission Rule

When `go/types` generates a version-laid-out current package `vM`:

- **Baseline** — the latest snapshot in `schemas/snapshots/` whose resolution
  declares version `M-1` for the path. Bumps are validated against the latest
  snapshot (`detectBumps`, `oracle/plugin/go/migrate/migrate.go:140`), so this
  snapshot always exists; snapshots are append-only and checked in, so the rule is
  stateless and deterministic across syncs.
- **Define** a type when it has no baseline counterpart (brand-new types fold into
  the current version, `migrate.go:180`), or when `schemasEqual` against the
  baseline reports a change (own shape or transitive).
- **Alias** everything else: `type T = vPrev.T`, with enum and union discriminator
  consts re-declared beside the alias. The emission machinery exists in
  `aliasFileGenerator` (`oracle/plugin/go/types/alias.go:26`), including generic
  type parameters (`Status[Details any] = vPrev.Status[Details]`).
- **v0** and paths whose only snapshots predate per-resource versioning
  (`versioning.PreVersioning`) define everything — the current fallback.

## 4.1 - Codecs and Methods

`codec.gen.go` and enum `_string.go` files are emitted only for types defined at
that version. Aliased types carry their codec methods and stringers through the
alias from their definer. The same holds for `gorp.Entry` methods: they live in the
definer's `helpers.go` (the existing placement,
`core/pkg/service/status/types/v2/helpers.go:74`), and the freeze-time
`generateGorpEntryMethods` append (`migrate.go:646`) is deleted — a frozen package
already has whatever it needs, because it was current when its content was written.

## 4.2 - The Bump

`freezePath`'s re-emission (`migrate.go:221`) disappears. The outgoing package is
already in its final form on disk: its defined types reference dependency versions
that are themselves immutable, so imports are pinned by construction. A bump now:

1. Emits the incoming `v(N+1)` package under the emission rule (mostly aliases).
2. Scaffolds migration files into the incoming package (§4.3).
3. Does **not** move `helpers.go`. The developer ports methods for redefined types
   only, guided by compile errors; methods on still-aliased types stay put.

`detectBumps`' discipline is unchanged: shape change without a bump, skipped
versions, and decreases remain errors.

## 4.3 - Migrations

Migration files for **all** paths live in the incoming version package: `vN` holds
`migrate.go` / `migrate_auto.gen.go` transforming `v(N-1)` types into `vN` types.
The generator already scaffolds both gorp-entry and value-type paths this way
(`scaffoldIncoming`, `oracle/plugin/go/migrate/migrate.go`); the reverse-direction
files on disk (`arc/go/types/types/v0/migrate.go` imports v1) predate it. Those
legacy files are harmless under the chain: a cycle would need v1 to import v0, and
frozen full-copy packages never alias backward. They can migrate to the incoming
convention whenever their resources next bump.

Auto-copy generation simplifies: when a nested type is `TypeUnchanged`, old and new
names denote the same Go type, and the generated copy is direct assignment. Only
`TypeChanged` and `TypeDescendantChanged` types need per-field copies.

## 4.4 - Cross-Path Dependencies

Current packages keep referencing their dependencies' current versions via
`versioning.RewriteCurrent`. Two cases:

- **Dep change reaches this path's types** — transitivity marks them changed, this
  path must bump, and the redefined types point at the dep's new version.
- **Dep change does not reach them** — this path's regenerated current package
  repoints imports at the dep's new version directory, but every referenced dep
  type is unchanged there and therefore an alias; type identity is preserved
  through the chain and the emitted code is byte-stable.

## 4.5 - Amendments to RFC 0041

RFC 0041 §4.3.1 ("Each Version Is Self-Contained") is amended: a version package
presents a complete *namespace*, not a self-contained copy. `vN` imports `v(N-1)`
by design — the RFC 0033 §3.6 rule that a version imports nothing outside itself
already bent when incoming-package migrations imported their predecessor; the chain
makes that the norm. The consequence is that historical version directories cannot
be individually deleted while newer versions alias into them (§6.4).

---

# 5 - Implementation Phases

**Phase 1 — generator.** Teach `go/types` the emission rule (baseline resolution,
define-vs-alias split, alias + const emission for version packages); restrict codec
and stringer emission to defined types; delete `freezePath` re-emission,
`closureForPath`'s package expansion, `generateGorpEntryMethods`, and `moveHelpers`;
move value-type migration scaffolding to the incoming package. Oracle's own test
suite covers the split (unchanged → alias, own-change → define, descendant-change
→ define, new type → define at current, pre-versioning baseline → define all).

**Phase 2 — activation.** No cutover sync is needed. Every snapshot on disk today
(v53–v56) predates `@go version`, so no baseline resolves and the new generator is
byte-identical on the current repo — `oracle check` passes unchanged. Existing
full-copy current packages are exactly what no-baseline emission produces, so they
are grandfathered by construction. The chain begins organically: the next
`oracle snapshot` captures per-resource versions, and the first bump after it emits
the first alias-form current package. Wire formats are untouched throughout — the
change is purely source-level.

---

# 6 - Resolved Decisions

**6.0 - Backward aliasing (new → old), not forward materialization.** The
alternative kept definitions in the current package and materialized frozen copies
into historical packages at divergence time. Rejected: it rewrites history on every
divergence, contradicting the frozen-snapshot principle, and the current packages
churn anyway. The trade is real: under backward aliasing, a type's definition and
methods stay in the version folder where it last changed, so current packages
become thin alias surfaces over definitions scattered by last-change time.

**6.1 - Predecessor chain, not direct-to-definer.** `v2.T = v1.T` even when the
definer is v0. Both compile to the identical type; the chain needs only the
predecessor baseline (what `plugin.Request` carries), keeps each package to one
version import, and makes a bump's diff read as the delta against its predecessor.
The trade: a human reading `v4/types.gen.go` may hop files to reach a struct body —
mitigated by gopls and by grepping for the definition.

**6.2 - Existing full-copy history is grandfathered.** Frozen full-copy directories
stay as they are; an alias-form current version chains into them exactly as it
would into alias-form history. Rewriting history for uniformity was rejected as pure
churn against immutable packages.

**6.3 - Migrations live in the incoming package for all paths.** The generator
already scaffolds this way; the RFC ratifies it. Legacy reverse-direction files
(arc's `v0/migrate.go`) stay where they are — no cycle is possible against frozen
full-copy packages (§4.3) — and adopt the convention at their next bump.

**6.4 - Version directories become permanently pinned.** Full copies allowed, in
principle, deleting ancient version packages wholesale. The chain gives that up:
every version is reachable from current through aliases. Accepted — nothing prunes
these directories today, and a future compaction mechanism (§7) can restore the
ability if it is ever needed.

---

# 7 - Open Questions

1. **Chain compaction.** A future `@go` marker could force a bump to define every
   type, cutting the chain and making everything older prunable. Deferred until
   chain length is a demonstrated problem.
2. **Baseline resolution cost.** Resolving the latest snapshot declaring `M-1`
   loads historical snapshot resolutions per sync. If measurable, cache per-path
   baselines; not expected to matter at current snapshot counts.

---

# 8 - What This RFC Does Not Cover

- TypeScript, Python, C++, and protobuf outputs: the `types/vN` layout is Go-only
  (RFC 0041 §4.3), and this RFC changes nothing outside the Go generator.
- Snapshot cadence, `oracle snapshot`/`oracle migrate` CLI behavior, and the bump
  discipline in `detectBumps` — all unchanged.
- The runtime migration dispatch (`types/decode.go`) and gorp integration.
