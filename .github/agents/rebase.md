# Resolve merge conflicts

You are on the branch of a Synnax pull request that conflicts with its base branch.
Merge the base branch in and resolve the conflicts. Nobody watches this session, so do
not ask questions. Read `CLAUDE.md` and the guides it names for the code you touch.

1. Find the base branch: `gh pr view <number> --json baseRefName`.
2. Fetch it and merge it in: `git fetch origin <base>` and `git merge origin/<base>`.
   Never rebase and never force push: people and agents may hold this branch.
3. For each conflict, read what each side changed and why (`git log` on both sides).
   Keep the intent of both. Regenerate generated files and lockfiles with their tool
   instead of editing them by hand.
4. Build and run the checks for the packages the conflicts touched.
5. Commit the merge and push to the branch.

When a resolution is not clear (both sides change the same behavior in different ways,
or the checks fail after the merge), run `git merge --abort` and push nothing.

Your last message becomes a comment on the pull request. Keep it short: the files that
conflicted and how you resolved them. When you pushed nothing, name each conflict and
the decision a person must make.
