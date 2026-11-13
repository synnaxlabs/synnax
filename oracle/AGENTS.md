Go development rules for this module: @../docs/claude/toolchains/go.md

# Oracle

Schema-driven code generator. `.oracle` schemas in `/schemas/{synnax,arc,x}/` define
types once; plugins (`oracle/plugin/`: go, ts, py, cpp, pb, enum, domain, primitives,
framework) generate bindings for every language.

## Sync workflow

Claude runs `oracle sync` itself. Before every sync, install the latest CLI first:

```bash
bash oracle/install.sh --cli-only --no-path && oracle sync
```

`./oracle/install.sh` is the ONLY blessed way to install oracle — never `go install`,
`go build`, or a hand-copied binary. It installs one global binary at
`~/.local/bin/oracle`; always reinstall from the same worktree you sync in, every time,
or generator logic and schemas disagree.

- Never sync on `main` — sync writes generated files repo-wide. Sync only on the branch
  owning the schema changes.
- `oracle check` verifies generated files match schemas — read-only, safe anywhere.
- Confirm with the user before `oracle migrate` (version-affecting: scaffolds the next
  version file).

## Versioning rules (RFC 0053)

- **Version files are hand-owned and hold the version story.**
  `schemas/<domain>/versions/<resource>/vN.oracle` enumerates the resource's complete
  persisted namespace at N: full declarations for shapes that changed at N, alias lines
  (`Key = v0.Key`, pointing at the defining version) for the rest. Absence means the
  type was removed at N. The versions directory is the sole version authority: there is
  no `@go version` tag; the current version is the highest vN file, and membership in it
  marks a type persisted. Version-owned content: fields, optionality, docs, `@key`, and
  the `@go` persistence set (`marshal` incl. field-level `omit`, `wire`, `json`, `hand`,
  `migrate`, `imex`). Codecs are explicit: a struct or union gets one iff its
  declaration carries `@go marshal`; references never pull a codec in.
  `@go marshal hand` marks hand-written codec methods: nothing generates, references
  stay valid. Generation fails when a codec's persisted graph reaches a struct or union
  carrying neither form.
- **A full declaration can be a patch on an earlier version.** `X struct extends vK.X`
  (or a union of the same name) copies vK's declaration, resolved in vN's scope, then
  applies its body: an entry of the same name merges in place, `-field` removes one, and
  a new entry follows the nearest restated parent entry above it (a bare `name` restates
  a field), or ends the list. An `extends` of another name is a plain mixin.
- **The live file is a generated projection, then an annotation surface.** Sync writes
  each versioned resource's live schema by merging chain resolution (version-owned
  content) with the live file's own annotations (outputs, `@ts`/`@py`/`@cpp` bindings,
  `@validate`, `@index`, `@pb`, wire-only types like `APIChannel`, and `action`
  declarations — actions are wire mutations, not persisted content, so the analyzer
  rejects them in version files). Edit shapes in version files; edit live-only concerns
  in the live file. A hand edit to version-owned live content is overwritten by the next
  sync, and the `versions` gate errors on the drift naming the version files as
  authority. A `@doc` on a kept declaration that the chain leaves undocumented fails
  sync and check, and the error names the version file to move it to.
- **A resource is versioned iff its data is gorp-persisted.** Never version derived
  artifacts. A resource that stops being persisted ENDS its chain with a tombstone: an
  empty v(N+1).oracle (header and a comment only) records that everything was removed at
  N+1. An ended chain keeps every earlier version frozen (packages, codecs, fixture
  tests keep regenerating), has no current version, and its live file goes back to being
  hand-owned — sync no longer projects it. Nothing current may hold a stored reference
  into an ended chain; `oracle migrate` refuses it; declare members in a new version
  file to revive it. Example: arc `Program` ended at v1 — arc v0 records embed its v0
  bytes, and arc v1+ resolve the live shape at read time.
- **A resource whose stored data the Core deletes is removed whole.** Delete its live
  file, every version file, and its generated outputs (sync leaves them), and add a Core
  migration that deletes its stored entries. The `versions` gate passes a removed
  shipped chain only when its live file is gone too, no stored type imports it, and no
  current chain gained one of its stored types since the release.
