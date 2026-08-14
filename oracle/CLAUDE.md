Go development rules for this module: @../docs/claude/toolchains/go.md

# Oracle

Schema-driven code generator. `.oracle` schemas in `/schemas/{synnax,arc,x}/` define
types once; plugins (`oracle/plugin/`: go, ts, py, cpp, pb, enum, domain, primitives,
framework) generate bindings for every language.

## Sync Workflow

Claude runs `oracle sync` itself. Before every sync, install the latest CLI first:

```bash
bash oracle/install.sh --cli-only --no-path && oracle sync
```

`./oracle/install.sh` is the ONLY blessed way to install oracle — never `go install`,
`go build`, or a hand-copied binary. It installs one global binary at
`~/.local/bin/oracle`; always reinstall from the same worktree you sync in, every time,
or generator logic and schemas disagree.

- Never sync on `rc` or any shared branch — sync writes generated files repo-wide. Sync
  only on the feature branch owning the schema changes.
- `oracle check` verifies generated files match schemas — read-only, safe anywhere.
- Confirm with the user before `oracle migrate` (version-affecting: scaffolds the next
  version file).

## Versioning Rules (RFC 0053)

- **Version files are hand-owned and hold the version story.**
  `schemas/<domain>/versions/<resource>/vN.oracle` enumerates the resource's complete
  persisted namespace at N: full declarations for shapes that changed at N, alias lines
  (`Key = v0.Key`, pointing at the defining version) for the rest. Absence means the
  type was removed at N. The versions directory is the sole version authority: there is
  no `@go version` tag; the current version is the highest vN file, and membership in it
  marks a type persisted. Version-owned content: fields, optionality, docs, `@key`, and
  the `@go` persistence set (`marshal` incl. field-level `omit`, `hand`, `migrate`,
  `imex`). Codecs are explicit: a struct or union gets one iff its declaration carries
  `@go marshal`; references never pull a codec in. `@go marshal hand` marks hand-written
  codec methods: nothing generates, references stay valid. Generation fails when a
  codec's persisted graph reaches a struct or union carrying neither form.
- **The live file is a generated projection, then an annotation surface.** Sync writes
  each versioned resource's live schema by merging chain resolution (version-owned
  content) with the live file's own annotations (outputs, `@ts`/`@py`/`@cpp` bindings,
  `@validate`, `@index`, `@pb`, wire-only types like `APIChannel`, and `action`
  declarations — actions are wire mutations, not persisted content, so the analyzer
  rejects them in version files). Edit shapes in version files; edit live-only concerns
  in the live file. A hand edit to version-owned live content is overwritten by the next
  sync, and the `versions` gate errors on the drift naming the version files as
  authority.
- **A resource is versioned iff its data is gorp-persisted.** Never version derived
  artifacts. A resource that stops being persisted ENDS its chain with a tombstone: an
  empty v(N+1).oracle (header and a comment only) records that everything was removed
  at N+1. An ended chain keeps every earlier version frozen (packages, codecs, fixture
  tests keep regenerating), has no current version, and its live file goes back to
  being hand-owned — sync no longer projects it. Nothing current may hold a stored
  reference into an ended chain; `oracle migrate` refuses it; declare members in a new
  version file to revive it. Example: arc `Program` ended at v1 — arc v0 records embed
  its v0 bytes, and arc v1+ resolve the live shape at read time.
- **Imports split on the persistence boundary.** A stored reference (part of the
  record's persisted bytes, e.g. a range's color) imports a pinned version file
  (`import "schemas/x/versions/color/v0"`). A resolved reference (read-time materialized
  `@go marshal omit` fields, e.g. a task's status) imports the dependency's live schema
  and always resolves its current surface. A live schema may import only its own
  resource's versions directory. The `versions` gate enforces placement, and fails the
  current surface on any stored pin lagging its dependency chain's current version.
- **Change workflow**: for a version that has never shipped in a release, edit the
  current vN file in place; for a shipped one, `oracle migrate <resource>` scaffolds
  v(N+1).oracle (alias lines to definers, omit-transient declarations redeclared,
  imports carried), then convert the changed types to full declarations and run
  `oracle sync`. Shipped-ness is developer knowledge; Oracle never tracks it.
- `oracle check`'s blocking `versions` gate enforces: live-file consistency (on-disk
  live file == merged projection), import placement, stored-pin currency, and delta
  minimality (a redeclaration structurally identical to its resolved predecessor must be
  an alias; enum member sets compare exactly).
