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
- Confirm with the user before `oracle migrate` / `oracle snapshot`
  (schema-version-affecting).

## Versioning Rules

- **Version a schema (`@go version`) iff its data is gorp-persisted** — directly, via
  ImEx, or by being embedded in a versioned schema. Nothing else needs migrations: wire
  peers are never version-skewed, and there are no unvalidated caches.
- **Never version derived artifacts** (compiled output like arc `Program`). On mismatch
  they are recomputed from their versioned sources, not migrated.
- `@go version` is type-granular and must be declared per type; the analyzer rejects
  file-level declarations. Declare it struct-level on persisted types (channel-style).
  Unversioned siblings are transient — they generate real declarations at the package
  root (merged into `types.gen.go` beside the version aliases) instead of riding the
  versions/vN layout, and their shape changes never force a version bump.
- Two classes must stay versioned despite being unpersisted: types referenced by a
  versioned sibling (even via `@go marshal omit` fields — their Go home cannot leave the
  package without an import cycle; the persistence gate exempts these automatically),
  and types whose hand-written Go methods entangle with versioned siblings (telem's
  Size/Rate). Mark the latter `@go version N pinned` — the gate skips pinned types and
  warns if a pinned type is actually persisted; the analyzer rejects any other version
  argument.
- `oracle check` runs a non-blocking persistence gate warning on versioned types outside
  the persisted closure and on persisted types missing @go version at a versioned path.
  Use `--verbose` to see warnings on passing gates.
- Migrate wrapper visibility follows consumption (see `plugin/go/migrate`): exported
  when another versioned schema embeds the type; unexported when only the package's own
  gorp wiring or auto-copies call it.
- `@go imex` (bare marker, requires `@go version`) on an imex-registered resource's
  root struct emits `imex.gen.go` with a `Version imex.Version` constant equal to the
  schema version — the portable envelope version is never hand-maintained.

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

- Never hand-edit generated output — edit the schema, then sync (see Sync Workflow).
- Before changing plugin output, mirror the nearest existing `.gen` analog — never
  invent new generated shapes.
- Wire-format or generator-wide changes hit every language at once: describe the blast
  radius and get explicit user sign-off first.
- Schema migrations: `oracle migrate` generates migration code, then runs sync.
