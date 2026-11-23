# Bug hunt

You are on a fresh clone of `main` of the Synnax monorepo. Read all the code in your
area and fix the real bugs you find. Nobody watches this session, so do not ask
questions.

1. Read `CLAUDE.md`, `CONTRIBUTING.md`, `docs/claude/testing.md`, and the guides they
   name for your area. Follow them.
2. Read every source file in your area. Skip generated code and vendored code. Read the
   tests and the callers outside the area when you must know how the code is used.

## What counts

A real bug is code that does the wrong thing on an input or an order of events that can
occur: a wrong result, a crash, lost or corrupt data, a race, a leak, or a wrong or
swallowed error. You must have evidence: a spec that fails on `main`. A security issue
follows the next section instead.

Never report style, naming, comments, structure, or speed with no wrong result. "Could
be cleaner" is not a bug.

## Security issues

This repository is public. Never write about a possible security issue in a public
place: no branch, commit, PR, comment, or issue text. This covers injection, cross-site
scripting, a way past authentication or access control, unsafe deserialization, path
traversal, an exposed secret, weak or missing TLS verification, and any other fault an
attacker can use. When you suspect one, stop work on it and push nothing for it. In your
report, write only `Possible security issue in <area>, see the private security audit`,
with no details.

## For each bug

1. Look for an open PR or Linear issue on the same bug
   (`gh pr list --search "<words>"`). Skip a bug that already has one.
2. Branch from `main`: `agent/bug-hunt-<area>-<short-slug>`.
3. Write a spec that shows the bug, co-located as the language guide says. Run it and
   see it fail.
4. Look for the same bug in the sibling implementations: the TS, Python, and C++
   clients, `x` and Freighter (Go, TS, Python, and C++), the Go and C++ runtimes of Arc,
   and the Go code of the Core that the clients mirror. Fix each sibling that has the
   bug in the same PR, with a spec for each. A sibling spec that needs no Core must fail
   on `main` and pass with the fix. This job has no Core, so for a spec that needs one,
   let CI run it, and say in one sentence in the PR body why it fails on `main`. Open
   one PR per bug, not one per language.
5. Fix the cause in one place in each implementation. Run the specs, the package's
   tests, the format check, and the lint. All must pass, except the tests that need a
   Core, which CI runs.
6. Before you push, search the open PRs from `agent/bug-hunt-*` branches that change the
   same files. If one fixes the same bug, do not push or open a PR, and link that PR in
   your report. Else, push.
7. Open one PR per bug with `gh pr create --base main --title ... --body ...`:
   - Title: `[<package>] Fix ...`, or `SY-####: Fix ...` when a Linear issue covers it.
   - Body: fill `.github/pull_request_template.md`. Say what goes wrong, when, and why.
     Paste the spec's failure output from before the fix. List the siblings you checked,
     and say which had the bug.
   - Add one review tier label, as the "Review tiers" table in `CONTRIBUTING.md` says:
     `review/human` when the fix changes a user-facing interface, else `review/bot`. For
     `review/human`, also request a review from the reviewer the prompt names
     (`gh pr edit <n> --add-reviewer <login>`).
   - Never add a co-author line or a "Generated with" footer.

Keep each PR small, with one bug. When the fix needs a design decision or a large
change, open no PR; write a draft Linear issue in your report, to
`.claude/skills/create-linear-issue/SKILL.md`.

## When you find nothing

Open no PR. A run with no bug is a good result. Never open a PR only to have something
to show.

## Report

Your last message is the report. Keep it short: the PRs you opened, the suspected bugs
you could not prove and why, and the parts of the area you did not read.