- Frozen `versions/vN` Go packages are regenerated, checked outputs of the version
  files. Hand-written frozen definer files are marked `@go hand` in their version file;
  hand-ness is version-local — an aliased member whose definer is an older version still
  generates in the current package as a backward alias. `migrate.gen.go` is a pure
  function of the two adjacent version files; hand-written transforms live in
  `migrate.go` (renames and cross-resource moves have no generated counterpart).
- Omit-transient members — types reachable from a file's `@go marshal` declarations only
  through omitted fields, e.g. task's StatusDetails — track the live shape: they always
  declare fully, may match their predecessor (exempt from the minimality gate), and
  resolve their references at read time. A transient shape change to a shipped version
  requires a mint.
- `@go imex` (bare marker) on a versioned resource's root struct emits `imex.gen.go`
  files across the versions tree: a `Version imex.Version` constant in every
  `versions/vK` package the Core has exported (from the earliest version file carrying
  the marker up to the current version), plus `Latest` and an `autoDecodeEnvelope`
  ladder in the versions root that lifts server-era envelopes through the per-bump
  `Migrate<Type>` steps. Version files record the marker, so the chain dates the export
  history. Earlier version packages predate Core export and get no constant. The
  envelope version and migration chain are never hand-maintained; hand
  `versions/imex.go` files route `> legacy.LastVersion` envelopes to the ladder and keep
  only frozen Console-era decoding.

## Field Optionality

Four states. Pick by asking whether the field always means something, and when it does
not, whether the schema itself can tell.

- **`X = value`**: always meaningful, and absence is harmless, because absence and the
  default were always the same value. Most fields.
- **`X`** (bare, no `?`, no default): always meaningful, but no value is a safe guess.
  Required everywhere: TS parse fails, C++ raises "this field is required", and Python
  raises. Use for values only the author knows, such as a timestamp encoding.
- **Union variants**: the field applies only in some cases, and the deciding fact is in
  the same message. Split the type so the field exists only where it applies, then apply
  the two rules above inside each variant. Restructuring beats a validation rule. A new
  client forgets a rule, but cannot forget a shape.
- **`X?`**: the field applies only in some cases, and the deciding fact is outside the
  message, such as a channel data type. No schema can settle it, so absence is the
  honest encoding.

Never fake absence with a value. No `none` enum member, no empty string meaning "not
applicable". A sentinel claims the question was answered when it was not, and it makes
the "did you forget" check impossible to write.

`?` and `= value` are mutually exclusive; the analyzer rejects a field carrying both.

## Contextual Validation

A rule that needs facts outside the config, such as the data type of a referenced
channel, belongs in the Core service, on the write path only.

- The Core is the one layer every client passes through, so Python and TypeScript
  inherit the rule without writing any validation of their own.
- Never validate on read. Stored records predate the rule and must stay readable.
- The Driver keeps its own check as a second line, because channels drift between config
  time and deploy.
- The Console duplicates a rule only for better messages, never as the guarantee.

## Tag Minimization

Prefer the tagging that minimizes total tag count. When only a few types in a file need
a domain (@pb on control.Subject), tag those types and omit the file-level declaration;
when most types need it, declare it file-level and omit the exceptions.

## Omit vs Hand

- `@<lang> omit`: the type does not exist in that language. References to it from
  generating types are analyzer errors; a language output whose types are all omitted is
  an analyzer error (remove the output).
- `@<lang> hand`: the type exists, hand-written at the declared output path. No
  declaration is generated, but references resolve to it, pb translators generate
  against it, and it anchors the language output.

## Rules

- Never hand-edit generated output — edit the schema, then sync (see Sync Workflow). For
  a versioned resource's live schema this means: shapes in the version files,
  annotations in the live file.
- Before changing plugin output, mirror the nearest existing `.gen` analog — never
  invent new generated shapes.
- Wire-format or generator-wide changes hit every language at once: describe the blast
  radius and get explicit user sign-off first.
- Schema migrations: `oracle migrate` scaffolds the next version file; sync then
  regenerates `migrate.gen.go`, the frozen packages, and the live projection.
