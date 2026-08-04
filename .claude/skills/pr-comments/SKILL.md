---
name: pr-comments
description:
  Workflow for working through unresolved review comments on a pull request you
  authored. Use when asked to address, triage, or work through PR review comments,
  reviewer feedback, or bot comments. Finds unresolved threads whose last word belongs
  to the reviewer, researches each one, and brings a recommendation per comment. The
  user chooses implement, skip, or respond. GitHub writes batch to the end.
---

# PR Comments

Working through review comments on the user's own PR, one at a time. Every comment is
researched before it gets a verdict, because reviewers and bots are each sometimes
wrong. The user decides implement, skip, or respond on every one. Nothing is written to
GitHub until the work is committed and pushed.

Reviewing someone else's diff is `code-review`'s job. This skill handles only incoming
comments on a PR the user authored.

## Step 1: Find the comments that need action

Resolution state exists only in GraphQL. `gh pr view --comments` cannot answer this.

```bash
gh pr view --json number,author # the PR for the current branch

gh api graphql -f query='
query($o:String!,$r:String!,$p:Int!){
  repository(owner:$o,name:$r){
    pullRequest(number:$p){
      author{login}
      reviewThreads(first:100){nodes{
        id isResolved isOutdated path line
        comments(first:100){nodes{author{login} body url}}
      }}
    }
  }
}' -f o=synnaxlabs -f r=synnax -F p="$PR"
```

A thread needs action when it is unresolved and the last comment is not the PR author's.
That catches a reviewer comment nobody has answered yet, and skips threads where the
author already had the last word.

```bash
jq '.data.repository.pullRequest as $pr | $pr.reviewThreads.nodes[]
    | select(.isResolved == false
             and (.comments.nodes[-1].author.login != $pr.author.login))'
```

Bots and humans arrive through the same query and get the same treatment. An
`isOutdated` thread points at code that has since changed, so say so when presenting it;
the comment may already be moot.

## Step 2: Surface what cannot be looped

Top-level PR comments and review summary bodies carry no resolution state, so they can
never enter the loop. List them once, up front, as context. When one of them carries a
significant body of work rather than a passing remark, ask the user whether they want it
worked through as well.

## Step 3: The loop, one comment at a time

Simplest first. Never batch several comments into one proposal, and never act on all of
them at once. For each comment:

1. **Research it.** What is the comment actually asking, and what code changes if it is
   implemented? Read the surrounding file in the current tree, not just the hunk the
   comment sits on, and follow whatever it points at.
2. **Verify the claim.** Reviewers and bots are each sometimes right and sometimes
   wrong. Confirm the behavior at file:line before agreeing or rebutting. A rebuttal
   cites evidence, never vibes.
3. **Recommend, verdict first.** One of "real, the fix is X", "intentional, because Y",
   or "needs discussion". A few sentences, naming the affected code.
4. **Wait for the user's call.** Implement, skip, or respond.

The three outcomes:

- **Implement** -> make the edit now. It lands locally; nothing goes to GitHub yet.
- **Skip** -> record the decision and move on. Nothing goes to GitHub.
- **Respond** -> draft the reply, show it, and hold it until the end.

Contested design threads escalate to a synchronous conversation rather than a long
comment exchange.

## Step 4: Land the work, then write to GitHub

All GitHub writes batch to the end, after the fixes are committed and pushed. Resolving
a thread whose fix exists only in the working tree shows the reviewer a resolved thread
with no code behind it.

1. Commit and push the accumulated edits.
2. Resolve the threads that code edits fixed:

```bash
gh api graphql -f query='mutation($id:ID!){
  resolveReviewThread(input:{threadId:$id}){thread{isResolved}}
}' -f id="$THREAD_ID"
```

3. Post the replies the user approved:

```bash
gh api graphql -f query='mutation($id:ID!,$b:String!){
  addPullRequestReviewThreadReply(input:{pullRequestReviewThreadId:$id,body:$b}){
    comment{url}
  }
}' -f id="$THREAD_ID" -f b="$BODY"
```

**Resolve only what code fixed.** A thread answered with an explanation and no code
change stays open. The reviewer decides whether the answer satisfies them.

## Replies

- **Only on an explicit instruction.** A reply is never posted because it looked like
  the natural next move. The user says respond, and approves the text first.
- **Disclose the collaboration.** Every posted reply opens with a short italic line
  saying it was drafted by Claude together with the dev, using the PR author's first
  name: `*<author> + Claude: drafted in our review session.*`
- **Tight, focused, natural.** A few sentences in the user's voice. No preamble, no
  restating the reviewer's comment back at them, no em dashes, no customer names.

## Quick reference

| Situation                        | Gate                                                     |
| -------------------------------- | -------------------------------------------------------- |
| Finding the comments             | GraphQL `reviewThreads`; `gh pr view` cannot do it       |
| Deciding a thread needs action   | Unresolved and the last comment is not the author's      |
| Top-level comment or review body | Surfaced as context; looped only if the user asks        |
| Working the list                 | One comment at a time, simplest first, never batched     |
| Before recommending anything     | Claim verified at file:line in the current tree          |
| Choosing what happens            | The user's call: implement, skip, or respond             |
| Writing anything to GitHub       | Only after the fixes are committed and pushed            |
| Resolving a thread               | Only when code edits fixed it; replied threads stay open |
| Posting a reply                  | Explicit instruction, approved text, Claude disclosed    |
