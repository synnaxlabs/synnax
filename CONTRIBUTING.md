# Contributing to Synnax

How a change gets into `main`. Setup lives in the package guides
([Core](core/CONTRIBUTING.md), [Console](console/CONTRIBUTING.md),
[Pluto](pluto/CONTRIBUTING.md), [TypeScript client](client/ts/CONTRIBUTING.md),
[Python client](client/py/CONTRIBUTING.md), [docs site](docs/site/CONTRIBUTING.md)),
design in the [RFCs](docs/tech/rfc), and code style in [CLAUDE.md](CLAUDE.md).

## Branches and pull requests

`main` is the only long-lived branch and is always releasable. A branch is named after
its Linear issue (`sy-4892-review-tiers`) and returns to `main` by pull request. A
hotfix lands on `main` first, then a cherry-pick PR targets `release/<product>-X.Y`. A
person dispatches every release ([RFC 0058](docs/tech/rfc/0058-release-workflow.md)).

- Title: `SY-####: Sentence case description`, or a prefix such as `[docs]` with no
  issue.
- Description: what changed and why, leading with the effect on a user or the
  architecture. Never a file-by-file walk.
- One review tier label. The author merges once the gate passes, after answering every
  Greptile comment with a fix or a reason.
- A version bump rides the PR that ships the change. Test-only and docs-only changes
  bump nothing.

## Review tiers

A tier measures what a PR can change in a shipped enterprise product, not how large it
is. The author picks; a reviewer can raise it by swapping the label; the gate never
lowers it.

| Tier | Label             | Review                   | Covers                                                                                                                                                                                                                                                                           |
| ---- | ----------------- | ------------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1    | `review/thorough` | One human, plus Greptile | A flag turning on for the Console or being deleted (a feature promotion). A change to the telemetry pipeline (Cesium, the framer, the Driver pipeline) or a metadata system (Aspen, Gorp, the ontology, Oracle, the schemas' semantics). A refactor that moves a layer boundary. |
| 2    | `review/light`    | One human, plus Greptile | A user-facing surface changes: CLI commands or flags, a client API, how the Console looks or behaves. A stored shape changes: anything under `schemas/`, a migration. A feature lands dark behind a flag. Code that ships only in Synnax Desktop.                                |
| 3    | `review/bot`      | Greptile only            | A simple bug fix. Tests, lints, renames, formatting, generated code, and other mechanical refactors.                                                                                                                                                                             |

**Tier 1**: The reviewer pulls the branch, runs the feature, and reads every file and
the code around it, against the `CLAUDE.md` principles, the tests, and the docs. **Tier
2**: The reviewer reads the changed surface and its tests, and confirms a stored shape
change carries a migration and a dark feature stays dark. **Tier 3**: Only Greptile
reads it. A fix that turns out to change behavior gets its tier raised.

Two statuses gate a merge into `main` or a `release/**` branch, both through a merge
queue. `Review gate` stays pending until the PR has one tier label, a successful
Greptile review and, for Tier 1 or 2, an approval from a human other than the author.
A later request for changes or a dismissal retires an approval. Two tier labels fail
it. `OK` is the CI workflow's last job: it fails when any check the PR's paths select
failed. The queue reruns CI on the merged result with the integration suite added
before the branch moves. Admins bypass when Greptile is down. The gate script is
`.github/scripts/check_review.sh`, the ruleset `.github/rulesets/main.json`.

## Size

A reviewer finishes a PR in one sitting; a couple hundred lines is the target. The
failure to avoid is a branch that grows for days and lands as one PR nobody can review.

- Cut early. When the diff passes a few hundred lines or picks up a second idea, open
  the PR and start the next piece as a separate branch off `main`.
- Prefer separate branches off `main` over a stack. A stack is fine when a piece truly
  cannot land alone, but most pieces can.
- Unfinished work ships dark behind a flag as small Tier 2 PRs, not on a branch.
- Mechanical changes (renames, formatting, lint, regenerated code) ship alone as
  `review/bot` PRs. A fix and the refactor it needed are two PRs; the refactor first.

## Dark launches and Synnax Desktop

Synnax Desktop is the free edition, where a feature ships first. The Console is the
enterprise edition and gets it once it has proved itself. A static flag in
`console/src/flags.ts` (or `docs/site/src/flags.ts`) is on in dev and Desktop builds and
off in the Console build; its entry names the owner, the Linear umbrella issue, and the
release that removes it. A flag past that release is a bug: promote or delete.

1. **Land dark** in Tier 2 PRs. The reviewer confirms the Console build tree-shakes the
   code out.
2. **Ship in Desktop.** Field use and bug reports are the promotion evidence.
3. **Promote** after at least one stable Desktop release, in a Tier 1 PR that only flips
   or deletes the flag and removes the dark branches. The reviewer reads the feature as
   it stands on `main`, from the umbrella issue's PR list, and holds it to the
   enterprise bar: the `CLAUDE.md` principles, tests, docs, no dev-only path left. A
   feature that fails stays in Desktop until a later promotion passes.

## Issue priority

| Priority | Feature                                        | Bug                                                                                    |
| -------- | ---------------------------------------------- | -------------------------------------------------------------------------------------- |
| Urgent   | Blocks an upcoming pilot or current customer.  | Stops the user and costs significant time or resources; a core element fails outright. |
| High     | Requested by users and important to their use. | Degrades the experience enough to outrank most feature work; slows or repeats work.    |
| Medium   | Some interest, or a clear improvement.         | Noticeable, targeted for the next release, or has a workaround today.                  |
| Low      | Not critical to users or the product.          | Low impact; can wait for a later release.                                              |
