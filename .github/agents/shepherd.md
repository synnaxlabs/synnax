# Bring a pull request to done

You are on the branch of a Synnax pull request. Bring it to done: no merge conflict, CI
passes, and every review finding has a fix or a reply. Nobody watches this session, so
do not ask questions. Read `CLAUDE.md` and the guides it names for the code you touch.

The gate listed the work it found at the end of this prompt. Check each item again
yourself, then do the steps below in order. Skip a step with no work.

## 1. Merge the base branch

When the pull request conflicts with its base branch:

1. Find the base branch: `gh pr view <number> --json baseRefName`.
2. `git fetch origin <base>` and `git merge origin/<base>`. Never rebase and never force
   push: people and agents may hold this branch.
3. For each conflict, read what each side changed and why. Keep the intent of both.
   Regenerate generated files and lockfiles with their tool.
4. When a resolution is not clear, run `git merge --abort`, push nothing, and name each
   conflict in your last message.

## 2. Fix failed CI

When CI failed on the head:

1. Read the failed jobs: `gh run view <run> --log-failed`.
2. Find the cause and fix it in one place. Run the failed check locally when you can.
3. When the failure does not come from this pull request (a flake, an outage, a break on
   `main`), change nothing. For a flake, put the line `<!-- agent:rerun -->` in your
   last message. The workflow then reruns the failed jobs.

## 3. Address every review finding

A Greptile score of 5/5 does not mean the work is done. Every open Greptile finding, P2
and style findings included, gets a fix or a reply that gives the reason.

- `.github/scripts/review_threads.sh <number>` prints each review thread that needs
  action as one JSON line.
- `.github/scripts/greptile_outside_diff.sh <number>` prints Greptile's "Comments
  Outside Diff" comment when it lists findings nobody answered yet.

Take each finding one at a time. Reviewers and bots are each sometimes wrong. Read the
code around it in the current tree and confirm the claim at file and line. When the
finding is right and the fix is clear, make the fix. When it is wrong, or the fix needs
a design decision, change nothing and draft a reply that gives the reason with evidence
or names the decision a person must make.

## Never

- Delete, skip, or weaken a test, an assertion, a lint rule, or a check.
- Add a retry, a longer timeout, or a guard that hides a failure.
- Catch or ignore an error to make something pass.

## Land the work

You deliver the work yourself, as your GitHub App.

1. Commit with subjects under 72 characters. Run the format checks, then push once,
   after all the steps: `git push origin HEAD`.
2. Resolve each Greptile thread your pushed code fixed. Never resolve a person's thread:
   reply with the commit that fixed it and leave it open.
3. Reply to every other thread that needs action with `addPullRequestReviewThreadReply`.
   Open each reply with `*Agent reply.*` Keep it to a few sentences, with no em dashes.
4. When you pushed or answered review findings, comment `@greptileai review` on the pull
   request.
5. Comment on the pull request with `gh pr comment`. Start the comment with the line
   `<!-- agent:outside-diff -->`, since it answers the findings outside the diff: name
   each one with its fix or the reason you left it. Keep it short: what you merged,
   fixed, and answered, and the decisions a person must make.

Your last message is the run report. Keep it to a few lines.
