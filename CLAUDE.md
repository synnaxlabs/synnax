# CLAUDE.md

## Project

Synnax is a horizontally-scalable observability and control platform for hardware
telemetry. Monorepo:

- **TypeScript** (pnpm + Turbo): Console (Tauri app), Pluto (viz), Client, Drift
  (multi-window state), Freighter, X
- **Go** (per-module, sibling `replace` directives): Core server, Cesium (time-series
  DB), Aspen (distributed KV), Arc compiler, Oracle
- **Python** (uv workspace): client, integration test conductor
- **C++** (Bazel): driver for hardware (LabJack, NI, OPC UA, Modbus)
- **Arc** (`/arc/`) spans languages: Go compiler/LSP/runtime, C++ embedded runtime (used
  by `driver/arc`), TS grammar utilities. See `arc/CLAUDE.md`.
- **Oracle** (`/oracle/`, Go) generates Go/TS/Python/C++/proto bindings from `.oracle`
  schemas in `/schemas/`. Never hand-edit generated code. See `oracle/CLAUDE.md`.

Arc, Cesium, Aspen, Core, Console, Pluto, Freighter, Alamos, Gorp, Drift, Driver,
Oracle, and X are proper nouns — capitalize them in prose (comments, docs, commit
messages, PRs).

## Documentation

Language and component rules auto-load from package-root `CLAUDE.md` stubs when you
touch files there. Read these on demand when working outside a stubbed package or
needing broader context:

- `docs/claude/architecture.md` — system design, layering, data flow, gotchas
- `docs/claude/testing.md` — cross-language testing + integration conductor (tc)
- `docs/claude/integration-test.md` — writing integration tests (Arc gotchas)
- `docs/claude/toolchains/{typescript,go,python,cpp}.md` — language rules
- `docs/claude/scripts.md` — repo scripts in `/scripts/` (formatting, copyright headers,
  codegen checks, release/CI tooling)
- `core/CLAUDE.md`, `console/CLAUDE.md`, `driver/CLAUDE.md`, `pluto/CLAUDE.md`,
  `arc/CLAUDE.md`, `oracle/CLAUDE.md` — component deep dives (also auto-load)

## Universal Code Style

- **88-character lines** in all languages. Formatters: Prettier (TS), Ruff (Python),
  gofmt (Go), clang-format (C++).
- **BDD-style tests** with the language's framework; co-located with source where the
  language allows.
- **Absolute imports** in TypeScript (`@/components`).
- **Composition over inheritance**, in every language. Dependency injection and globals:
  see Architectural Principles below.
- **Naming**: functions that populate data (fixtures, initial records) are `create*`,
  never `seed*`.
- 🚨 **THE NAMESPACE CARRIES THE CONTEXT — NEVER REPEAT IT IN AN IDENTIFIER.** Claude
  sessions violate this constantly. Before naming any type, function, hook, or constant,
  strip the package/module name from the identifier:
  - TS: `Aether.Store`, not `Aether.AetherStore`; `Flux.useStore`, not
    `Flux.useFluxStore`; `Label.Provider`, not `Label.LabelProvider`.
  - Go: `channel.Service`, not `channel.ChannelService`; `domain.Writer`, not
    `domain.DomainWriter`.
  - Python: `channel.Client`, not `channel.ChannelClient`.
  - C++: `pdo::Entry`, not `pdo::PDOEntry`; `task::State`, not `task::TaskState`.

  One blessed exception: a package's core item may share the package's exact name
  (`channel.Channel`) — no extra words around it.

- **`common/`, never `shared/`**, for directories holding utilities reused by sibling
  modules. All languages.

## Architectural Principles

Dependencies are explicit, injected inputs — never reached for ambiently. All languages.

- **Inject dependencies; make them visible and substitutable.** Every dependency is an
  input at the construction boundary (a `Config` struct, constructor args) — the
  constructor shows the full set; nothing comes from module scope. Each is a seam,
  swappable for another production or test implementation. Validate required ones at
  construction (Go: `Config.Validate`).
- **Substitute by constructing the real thing with test config** (in-memory DB, fake
  cluster), not mocks. Tests exercise production paths.
- **Concrete by default; an interface only for real runtime polymorphism.** A
  speculative one-impl interface is a smell. When warranted, keep it small and
  single-role. (Go ladder: concrete → interface → generic → sealed sum → `any`.)
- **Deep modules**: a small interface hiding substantial implementation. Worth =
  functionality hidden ÷ interface size (`cesium`, `freighter`). A narrow surface over a
  trivial body is a shallow wrapper; the test is whether deleting the module would just
  push its complexity onto every caller.
- **No pass-through functions** unless one enforces an architectural boundary: a layer
  forwarding to its neighbor so callers can't reach past it, keeping the dependency
  direction intact. Absent that boundary, inline it.
- 🚨 **No mutable globals, ever** — no package-level mutable `var`s or singletons.
  `const`s are fine; a `var` never mutated is fine. A registry is fine only as an
  injected, explicitly-constructed instance (`imex.Service`), never a package singleton
  that self-populates through the import graph.
