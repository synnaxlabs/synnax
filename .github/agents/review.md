# Address review findings

You are on the branch of a Synnax pull request. Reviewers left findings on it. Work
through every finding that needs action. Nobody watches this session, so do not ask
questions. Read `CLAUDE.md` and the guides it names for the code you touch.

A Greptile score of 5/5 does not mean the work is done. Every open Greptile finding, P2
and style findings included, gets a fix or a reply that gives the reason.

## Find the findings

- Run `.github/scripts/review_threads.sh <number>`. It prints each review thread that
  needs action as one JSON line: unresolved, with the last word from Greptile or a
  person other than the author. An outdated thread may already be moot.
- Run `.github/scripts/greptile_outside_diff.sh <number>`. It prints Greptile's
  "Comments Outside Diff" comment when it lists findings nobody answered yet. These
  findings have no thread.

## Decide each finding

Simplest first, one at a time. Reviewers and bots are each sometimes wrong.

1. Read the code around the finding in the current tree, not only the hunk.
2. Confirm the claim at file and line before you agree or disagree.
3. When the finding is right and the fix is clear, make the fix.
4. When the finding is wrong, or the fix needs a design decision, change nothing. Draft
   a reply that gives the reason with evidence, or names the decision a person must
   make.

Never satisfy a finding by deleting or skipping a test or by adding a guard that hides a
failure.

## Land the work

1. When you changed code, commit with a subject under 72 characters and push to the
   branch before you write anything to GitHub. A round with only replies needs no
   commit.
2. Resolve a thread only when your pushed code fixed it and its author is Greptile.
   Never resolve a person's thread: reply with the commit that fixed it and leave it
   open.
3. Reply to every other thread with `addPullRequestReviewThreadReply`. Open every reply
   with `*Agent reply.*` Keep it to a few sentences, with no em dashes.
4. When you pushed, comment `@greptileai review` on the pull request.

Your last message becomes a comment on the pull request. It also answers the findings
outside the diff, so name each one with its fix or the reason you left it, and start the
message with the line `<!-- agent:outside-diff -->`. Keep it short: the findings you
fixed, the findings you answered, and the decisions a person must make.
