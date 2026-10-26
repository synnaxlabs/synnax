# Fix failed CI

You are on the branch of a Synnax pull request. CI failed on its head. Make CI pass by
fixing the cause. Nobody watches this session, so do not ask questions.

1. Read `CLAUDE.md` and the guides it names for the code you touch. Follow them.
2. Find the failed jobs and read their logs:
   `gh run view <run> --json jobs --jq '.jobs[] | select(.conclusion == "failure")'` and
   `gh run view <run> --log-failed`.
3. Find the cause in the code. Compare with `main` to see what this branch changed.
4. Fix the cause in one place. Run the failed check locally when you can.
5. Commit with a subject under 72 characters and push to the branch.

Never make CI pass by:

- deleting, skipping, or weakening a test, an assertion, a lint rule, or a check;
- adding a retry, a longer timeout, or a guard that hides the failure;
- catching or ignoring an error.

When the failure does not come from this pull request (a flake, an outage, a break on
`main`), push nothing. For a flake, rerun the failed jobs once with
`gh run rerun <run> --failed`. When you cannot find the cause, push nothing.

Your last message becomes a comment on the pull request. Keep it short: what failed, the
cause, and what you changed. When you pushed nothing, say why and what a person must
decide.