- **Version files pin every versioned dependency.** A stored reference (a range's color)
  and a `@go marshal omit` field (a task's status) both import a pinned version file
  (`import "schemas/x/versions/color/v0"`), and the version package resolves them there.
  Only a chainless or ended dependency (arc's Program) imports its live schema. A live
  schema may import only its own resource's versions directory. The `versions` gate
  enforces placement, and fails the current surface on any stored pin lagging its
  dependency chain's current version.
- **Change workflow**: mint a new version only when the change makes the stored bytes on
  disk different from what a shipped release wrote. Every other change, including one to
  a shipped version, edits the current vN file in place: an unshipped version, an
  `@go marshal omit` field, a doc, or a validation rule. To mint,
  `oracle migrate <resource>` scaffolds v(N+1).oracle (alias lines to definers,
  omit-transient declarations redeclared, imports carried), then convert the changed
  types to full declarations and run `oracle sync`. A version is shipped when the
  shipped release holds its file: the newest stable Core tag (`core/vX.Y.Z`) on the X.Y
  train of the newest stable tag that HEAD descends from.
- `oracle check`'s blocking `versions` gate enforces: live-file consistency (on-disk
  live file == merged projection), import placement, stored-pin currency, and delta
  minimality (a redeclaration structurally identical to its resolved predecessor must be
  an alias; enum member sets compare exactly), and rule parity (a type the current
  version aliases from a frozen definer has the same `@validate`, `@default`,
  `@go fields`, `@go imports`, and `@go to_any` in the live file and the definer; each
  dependency type such a definer reaches through its pins, at its current version, has
  the same `@validate` and `@default` in its live file and its version file), and
  shipped versions (each stored type of a version file the shipped release holds must
  still decode the data that release wrote). The shipped check fills in each generic
  type with its type arguments (defaults included) through aliases, distinct types, and
  bases, then compares the filled-in shapes across pins: stored field names and order,
  field types, optionality, `@key`, defaults, and enum and union members. A type may
  move to another file or chain or become an alias with the same filled-in shape, and an
  enum or union may gain members. Docs, rules, codec markers, and omit-transient types
  may change.
- Frozen `versions/vN` Go packages are regenerated, checked outputs of the version
  files. In current and frozen packages alike, a stored reference resolves to the
  dependency version that the declaring version file imports, so a freeze never moves a
  generated import. A frozen package takes its `@validate` rules, `@default` groups,
  `@go fields`, `@go imports`, and `@go to_any` from its version files only, never from
  the live file. The current package's alias of a frozen type is that frozen type, so Go
  reads an aliased type's rules from its definer while the other languages read the live
  file. Restate each live rule in the definer's version file; the rule parity gate fails
  until you do. Current and frozen packages keep the version file's declaration order,
  declare every member it records even when the live file marks it `@go omit`, and
  resolve every dependency reference at the version it pins, never at the root. A
  version package therefore reaches only declarations its version files fix, so it
  compiles and stands still until a version file changes. A mint demotes the current
  package, so it loses every live-only rule unless a version file restates it. A frozen
  package also reads a pinned dependency type's rules from its version file only, so a
  live-only rule on that type (label's `name`) drops the frozen package's `Validate`
  call into it. Restate such a rule in the dependency's version file too; the rule
  parity gate fails until you do. Hand-written frozen definer files are marked
  `@go hand` in their version file; hand-ness is version-local — an aliased member whose
  definer is an older version still generates in the current package as a backward
  alias. `migrate.gen.go` is a pure function of the two adjacent version files;
  hand-written transforms live in `migrate.go` (renames and cross-resource moves have no
  generated counterpart).
- Omit-transient members — types reachable from a file's `@go marshal` declarations only
  through omitted fields, e.g. task's StatusDetails — track the live shape: they always
  declare fully, may match their predecessor (exempt from the minimality gate), and
  resolve their references at the pinned version. A transient shape change never
  requires a mint, because it does not change the stored bytes.
- `@go imex` (bare marker) on a versioned resource's root struct emits `imex.gen.go`
  files across the versions tree: a `Version imex.Version` constant in every
  `versions/vK` package the Core has exported (from the earliest version file carrying
  the marker up to the current version), plus `Latest` and an `autoDecodeEnvelope`
  ladder in the versions root that lifts server-era envelopes through the per-bump
  `Migrate<Type>` steps. Version files record the marker, so the chain dates the export
  history. Earlier version packages predate Core export and get no constant. The
  envelope version and migration chain are never hand-maintained; hand
  `versions/imex.go` files route `> legacy.LastVersion` envelopes to the ladder and keep
  only frozen Console-era decoding, which builds the first exported shape and enters the
  ladder through the generated `liftFirst`, so a version bump never edits it.
  `@go imex body` marks a type another resource exports, such as a task config inside a
  task file: no `Export`, several types may share a package, and each gets an exported
  `<Type>Ladder` (`imex.Ladder`) over body blobs. A version that redeclares a marked
  type must declare its exported `Migrate<Type>` step.

## Field optionality

Five states. Pick by asking whether the field always means something, and when it does
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
  honest encoding. On the wire, an absent field is left out of the JSON, never sent as
  `null`.
- **`X = default`, the default means "not applicable"**: as `X?`, but no valid value can
  equal the default or the type's zero value (`0`, `""`, or an enum member that this
  field never holds). Go fills a default only over the zero value. A sample rate is
  never `0` Hz, so `0` can mean "not supported". A client that leaves the field out
  sends "not applicable", so use this only where that guess is harmless. The doc comment
  says what the default means.

Outside that case, never fake absence with a value. No `none` enum member, no `""`
meaning "not applicable" where `""` is a real value, no `-1` meaning "inherit" (that is
a behavior, not "not applicable"). A sentinel claims the question was answered when it
was not, and it makes the "did you forget" check impossible to write. Never add a value
to a type so that it can mean absence.

An `@ref` scalar with a zero default (`= 0`, `= ""`) holds no target while it is zero.
This is the "not applicable" default above, so use it only where no valid key is zero. A
`detach` reference needs `X?`, an array, or a zero default. A required field in an array
element also detaches: the detach drops the element. The same rules hold for top-level
and nested fields.

`?` and `= value` are mutually exclusive; the analyzer rejects a field carrying both.

`@py optional` makes a field optional in Python alone. Use it for an `any` field that
must read an absent key as `None`, because the Python JSON codec drops `None` fields.

## Action aliases

`action Set = patch.Set` takes an existing struct as the payload, with the struct's doc.
A body of domains only, `action Set = patch.Set { @doc value "..." }`, gives the action
its own doc in place of the struct's. Generators keep the `SetPayload` name as an alias
(in Go, a defined type that can carry `Handle`). The target must be a struct with no
type parameters, and no two actions of a type may share one.

## Union discriminators

A union's discriminator is a generated `<Union>Type` with one constant per variant. When
a declared enum already names the same set, borrow it:
`Proof union on method Method { ... }`. The discriminator then has the enum's type in
every language, the variant constants are the enum's members, and no `<Union>Type` is
emitted. The enum must be a string enum whose values are exactly the variant names; the
analyzer rejects a variant without a member and a member without a variant.

## Union bases

When a union extends one base struct and every variant inherits it, Oracle generates
accessors for the base: `Base()` and `SetBase(b)` on the Go wrapper, and a read-only
`base(const U&)` in the C++ namespace. Never write these by hand. A union with no base,
with two or more bases, or with a variant that drops a base field (`-port`) gets none.

- `Base()` returns the zero base when `Variant` is nil. A nil variant is valid decoded
  input (`null`), and readers such as validators must not crash on it.
- `SetBase` panics when `Variant` is nil. There is no variant to hold the base, and a
  dropped write would lose data with no error.
- C++ has no nil case, because a `std::variant` always holds an alternative.

## Default groups

`@default group "<name>"` on two or more fields makes their defaults fill as a unit: the
generated fill runs only when every member holds its zero value. Use it when a field's
zero is valid on its own but the group's all-zero state is not, such as a `min_val` /
`max_val` pair, where an all-zero range means "unset" but a `max_val` of 0 is a real
bound the fill must not overwrite.

Grouping only changes Go. TypeScript, Python, and C++ fill from key presence at decode
and never overwrite a value the caller sent.

## Contextual validation

A rule that needs facts outside the config, such as the data type of a referenced
channel, belongs in the Core service, on the write path only.

- The Core is the one layer every client passes through, so Python and TypeScript
  inherit the rule without writing any validation of their own.
- Never validate on read. Stored records predate the rule and must stay readable.
- The Driver keeps its own check as a second line, because channels drift between config
  time and deploy.
- The Console duplicates a rule only for better messages, never as the guarantee.

## Tag minimization

Prefer the tagging that minimizes total tag count. When only a few types in a file need
a domain (@pb on control.Subject), tag those types and omit the file-level declaration;
when most types need it, declare it file-level and omit the exceptions.

## Omit vs hand

- `@<lang> omit`: the type does not exist in that language. References to it from
  generating types are analyzer errors; a language output whose types are all omitted is
  an analyzer error (remove the output). One exception: a live `@go omit` on a member
  that a version file records only keeps it out of the root package. Its `versions/vN`
  package still declares it.
- `@<lang> hand`: the type exists, hand-written at the declared output path. No
  declaration is generated, but references resolve to it, pb translators generate
  against it, and it anchors the language output.
- Field-level `@<lang> omit` (ts, py, cpp, pb): the field does not exist in that
  language's type, codecs, translators, or action payloads. An override inherits its
  parent field's omit and cannot add one; a pb message reserves the omitted number.
  Never on a `@key` field, never in a language the type does not generate, and never in
  every language the type generates.
- Go has no field-level omit. `@go marshal omit` keeps a field out of the codec;
  `@go wire omit` keeps it out of JSON and msgpack (`json:"-" msgpack:"-"`), so the API
  can return a stored record without a secret. Both are version-owned.

## Path actions

`@patch` marks a field of a document (a struct with `action Set = patch.Set`) that path
actions may edit. A path to any other field fails, so a Core-owned field such as
`labels` gets no tag. Go generates a `Patch` method for the document and for every
struct that a `@patch` field reaches, in the version package that declares the struct. A
struct with a hand-written Go JSON decoder, such as `color.Color`, declares
`@go json hand` and gets no `Patch` method. Only a whole set changes it, and that set
runs the decoder.

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
