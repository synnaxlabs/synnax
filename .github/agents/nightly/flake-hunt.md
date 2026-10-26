# Theme: flake hunt

Find tests that failed and then passed on a retry, and fix the cause.

1. List CI and nightly runs of the last 14 days that passed on a later attempt:
   `gh run list --workflow ci.yaml --limit 500 --json databaseId,attempt,conclusion` and
   the same for `nightly.yaml`. Keep the runs with `attempt > 1` and
   `conclusion == "success"`.
2. For each, read the failed jobs of attempt 1:
   `gh api repos/{owner}/{repo}/actions/runs/<id>/attempts/1/jobs` and
   `gh run view <id> --attempt 1 --log-failed`. Find the failed test names.
3. Also look at `merge_group` runs that failed and then passed for the same pull request
   with no new commit.
4. Rank the tests by failure count. A test that failed in two or more runs with no
   related change is a flake. Work from the top.

For each flake, find the race or timing fault and fix it. Usual causes in this repo:

- a timer or deadline that waits on the wall clock instead of an injected clock;
- the coarse ~16 ms Windows clock;
- a read before a stream or observer is live, or an async observer checked with no
  `Eventually`;
- fixtures that share state or fixed ports across tests.

Never fix a flake with a retry, a longer timeout, a sleep, a skip, or a weaker
assertion. When you cannot prove the cause, do not open a PR; put the test, the runs,
and the log lines in the report. One PR per flake.
