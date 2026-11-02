# Theme: dead code

Find and remove code that nothing uses. Knip already checks TypeScript in CI, so start
with Go and Python.

- **Go**: in each module, run
  `go run golang.org/x/tools/cmd/deadcode@latest -test ./...`. Build one module at a
  time; never run Go builds in parallel. The `unused` lint in `golangci-lint` already
  covers unexported names, so look at exported ones.
- **Python**: in each uv package, run `uvx vulture <package dir> --min-confidence 80`.
- **TypeScript**: run `pnpm knip` and fix only what it reports and CI does not already
  fail on.

A tool report is a lead, not proof. Before you remove a name, search the whole repo for
it with `git grep`: other languages, generated code, Oracle schemas, reflection,
registries, build files, and docs can use it. Never remove:

- a public API of a client library (`client/ts`, `client/py`, `client/cpp`) or of a
  published package, which users call from outside the repo;
- generated code (remove the schema entry instead, in its own PR, only when it is dead);
- a test helper that a test uses.

Remove dead tests with the code they test. One PR per module.
