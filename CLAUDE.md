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

- **PNPM** for TypeScript packages
- **Turbo** for build orchestration
- **Go workspace** for Go modules
- **Poetry** for Python packages
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
poetry install     # Install dependencies
poetry run pytest  # Run tests
poetry run black . # Format code
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
- Poetry for package management

### C++

- RAII for resource management
- Smart pointers, no raw pointers
- Bazel `select()` for platform-specific code
- Google Test with custom xtest utilities

## Getting Help

- `/help` - Get help with using Claude Code
- Report issues: https://github.com/anthropics/claude-code/issues

## Self-Editing Guidelines

When provided with useful context from humans that would benefit future interactions:

- Make minimal, sparing edits to preserve context
- Only add genuinely useful information for development work
- Keep additions concise and relevant to the codebase
- Prefer editing specific documentation files over this main file
