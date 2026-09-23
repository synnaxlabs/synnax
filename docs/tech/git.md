# The Synnax Git workflow in three instructions

1. Branch names are tied to Linear issues. Prefix your branch with the issue it belongs
   to, then a short description: `sy-175-line-plot-rules`.
2. Branches check out from `main` and merge back into `main`. A stacked branch targets
   its parent; a hotfix is cherry-picked onto a `release/<product>-X.Y` branch after it
   lands on `main`.
3. Open a pull request, add a review tier label, and merge when the gate passes. The
   tiers are in the [contributing guide](../../CONTRIBUTING.md).
