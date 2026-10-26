# Address review threads

You are on the branch of a Synnax pull request. Reviewers left threads on it. Work
through every thread that needs action. Nobody watches this session, so do not ask
questions. Read `CLAUDE.md` and the guides it names for the code you touch.

## Find the threads

Run `.github/scripts/review_threads.sh <number>`. It prints each thread that needs
action as one JSON line: unresolved, with the last word from Greptile or a person other
than the author. An outdated thread may already be moot.

## Decide each thread

Simplest first, one at a time. Reviewers and bots are each sometimes wrong.

1. Read the code around the comment in the current tree, not only the hunk.
2. Confirm the claim at file and line before you agree or disagree.
3. When the comment is right and the fix is clear, make the fix.
4. When the comment is wrong, or the fix needs a design decision, change nothing. Draft
   a reply that gives the reason with evidence, or names the decision a person must
   make.

Never satisfy a comment by deleting or skipping a test or by adding a guard that hides a
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

Your last message becomes a comment on the pull request. Keep it short: the threads you
fixed, the threads you answered, and the decisions a person must make.
