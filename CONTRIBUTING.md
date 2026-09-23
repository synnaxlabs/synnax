# Contributing to Synnax

This guide covers how a change gets into `main`: branches, pull requests, review tiers,
pull request size, and how unfinished work ships dark behind a flag. Environment setup
lives in the package guides: [Core](core/CONTRIBUTING.md),
[Console](console/CONTRIBUTING.md), [Pluto](pluto/CONTRIBUTING.md),
[TypeScript client](client/ts/CONTRIBUTING.md),
[Python client](client/py/CONTRIBUTING.md), and [docs site](docs/site/CONTRIBUTING.md).
Design decisions live in the [RFCs](docs/tech/rfc) and the
[technical documentation](docs/tech). Code style rules live in [CLAUDE.md](CLAUDE.md)
and apply to every author, human or not.

## Branches

`main` is the only long-lived branch and is always releasable. A change branches from
`main`, is named after its Linear issue (`sy-4892-review-tiers`), and returns to `main`
by pull request, and never targets another unmerged branch. A hotfix lands on `main`
first and is then cherry-picked by pull request onto `release/<product>-X.Y`. A person
dispatches every release; nothing publishes on push. See
[RFC 0058](docs/tech/rfc/0058-release-workflow.md).

## Pull requests

- The title is `SY-####: Sentence case description`. Work with no issue uses a prefix
  such as `[docs]`.
- The description says what changed and why, and leads with the effect on a user or the
  architecture. It never walks the diff file by file.
- The author adds one review tier label (below). The gate refuses a pull request with
  none or two.
- The author merges once the gate passes. A pull request with a human reviewer waits for
  that reviewer's approval; a `review/bot` pull request merges once Greptile has
  reviewed.
- Every Greptile comment gets an answer before merge: a fix, or a reply that says why
  not.
- A version bump belongs in the pull request that ships the change; a test-only or
  docs-only change bumps nothing.

## Review tiers

A tier measures what a pull request can change in a shipped enterprise product, not how
large the diff is. Code that cannot reach an enterprise user, because a flag keeps it
dark, needs less review than code that can. The author picks the tier; any reviewer can
raise it by swapping the label, and the gate never lowers it.

| Tier | Label             | Review                   | Covers                                                                                                                                                                                                                                                                           |
| ---- | ----------------- | ------------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1    | `review/thorough` | One human, plus Greptile | A flag turning on for the Console or being deleted (a feature promotion). A change to the telemetry pipeline (Cesium, the framer, the Driver pipeline) or a metadata system (Aspen, Gorp, the ontology, Oracle, the schemas' semantics). A refactor that moves a layer boundary. |
| 2    | `review/light`    | One human, plus Greptile | A user-facing surface changes: CLI commands or flags, a client API, how the Console looks or behaves. A stored shape changes: anything under `schemas/`, a migration. A feature lands dark behind a flag. Code that ships only in Synnax Desktop.                                |
| 3    | `review/bot`      | Greptile only            | A simple bug fix. Tests, lints, renames, formatting, generated code, and other mechanical refactors.                                                                                                                                                                             |

**Tier 1 review**: The reviewer pulls the branch, runs the feature, and reads every
file. The reviewer checks the architectural principles in `CLAUDE.md`, the tests, and
the docs, and reads the code the diff touches, not only the diff.

**Tier 2 review**: The reviewer reads the changed surface and its tests, and confirms
that a stored shape change carries a migration and that a dark feature stays dark.

**Tier 3 review**: Nobody reads the diff but Greptile. The author answers its comments
and merges. When a fix turns out to touch behavior the author did not expect, the author
raises the tier.

### The gate

The `Review gate` check runs on every pull request event and review, and the `main`
ruleset requires it together with `Greptile Review`. It fails when:

- The pull request carries no tier label, or more than one.
- The tier is below the floor its paths set. A non-test file under `schemas/`,
  `oracle/`, `cesium/`, `aspen/`, `x/go/gorp/`, `x/go/kv/`, `core/pkg/storage/`,
  `core/pkg/distribution/`, `core/pkg/service/framer/`, `core/pkg/service/channel/`,
  `core/pkg/service/ontology/`, or `driver/pipeline/` can never be `review/bot`.
- Tier 1 or 2 has no approval from a human other than the author. A later request for
  changes or a dismissal retires an approval.

Repository admins can bypass the ruleset when Greptile is down. The paths live in
`.github/scripts/check_review.sh`; the ruleset lives in `.github/rulesets/main.json`.

## Pull request size

There is no size limit, but a reviewer must finish a pull request in one sitting, and a
couple hundred lines is the target. The failure to avoid is a branch that grows for days
and lands as one pull request nobody can review. So:

- **Cut early.** When the diff passes a few hundred lines, or a second idea appears in
  it, open the pull request and continue from `main` once it merges.
- **Every pull request merges on its own.** None depends on another unmerged branch. If
  a piece only makes sense after another lands, land the other first. A chain of pull
  requests based on each other is the failure mode with extra steps.
- **Unfinished work ships dark.** A feature that is not ready merges behind a flag as
  small Tier 2 pull requests. The promotion pull request is the one place the whole
  feature is read at once.
- **Mechanical changes go alone.** A rename, a format run, a lint fix, or regenerated
  code ships as its own `review/bot` pull request, so the human review reads only the
  hand-written change.
- **A fix and the refactor it needed are two pull requests.** The refactor lands first.

## Dark launches and Synnax Desktop

Synnax Desktop is the free edition and the place where a feature ships first. The
Console is the enterprise edition and gets a feature once it has proved itself in
Desktop. Both are one codebase; a static flag decides what each build wires in.

A flag is a build-time boolean in `console/src/flags.ts` (and `docs/site/src/flags.ts`
for a docs page). A flag is on in dev builds and in the Desktop build, and off in the
Console build. Its registry entry names the owner, the Linear umbrella issue, and the
release that removes it. A flag that outlives its removal release is a bug: promote the
feature or delete it.

The lane, in order:

1. **Land dark.** Each piece of the feature is a Tier 2 pull request. The reviewer
   confirms that the Console build tree-shakes the code out, and reads the rest lightly.
2. **Ship in Desktop.** The feature rides the next Desktop release with the flag on.
   Field use, bug reports, and analytics are the evidence the promotion review needs.
3. **Promote.** After the feature has shipped in at least one stable Desktop release,
   the owner opens the promotion pull request at Tier 1. It contains only the flag flip
   or the flag's deletion and the removal of the dark branches, so the diff is small.
   The reviewer reads the feature as it stands on `main`, using the umbrella issue's
   pull request list, and holds it to the enterprise bar: the `CLAUDE.md` principles,
   tests that cover the behavior, docs on the site, and no dev-only path left behind.

A feature that fails promotion stays in Desktop under its flag, with the findings on the
umbrella issue, until a later promotion pull request passes.

## Issue priority

| Priority | Feature                                        | Bug                                                                                    |
| -------- | ---------------------------------------------- | -------------------------------------------------------------------------------------- |
| Urgent   | Blocks an upcoming pilot.                      | Stops the user and costs significant time or resources; a core element fails outright. |
| High     | Requested by users and important to their use. | Degrades the experience enough to outrank most feature work; slows or repeats work.    |
| Medium   | Some interest, or a clear improvement.         | Noticeable, targeted for the next release, or has a workaround today.                  |
| Low      | Not critical to users or the product.          | Low impact; can wait for a later release.                                              |
