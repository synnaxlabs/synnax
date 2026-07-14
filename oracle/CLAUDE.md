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

## Rules

- Never hand-edit generated output — edit the schema, then sync (see Sync Workflow).
- Before changing plugin output, mirror the nearest existing `.gen` analog — never
  invent new generated shapes.
- Wire-format or generator-wide changes hit every language at once: describe the blast
  radius and get explicit user sign-off first.
- Schema migrations: `oracle migrate` generates migration code, then runs sync.
