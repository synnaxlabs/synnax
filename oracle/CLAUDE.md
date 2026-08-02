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
- Confirm with the user before `oracle migrate` (version-affecting: mints or amends
  version files).

## Versioning Rules (RFC 0047)

- Version history lives in explicit version files: `schemas/<domain>/versions/
  <resource>/vN.oracle`. Each file enumerates the resource's complete PERSISTED
  namespace at that version — full declarations for shapes that changed at N, alias
  lines (`Key = v0.Key`, pointing at the defining version) for the rest. Absence
  means the type was removed at N. The versions directory is the sole version
  authority: there is no `@go version` tag; the current version is the highest vN
  file, and membership in it marks a type persisted.
- **A resource is versioned iff its data is gorp-persisted.** Never version derived
  artifacts (arc `Program`); resources that leave the persisted world close their
  chain with an EMPTY current file (see `schemas/arc/versions/program/v1.oracle`).
- Import rules: inside `versions/`, only other version files may be imported (dep
  pins like `import "schemas/x/versions/telem/v0"`, computed at mint); a live schema
  may import only its own resource's versions directory. Version files are
  persisted-only — a field carrying `@go marshal omit` is an analyzer error there.
  Type-level `@go marshal` / `@go migrate` live in version files, not live schemas
  (field-level `@go marshal omit` stays live).
- `oracle migrate <resource>` mints the next version file from the live persisted
  shapes and syncs; `oracle migrate --amend <resource>` rewrites the current file in
  place — only for versions that have never shipped in a release. Bare
  `oracle migrate` mints every drifted resource.
- `oracle check`'s blocking `versions` gate enforces: chain coverage, byte-identical
  drift (current file == canonical emission), and delta minimality (a redeclaration
  structurally identical to its resolved predecessor must be an alias; enum member
  sets compare exactly).
- Frozen `versions/vN` Go packages are regenerated, checked outputs of the version
  files (persisted-only; omit fields exist only in current packages). Hand-written
  frozen definer files are marked `@go hand` in their version file. `migrate.gen.go`
  is a pure function of the two adjacent version files; hand-written transforms live
  in `migrate.go` (renames and cross-resource moves have no generated counterpart).
- Types versioned despite being unpersisted (hand-method entanglement, e.g. telem's
  Size/Rate, or sibling-referenced transient types like task's StatusDetails) are
  declared in the current version file with `@go pinned`: pinned members always
  declare fully, track the live shape, and are exempt from the minimality gate.

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
