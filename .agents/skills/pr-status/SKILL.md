---
name: pr-status
description:
  Report where the session's PRs stand as one short board. Use when the user invokes
  /pr-status, asks for the status of the PRs, asks what is blocking them, or asks if a
  stack is backmerged. Gives each PR's link, a one-sentence description, CI, Greptile
  and human review, and merge conflicts, in categories, with stack order and backmerge
  health.
---

# PR status

Report the live state of the PRs this session owns. This skill only reads. It never
merges, pushes, backmerges, comments, or requests a review.

## Step 1: Find the PRs

The list is every PR this session opened, pushed to, or was told to own. Get it from the
conversation. PR numbers in the arguments join the list. If the list is empty, say so
and stop. Never fall back to all of the user's open PRs.

## Step 2: Read the live state

Read every value from GitHub now. Never repeat a status from earlier in the session.

```bash
Q='query($n:Int!){repository(owner:"synnaxlabs",name:"synnax"){pullRequest(number:$n){
  number url title state isDraft mergeable headRefName headRefOid baseRefName body
  mergeCommit{oid}
  labels(first:20){nodes{name}}
  stack{number entries(first:50){nodes{position pullRequest{
    number state headRefName mergeCommit{oid}
  }}}}
  latestOpinionatedReviews(first:10){nodes{author{login} state}}
  reviewThreads(first:100){nodes{isResolved}}
}}}'
for n in $PRS; do
  gh api graphql -f query="$Q" -F n="$n" --jq '.data.repository.pullRequest
    | .headRefOid as $sha | {
    number, url, title, state, draft: .isDraft, mergeable,
    head: .headRefName, base: .baseRefName, mergeCommit: .mergeCommit.oid,
    tier: ([.labels.nodes[].name | select(startswith("review/"))] | first),
    greptile: ((.body // "") | [scan("Confidence Score: ([0-9])/5")] | first | first),
    greptileCurrent: ((.body // "") | contains("commit/" + $sha)),
    humans: [.latestOpinionatedReviews.nodes[]
      | select(.author.login | endswith("[bot]") | not)
      | {login: .author.login, state}],
    openThreads: ([.reviewThreads.nodes[] | select(.isResolved | not)] | length),
    stack: (.stack | if . then {number, entries: [.entries.nodes[]
      | {position, n: .pullRequest.number, state: .pullRequest.state,
         head: .pullRequest.headRefName,
         mergeCommit: .pullRequest.mergeCommit.oid}]} else null end)
  }'
  gh pr checks "$n" --json name,bucket,workflow --jq '
    [.[] | select(.workflow == "CI")] | {
      runs: length,
      ok: ([.[] | select(.name == "OK") | .bucket] | first),
      failed: [.[] | select((.bucket == "fail" or .bucket == "cancel")
                            and .name != "OK") | .name]
    }'
done
```

GitHub computes `mergeable` on request, so the first read can return `UNKNOWN`. Read the
PR again before you report it as unknown.

## Step 3: Work out each field

Start each status with ✅ when it is good, ❌ when it blocks the PR, or ⏳ when it is
waiting.

**CI.** `OK` is the last job of the CI workflow, and it fails if any selected check
fails. Ignore Vercel and other workflows.

| Result                                           | Status                      |
| ------------------------------------------------ | --------------------------- |
| `ok` is `pass`                                   | ✅ Passing                  |
| `failed` is not empty                            | ❌ Failing: `<check names>` |
| `runs` is above 0, nothing failed, `ok` not set  | ⏳ Running                  |
| `runs` is 0, or `gh pr checks` reports no checks | ❌ No run                   |

A PR that conflicts with its merge target gets no CI run, so check conflicts first when
you see `No run`.

**Review.** One cell holds Greptile, human review, and open threads.

- Greptile: `5/5` is good. Below 5 fails the `Review gate`. A clean re-review posts no
  review object, so the only source is the summary in the PR body. If `greptileCurrent`
  is false, the score is for an older commit: write `Greptile 5/5 on an older commit`.
  If `greptile` is null, write `no Greptile review`.
- Human: on `review/human`, name who approved or requested changes, or write
  `human pending`. On `review/bot`, omit the human part unless `humans` is not empty.
  With no tier label, write `no tier label`.
- Threads: add `N open threads` when `openThreads` is above 0.

Use ❌ for a score below 5, requested changes, or a missing tier label. Use ⏳ for
anything else that is not finished. An approval on an older commit stays valid.

**Conflicts.** ✅ None, ❌ Conflicts with `<target>`, or ⏳ Unknown. The target is the
PR's base, except in a native stack: there `mergeable` tests against the stack's base,
`main`. GitHub can cache a false `CONFLICTING` after a burst of pushes, so confirm it:

```bash
git fetch origin "$base" "$head"
git merge-tree --write-tree --name-only "origin/$base" "origin/$head"
```

An exit code of 1 confirms the conflict.

## Step 4: Check each stack

A PR is in a stack if `stack` is set (a native stack) or its base is not `main` or a
`release/` branch (a base chain). Add the other open PRs of the stack to the board, even
if this session does not own them. Order and backmerge health need all of them.

**Order.** For a native stack, sort `entries` by `position`, bottom first. Merged
entries stay in the list, so skip them. For a base chain, walk the bases:

```bash
gh pr list --state all --head "$base" --json number,state,baseRefName
gh pr list --state open --base "$head" --json number,headRefName
```

**Backmerge.** Each open branch must contain the branch below it. Compare each pair from
the bottom up:

```bash
gh api "repos/synnaxlabs/synnax/compare/$lower...$upper" --jq .behind_by
```

Zero means the upper branch contains the lower one. Any other number means that it needs
a backmerge. For the lowest open PR, if the PR below it merged, set `$lower` to that
PR's `mergeCommit`, because the branch can be gone. A base chain PR whose base PR merged
also needs a retarget to `main`.

Also get how far the lowest open PR is behind `main`. Report that count as information.
It is only a problem if the PR conflicts.

## Step 5: Report

Group the PRs by feature or area. Each stack is its own group, bottom PR first. Skip the
headings if one group would hold every PR.

Each group is one table. Write the description yourself from the PR, as one sentence
that starts with a verb, in 12 words or less. Do not copy the title. Mark a draft with
`draft` after the link.

```markdown
**Driver**

| PR                         | What                                           | CI         | Review                          | Conflicts |
| -------------------------- | ---------------------------------------------- | ---------- | ------------------------------- | --------- |
| [#4167](https://github...) | Refuses a second LabJack stream on one device. | ✅ Passing | ✅ Greptile 5/5                 | ✅ None   |
| [#4166](https://github...) | Exits the Driver with the Core on Windows.     | ⏳ Running | ⏳ Greptile 4/5, 2 open threads | ✅ None   |

**NI ports stack**, bottom to top

❌ #4124 is 1 commit behind #4122. #4122 is 54 commits behind `main`.

| PR  | What | CI  | Review | Conflicts |
| --- | ---- | --- | ------ | --------- |
```

The line above a stack's table starts with ✅ `Backmerged.` or lists each broken pair
with ❌. Put merged or closed PRs in one line at the end: `Merged: #4105, #4106.`

Stop after the board. No summary and no next steps.
