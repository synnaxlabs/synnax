# Triage a failed nightly run

The nightly CI run of Synnax failed on `main`. Find why, decide if each failure is a
flake or a regression, and act on it. Nobody watches this session, so do not ask
questions.

1. Read `AGENTS.md`, `docs/claude/testing.md`, and the guides they name for the code you
   touch. Follow them.
2. List the failed jobs and read their logs. This session runs inside the failed run, so
   the run is still in progress and `gh run view --log-failed` gives nothing. Read each
   job's log instead:
   `gh run view <run> --json jobs --jq '.jobs[] | select(.conclusion == "failure") | .databaseId'`
   and `gh api repos/{owner}/{repo}/actions/jobs/<job id>/logs`. For an integration job,
   use the `.claude/skills/synnax-integration-debug/SKILL.md` steps on the
   `test-results-*` artifacts (`gh run download <run> --pattern 'test-results-*'`).
3. For each failed test, give one verdict:
   - **Regression**: you can link a commit since the last green nightly to the failure.
     Find it with `git log`, and show how its change causes the failed assertion.
   - **Flake**: the test passed on the last nightly runs and on `main` pull requests
     with no related change, and you can name the timing or order fault. Usual causes
     here are a timer that waits on the wall clock, the coarse ~16 ms Windows clock, a
     stream read before the stream is live, and fixtures that share state.
   - **Inconclusive**: the evidence does not prove either. Say what is missing.
4. Search Linear (team `SY`) for an open issue on the same test or symptom.

Then act once per cause:

- **Small fix that you can prove**: fix the cause in one place, run the failed test
  locally, and commit with a subject under 72 characters. Name the Linear issue in the
  subject (`SY-####: Fix ...`) when one exists. The workflow opens one pull request from
  your commits, so put only one cause in it.
- **Matching Linear issue, no proven fix**: add a comment to the issue with the run
  link, the failed test, the log lines that matter, your verdict, and its evidence.
- **No issue, no proven fix**: push nothing. Put a draft issue in your report, written
  to `.claude/skills/create-linear-issue/SKILL.md`.

Never make a test pass by:

- deleting, skipping, or weakening a test, an assertion, or a check;
- adding a retry, a longer timeout, or a sleep;
- catching or ignoring an error.

A fix that changes how a user-facing surface behaves is not small. Report it instead.

Your last message is the report. Keep it short: each failed job, its verdict, the cause,
and what you did (commit, Linear comment, or draft issue).
