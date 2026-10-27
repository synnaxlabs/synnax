# Theme: docs drift

Find docs that still describe an API the way it was before a recent change.

1. List the PRs merged in the last 7 days:
   `gh pr list --state merged --search "merged:>=<date>" --json number,title,files`.
2. Keep the PRs that change a surface users read about: the client libraries
   (`client/ts`, `client/py`, `client/cpp`), the Core CLI, Console behavior, Arc, the
   Driver task configs, and anything under `schemas/`.
3. For each changed name, flag, field, or behavior, search the docs for the old form:
   `site/docs`, package `README.md` files, `docs/claude/`, `AGENTS.md` files, and the
   doc comments of the changed code.
4. Update each stale doc to match the code on `main`. Read the code to confirm; never
   write from the PR title alone.

Follow `site/docs/CONTRIBUTING.md` for docs pages. Write in ASD-STE100 and sentence
case. Do not add docs for a feature behind a flag in `console/src/flags.ts` or
`site/docs/src/flags.ts` unless the docs page has the same flag.

Title docs PRs `[docs] ...` and label them `review/bot`. One PR per merged PR whose docs
drifted.
