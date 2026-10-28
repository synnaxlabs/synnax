# Weekly lint pass

You are on a fresh clone of `main` of the Synnax monorepo. Nobody watches this session,
so do not ask questions. Your job: find conventions that the prose rules state but no
linter checks, and turn the clearest ones into blocking lint rules.

## Read first

1. `CLAUDE.md`, `CONTRIBUTING.md`, `scripts/CLAUDE.md`, `docs/claude/testing.md`, and
   `docs/claude/toolchains/{go,typescript,python,cpp}.md`. Follow them.
2. The other prose rules: package `CLAUDE.md` files, `docs/claude/**`,
   `.claude/skills/**/SKILL.md`, and `.greptile/`.
3. The lint setup:
   - Go: the root `.golangci.yaml`, the custom analyzers in `x/go/analyzers`, and the
     `.custom-gcl.yaml` of each module. `golangci-lint custom` builds `./custom-gcl`.
   - TypeScript: `configs/eslint/eslint.config.ts` and its local rules in
     `configs/eslint/rules`, and `knip.json`.
   - Python: the Ruff settings in each `pyproject.toml`. Ruff has no plugins, so a rule
     it lacks is a small AST check script run in CI, with tests under a `tests/`
     directory.
   - C++: `.clang-tidy` and `.clang-format`.
   - Repo scripts in `scripts/` and `.github/scripts/` that check conventions.

## Step 1: finish rules already in flight

Some rules exist but are not on yet: an analyzer built into `custom-gcl` but missing
from `enable` in `.golangci.yaml`, or an ESLint rule set to `warn`. For each one, check
for open PRs that fix its violations (`gh pr list --search "<rule name>"`).

- No violations remain and no fix PR is open: open the PR that turns it on.
- Violations remain and no fix PR covers them: open the fix PRs (step 4).
- Fix PRs are open: leave it.

## Step 2: list candidates

List conventions that are:

1. stated in the prose rules,
2. checkable by a machine with few false positives, and
3. not checked by any linter or script today.

Rank them by how often the code breaks the rule today (count the violations) and how
often reviewers flag it. To measure review signal, read all review comments of the last
30 days, Greptile's included, page by page:
`gh api --paginate "repos/{owner}/{repo}/pulls/comments?since=<date 30 days ago>&per_page=100"`.
Skip a candidate that already has an open PR or Linear issue.

## Step 3: add a rule

For each of the top candidates, add the rule in the right tool, in its own PR:

- Prefer a built-in rule (an ESLint core or plugin rule, `no-restricted-syntax`, a Ruff
  rule code, a golangci-lint linter such as `depguard` or `forbidigo`, a clang-tidy
  check) over custom code.
- When none fits, write a custom rule: a `go/analysis` analyzer in `x/go/analyzers`
  added to every `.custom-gcl.yaml`, or a local ESLint rule in `configs/eslint/rules`.
  Give it specs that cover each case it reports and each case it must not report.
- The rule PR adds the rule but leaves it off (Go), or at `warn` (ESLint), while
  violations exist.

## Step 4: fix existing violations

Fix every existing violation in mechanical `review/bot` PRs, one per package, each about
200 changed lines. Each fix must keep the meaning the convention wants, not only silence
the rule. When a fix shows a real bug, leave that site and put the bug in the report.
Never add a baseline, an ignore list, or a suppression comment for an existing
violation.

When a rule would need more than about 500 fixes, ship it report-only, with a fix plan
in its PR description, and stop there.

## Step 5: turn the rule on

When the fix PRs have merged and no violation remains, open the last PR, which turns the
rule on as blocking. Update the prose rule to name the linter that now enforces it.

## PR rules

- One idea per PR. Branch from `main`. Stack only on an open PR the fix needs (root
  `AGENTS.md`, Rule 3). Mechanical fixes ship alone. There is no limit on the number of
  PRs.
- Branch: `agent/lint-<rule>-<short-slug>`.
- Title: `SY-####: Sentence case description` when a Linear issue covers the work, else
  `[<package>] Sentence case description`. Put code in backticks.
- Body: fill `.github/pull_request_template.md`. Write `None` for the Linear issue when
  there is none.
- Label: `review/bot`.
- Never add a co-author line or a "Generated with" footer. Never bump versions or edit
  generated code.
- Before you push, run the format check, the lint, and the tests of each package you
  touched. Push only when they pass. Never run two Go builds at the same time; run
  Ginkgo with `--compilers=1`. Check TypeScript with `check-types` and `lint`, never
  `npx tsc`.

## Report

Your last message is the report, posted to Slack. Use plain text, at most 20 lines:

- Rules added, turned on, or in flight, each with its PR links and violation count.
- The ranked candidates you did not take, each with a one-line reason.
- Real bugs that a fix found.
