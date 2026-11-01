# Contributing to Synnax

How a change gets into `main`. Setup lives in the package guides
([Core](core/CONTRIBUTING.md), [Console](console/CONTRIBUTING.md),
[Pluto](pluto/CONTRIBUTING.md), [TypeScript client](client/ts/CONTRIBUTING.md),
[Python client](client/py/CONTRIBUTING.md), [docs site](site/docs/CONTRIBUTING.md)),
design in the [RFCs](docs/tech/rfc), and code style in [AGENTS.md](AGENTS.md).

## Commit hooks

`pnpm install` points Git at the hooks in `.githooks`. On each commit, the `pre-commit`
hook runs `scripts/fix.sh --staged`, which adds copyright headers to the staged files,
formats them, and stages the fixes. Skip it with `git commit --no-verify`. In a checkout
made before the hooks, run `pnpm prepare` once. Hooks in `.git/hooks` no longer run;
`.githooks` holds the Git LFS hooks.

## Branches and pull requests

`main` is the only long-lived branch and is always releasable. A branch is named after
its Linear issue (`sy-4892-review-tiers`) and returns to `main` by pull request. A
hotfix lands on `main` first, then a cherry-pick PR targets `release/<product>-X.Y`. A
person dispatches every release ([RFC 0058](docs/tech/rfc/0058-release-workflow.md)).

- Title: `SY-####: Sentence case description`, or a prefix such as `[docs]` with no
  issue.
- Description: what changed and why, leading with the effect on a user or the
  architecture. Never a file-by-file walk.
- One review tier label. Answer every Greptile comment with a fix or a reason. A
  `review/bot` PR that is not a draft merges by itself once CI and the gate pass. The
  author merges a `review/human` PR once the gate passes.
- The `preview` label deploys a live Core with the Console for the PR, and each push
  while the label stays deploys it again. Add it when the PR changes the Console UI or
  needs a live Core to review. `preview/docs` does the same for the docs site.
- A version bump rides the PR that ships the change. Test-only and docs-only changes
  bump nothing.

## Review tiers

A tier says if a human must read the PR. A reviewer can raise it by swapping the label.

| Label          | Review                          | Covers                                                                                                                                                                                                                                               |
| -------------- | ------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `review/human` | A human reads it, plus Greptile | A change to a user-facing interface: a new field or data structure (anything under `schemas/`, a stored shape, a migration), a Python or TypeScript client interface, a Core or Driver config (flags, config files), or a Console design or feature. |
| `review/bot`   | Greptile only                   | Everything else: bug fixes, internals, tests, lints, renames, formatting, and generated code.                                                                                                                                                        |

A `review/human` reviewer reads the changed interface and its tests, approves or
requests changes, and confirms a stored shape change carries a migration.

Two statuses gate a merge into `main` or a `release/**` branch. `Review gate` stays
pending until the PR has one tier label and a finished Greptile review of the head. A
Greptile confidence score below 5/5 fails it: push a fix, or retrigger the review after
answering the comments. The gate does not wait for a human approval on either tier.
Greptile skips Dependabot, so a pull request Dependabot opened and wrote every commit of
passes the gate without a review. `OK` is the CI workflow's last job: it fails when any
check the PR's paths select failed. A merge into `main` goes through a merge queue,
which reruns CI with the integration suite added before the branch moves. A queue
entry's CI builds `main` with the entries ahead of it and the entry itself, but selects
checks from the entry's own changes. Each entry merges only when its own CI passes: an
entry that fails leaves the queue, and GitHub rebuilds the entries behind it. GitHub
refuses a queue on a wildcard branch pattern, so a `release/**` merge goes in directly,
and CI runs again on the pushed commit. The queue also runs the gate again. A release
runs the integration suite itself. Admins bypass when Greptile is down. The gate script
is `.github/scripts/check_review.sh`, the rulesets `.github/rulesets/main.json` and
`.github/rulesets/release.json`. An admin applies a changed ruleset file by hand; the
`Check - Rulesets` workflow fails each day until GitHub matches the files.

A pull request into `main` also cannot merge while CodeQL finds a new alert in its
changes with a security severity of medium or higher, or a severity of error. Alerts
already on `main` do not block. The rule waits for the CodeQL scan of each language, so
every pull request scans all of them. GitHub does not apply the rule to a merge queue
group.

## Size

A reviewer finishes a PR in one sitting; a couple hundred lines is the target. The
failure to avoid is a branch that grows for days and lands as one PR nobody can review.

- Cut early. When the diff passes a few hundred lines or picks up a second idea, open
  the PR and start the next piece as a separate branch off `main`.
- Prefer separate branches off `main` over a stack. A stack is fine when a piece truly
  cannot land alone, but most pieces can.
- Unfinished work ships dark behind a flag as small PRs, not on a branch.
- Mechanical changes (renames, formatting, lint, regenerated code) ship alone as
  `review/bot` PRs. A fix and the refactor it needed are two PRs; the refactor first.

## Dark launches and Synnax Desktop

Synnax Desktop is the free edition, where a feature ships first. The Console is the
enterprise edition and gets it once it has proved itself. A static flag in
`console/src/flags.ts` (or `site/docs/src/flags.ts`) is on in dev and Desktop builds and
off in the Console build; its entry names the owner, the Linear umbrella issue, and the
release that removes it. A flag past that release is a bug: promote or delete.

1. **Land dark** in `review/human` PRs. The reviewer confirms the Console build
   tree-shakes the code out.
2. **Ship in Desktop.** Field use and bug reports are the promotion evidence.
3. **Promote** after at least one stable Desktop release, in a `review/human` PR that
   only flips or deletes the flag and removes the dark branches. The reviewer reads the
   feature as it stands on `main`, from the umbrella issue's PR list, and holds it to
   the enterprise bar: the `AGENTS.md` principles, tests, docs, no dev-only path left. A
   feature that fails stays in Desktop until a later promotion passes.

## Issue priority

Priority means when we act.

| Priority | Meaning                                                               |
| -------- | --------------------------------------------------------------------- |
| Urgent   | Within hours: a blocked customer or release, broken production or CI. |
| High     | This cycle or next.                                                   |
| Medium   | This quarter. The floor for customer requests.                        |
| Low      | Someday. Never cancel for age.                                        |
