# Nightly improvement

You are on a fresh clone of `main` of the Synnax monorepo. Your theme follows these
rules. Nobody watches this session, so do not ask questions.

## Rules for every theme

1. Read `AGENTS.md`, `CONTRIBUTING.md`, and the guides they name for the code you touch.
   Follow them.
2. Fix only what you can prove. A finding you are not sure of goes in the report, not in
   a PR.
3. Each PR holds one idea, branches from `main`, and merges alone. Aim for under 200
   changed lines. Never stack PRs. There is no limit on the number of PRs.
4. Keep each change mechanical. Change behavior only to remove the fault the theme
   names, such as a race. Never change stored shapes or wire formats, never bump
   versions, and never hand-edit generated code.
5. Before you open a PR, look for an open PR or Linear issue on the same finding
   (`gh pr list --search "<words>"`). Skip a finding that already has one.
6. Before you push, run the format check, the lint, and the tests of each package you
   touched. Push only when they pass.

## Opening a PR

- Branch: `agent/nightly-<theme>-<short-slug>`.
- Title: `[<package>] Sentence case description`, as in
  `[gorp] Remove the flaky create allocation test`. Use `SY-####: ...` when a Linear
  issue covers the finding.
- Body: fill `.github/pull_request_template.md`. Write `None` for the Linear issue when
  there is none. Say what changed and why in one short paragraph.
- Label: `review/bot`. Use `review/human` instead when the change touches a user-facing
  interface: CLI commands or flags, a client API, how the Console looks or behaves, or
  anything under `schemas/`.
- Never add a co-author line or a "Generated with" footer.
- Open it with `gh pr create --base main --title ... --body ... --label ...`.

## When you find nothing

Open no PR. A night with no PR is a good result when nothing is worth a fix. Never make
a change only to have something to show.

## Report

Your last message is the report. Keep it short: the PRs you opened, the findings you
skipped and why, and what you checked when you found nothing.
