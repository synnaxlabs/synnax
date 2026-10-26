# Triage a failed nightly run

The nightly CI run of Synnax failed on `main`. Find why, decide if each failure is a
flake or a regression, and act on it. Nobody watches this session, so do not ask
questions.

1. Read `CLAUDE.md`, `docs/claude/testing.md`, and the guides they name for the code you
   touch. Follow them.
2. List the failed jobs and read their logs:
   `gh run view <run> --json jobs --jq '.jobs[] | select(.conclusion == "failure")'` and
   `gh run view <run> --log-failed`. For an integration job, use the
   `.claude/skills/synnax-integration-debug/SKILL.md` steps on the `test-results-*`
   artifacts.
3. For each failed test, decide:
   - **Regression**: it fails the same way on rerun, or a commit since the last green
     nightly changed the code under test. Find that commit with `git log`.
   - **Flake**: it passed on the last nightly runs and on `main` pull requests with no
     related change, and the failure comes from timing or order. Usual causes here are a
     timer that waits on the wall clock, the coarse ~16 ms Windows clock, a stream read
     before the stream is live, and fixtures that share state.
4. Search Linear (team `SY`) for an open issue on the same test or symptom.

Then act once per cause:

- **Small fix that you can prove**: fix the cause in one place, run the failed test
  locally, and commit with a subject under 72 characters. Name the Linear issue in the
  subject (`SY-####: Fix ...`) when one exists. The workflow opens one pull request from
  your commits, so put only one cause in it.
- **Matching Linear issue, no proven fix**: add a comment to the issue with the run
  link, the failed test, the log lines that matter, and your flake or regression
  verdict.
- **No issue, no proven fix**: push nothing. Put a draft issue in your report, written
  to `.claude/skills/create-linear-issue/SKILL.md`.

Never make a test pass by:

- deleting, skipping, or weakening a test, an assertion, or a check;
- adding a retry, a longer timeout, or a sleep;
- catching or ignoring an error.

A fix that changes how a user-facing surface behaves is not small. Report it instead.

Your last message is the report. Keep it short: each failed job, flake or regression,
the cause, and what you did (commit, Linear comment, or draft issue).