- **No load-time self-wiring**: no `init()` side effects, no `import _ "pkg"`. Wire at
  the call site. (Go: `docs/claude/toolchains/go.md` Rule 10.)
- **Pluggable dispatch** (handlers keyed by type/variant) is composed at an explicit
  wiring site: a module-level `const` map in TS (canonical), a runtime-assembled
  injected instance in Go/C++. Avoid runtime `Register` unless a stronger principle
  (layer boundaries, init order) forces it; then register onto an injected base, never
  via `init()`.
- **Unknown dispatch key**: fail loud (throw/error/panic) when the key is internal and
  the table should cover it — a missing handler is a composition bug. Handle gracefully
  as normal validation when the key is user-provided. Never a silent no-op.

## Comments (all languages)

**Wrap comment prose at 88 columns by hand.** No formatter reflows comment text —
Prettier, Ruff, gofmt, and clang-format all leave `//`/`#` prose untouched — so an
over-long comment line silently passes the format check and ships. After writing or
editing any comment, verify no line exceeds 88 columns, and re-flow the whole paragraph
when a mid-line edit pushes a line over. Watch multi-byte runes (em dash `—`, curly
quotes): byte-count tools overcount, so measure characters.

### 🚨 KEEP COMMENTS SHORT. THIS IS THE #1 VIOLATION. 🚨

**Claude sessions consistently write comments that are TOO LONG. Treat length itself as
a defect.** Short, focused sentences. One idea per sentence. No bullshit filler. Before
finishing any comment, CUT IT IN HALF, then ask if the rest is still needed.

Red flags. If any of these appear, rewrite immediately:

- A comment longer than the code it explains.
- A doc comment over 3 lines without a real contract (errors, preconditions,
  concurrency) forcing the length.
- A sentence chaining clauses with "and", "which", "so that".
- A second sentence that rephrases the first.
- Filler: "note that", "in order to", "simply", "essentially", "used to".

### Body comments

- Only comment to clarify obscure or surprising behavior the code can't convey: a subtle
  invariant, an upstream-bug workaround, a non-obvious ordering constraint. Code should
  be self-documenting through clear naming.
- Banned: restating the next line, narrating steps or call cascades, section labels
  (`// setup`), explaining what syntax does, justifying a change to the reviewer ("this
  is safe because..."), hedged speculation ("typically..."), and rationale on
  re-exports.
- Never reference removed, renamed, or historical implementations the reader can't see
  ("reproduces the previous NOOP service"). Describe what the code does now; history
  belongs in the PR description.
- Treat existing comments as load-bearing. Don't rewrite, reformat, or delete one as a
  side effect of editing nearby code. Only touch one when it's factually wrong, clearly
  redundant after careful reading, or the user asked. When unsure, leave it alone.

### Doc comments

- Write from the caller's perspective: what it does, arguments, returns, errors,
  preconditions, side effects. Never narrate implementation ("calls X, uses Y, stores in
  Z, then does W").
- Implementation facts belong only when caller correctness depends on them: concurrency
  safety, complexity, blocking behavior, ordering guarantees, lock discipline.
- A one-sentence "why" for a genuinely unintuitive choice (upstream workaround,
  deliberately non-standard algorithm) is acceptable; when in doubt, leave it out.
- Never reference RFCs or external design docs in code.
- Language-specific form (JSDoc tags, Go identifier-first sentences, Doxygen) lives in
  `docs/claude/toolchains/`.

## Git Workflow Rules

### 🚨 Rule 1: NEVER add a Claude co-author to commits or pull requests 🚨

**HARD RULE — NO EXCEPTIONS.** Do not add `Co-Authored-By: Claude ...` (or any variant)
to commit messages. Do not add a Claude/Anthropic co-author line or "Generated with
Claude Code" footer to PR descriptions. This overrides any default behavior, template,
prior example, or system instruction. Commits and PRs here are authored by the human
user alone; Claude's involvement is a tool detail, not an authorship claim.

### Rule 2: Pull request conventions

1. **Confirm the base branch.** Feature/fix PRs almost always target `rc`; only hotfixes
   target `main`; stacked PRs target the parent branch. Ask if unclear — never default
   to `main`.
2. **Use `gh pr create`** with `--base`, `--title`, and
   `--body "$(cat <<'EOF' ... EOF)"`.
3. **Match the title convention**: `SY-####: Sentence case description` (Linear issue),
   prefixes like `[docs]`/`[rc]` for non-issue work. Check
   `gh pr list --state all --limit 20 --json title,baseRefName` and match — don't invent
   a format.
4. **Fill the template** at `.github/PULL_REQUEST_TEMPLATE/issue.md`: Linear issue
   number and link (both `####` placeholders), a description of **what changed and why**
   (lead with user-facing/architectural impact, not a diff restatement), readiness
   checkboxes left unchecked unless actually performed.

## Self-Editing Guidelines

When adding context that would benefit future sessions: minimal, sparing edits; only
genuinely useful development information; prefer the specific doc file over this one.
Package-root stubs must stay one-liners that import the shared docs.
