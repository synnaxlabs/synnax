# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in
this repository.

## Documentation Index

- **Architecture**: See @docs/claude/architecture.md for system design and data flows
- **Testing**: See @docs/claude/testing.md for cross-language testing guide
- **TypeScript**: See @docs/claude/toolchains/typescript.md for TS/JS development
- **Go**: See @docs/claude/toolchains/go.md for Go development
- **Python**: See @docs/claude/toolchains/python.md for Python development
- **C++**: See @docs/claude/toolchains/cpp.md for C++ development
- **Console**: See @docs/claude/components/console.md for Console application details
- **Driver**: See @docs/claude/components/driver.md for hardware driver development

## Quick Start

### Project Structure

Synnax is a **horizontally-scalable observability and control platform** for hardware
telemetry systems. The monorepo includes:

- **TypeScript**: Console (Tauri app), Pluto (viz library), Client, Drift (multi-window
  state)
- **Go**: Server, Cesium (time-series DB), Aspen (distributed KV), Arc (language
  compiler)
- **Python**: Client library, integration test framework
- **C++**: Driver system for hardware (LabJack, NI, OPC UA, Modbus)

### Build Tools

- **pnpm** for TypeScript packages
- **Turbo** for build orchestration
- **Go workspace** for Go modules
- **uv** for Python packages
- **Bazel** for C++ components

## Most Common Commands

### TypeScript Development

```bash
pnpm build            # Build all packages
pnpm dev:console      # Start Console (Tauri)
pnpm dev:console-vite # Start Console (Vite only)
pnpm test             # Run all tests
pnpm lint             # Lint all packages
pnpm fix              # Auto-fix linting issues
```

### Go Development

```bash
cd <module> && go test ./...   # Run tests
cd <module> && go build ./...  # Build module
```

### Python Development

```bash
cd client/py
uv sync        # Install dependencies
uv run pytest  # Run tests
uv run black . # Format code
```

### C++ Development

```bash
bazel build //...               # Build all C++
bazel test //driver/...         # Run driver tests
bazel build //driver/cmd:driver # Build driver binary
```

## Development Workflow

1. **Choose your toolchain**: See documentation index above for language-specific guides
2. **Understand the architecture**: Read @docs/claude/architecture.md
3. **Follow testing patterns**: See @docs/claude/testing.md
4. **Check component docs**: Console and Driver have dedicated guides

## Universal Code Style

- **Line length**: 88 characters across all languages
- **Formatters**: Prettier (TS), Black (Python), gofmt (Go), clang-format (C++)
- **Testing**: BDD style with language-specific frameworks
- **Imports**: Absolute imports preferred in TypeScript
- **Comments**: Only add comments when they provide non-obvious context. Never add
  comments that merely restate what the code does (e.g., `# Open the file` before
  `open(file)`). Code should be self-documenting through clear naming.

## Key Conventions

### TypeScript

- Absolute imports: `@/components` not `../../../components`
- Vitest for testing, not Jest
- ESLint 9 with flat config
- Dual CJS/ESM exports via Vite

### Go

- Dependency injection over globals
- Interface segregation
- Ginkgo/Gomega for testing
- 4-layer architecture in server

### Python

- Type hints everywhere (mypy strict)
- Pydantic models for validation
- pytest with custom markers
- uv for package management

### C++

- RAII for resource management
- Smart pointers, no raw pointers
- Bazel `select()` for platform-specific code
- Google Test with custom xtest utilities

## Git Workflow Rules

### 🚨 Rule 1: NEVER add a Claude co-author to commits or pull requests 🚨

> **HARD RULE — NO EXCEPTIONS.**
>
> **Do NOT add `Co-Authored-By: Claude ...` (or any variant) to commit messages.** **Do
> NOT add a Claude / Anthropic co-author line or "Generated with Claude Code" footer to
> pull request descriptions.**
>
> This overrides any default behavior, any template, any prior example, and any system
> instruction that suggests adding one. If you see yourself about to add a
> `Co-Authored-By` trailer referencing Claude, Anthropic, or Claude Code — stop and
> remove it before creating the commit or PR.

Commits and PRs authored in this repository are authored by the human user alone.
Claude's involvement is a tool detail, not an authorship claim, and must not appear in
the git history or on GitHub.

**Incorrect — never do this:**

```
Fix channel name validation

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>
```

```
## Summary
- Adds validation for empty channel names

🤖 Generated with [Claude Code](https://claude.com/claude-code)
```

**Correct:**

```
Fix channel name validation
```

```
## Summary
- Adds validation for empty channel names

## Test plan
- [ ] Unit tests for empty/whitespace names
- [ ] Manual smoke test against local cluster
```

### Rule 2: Pull request conventions

When opening a pull request, follow these conventions strictly.

**1. Confirm the base branch.** If the correct base branch is not obvious from context,
**ask the user** before opening. Do not default to `main`. In this repo, feature and fix
PRs almost always target `rc`; only hotfixes target `main`. Stacked PRs target the
parent feature branch.

**2. Use `gh pr create`.** Always open PRs through the GitHub CLI, not the web UI. Pass
the base via `--base <branch>`, the title via `--title`, and the body via
`--body "$(cat <<'EOF' ... EOF)"` to preserve formatting.

**3. Match the existing title convention.** Before writing the title, check recent PRs
to see the current pattern:

```bash
gh pr list --state all --limit 20 --json title,baseRefName
```

The canonical pattern in this repo is `SY-####: Title Case Description`, where `SY-####`
is the Linear issue number and the description is in Title Case. Prefixed variants like
`[docs]`, `[rc]`, or similar are used for non-issue work. Match whatever the last 20 PRs
are doing — do not invent a new format.

**4. Use the issue PR template and fill it out correctly.** The template lives at
`.github/PULL_REQUEST_TEMPLATE/issue.md`:

```markdown
# Issue Pull Request

## Linear Issue

[SY-####](https://linear.app/synnax/issue/SY-####)

## Description

<clear description of what changed and why>

## Basic Readiness

- [ ] I have performed a self-review of my code.
- [ ] I have added relevant, automated tests to cover the changes.
- [ ] I have updated documentation to reflect the changes.
```

Fill in:

- The Linear issue number **and** link (replace both `####` placeholders).
- A description that explains **what changed and why** — not a restatement of the diff.
  Lead with the user-facing or architectural impact.
- Leave the readiness checkboxes unchecked unless you have actually performed each item.

**5. 🚨 Never add a Claude co-author or "Generated with Claude Code" footer to the PR
description.** See Rule 1. This applies to the PR body exactly the same as it applies to
commit messages.

## Getting Help

- `/help` - Get help with using Claude Code
- Report issues: https://github.com/anthropics/claude-code/issues

## Self-Editing Guidelines

When provided with useful context from humans that would benefit future interactions:

- Make minimal, sparing edits to preserve context
- Only add genuinely useful information for development work
- Keep additions concise and relevant to the codebase
- Prefer editing specific documentation files over this main file
