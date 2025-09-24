# 20 - Engineering Process Standardization

- **Feature Name**: Engineering Process Standardization
- **Start Date**: 2024-06-13
- **Authors**: Emiliano Bonilla
- **Status**: Draft

# 0 - Summary

As the Synnax development team, codebase, and number of production deployments grow,
it's time to establish standard development and quality assurance workflows to guarantee
that released software is reliable and maintainable.

# 1 - Current Workflow

Our current development workflow is simple. For any change, an engineer:

1. Checks out a new branch from `main` with the linear issue number and short
   description.
2. Develops the feature, bug fix, or refactor.
3. Pull requests the branch back into `main` for review and merge, bumping any relevant
   package versions.
4. Receives feedback ad makes improvements until the pull request is approved.
5. Merges the branch into `main`, where CI automatically creates the relevant software
   releases.

The simplicity of this workflow is its strength, and was coined by GitHub as
[GitHub Flow](https://docs.github.com/en/get-started/using-github/github-flow). It's
easy to understand, and promotes rapid design and development cycles. It reduces
cognitive load by minimizing the number of branches that need to be tracked and merge
conflicts that need to be addressed.

This workflow has become the standard for SaaS product development. The challenge is
that **Synnax is not a SaaS product**. While SaaS products are deployed on company
controlled servers in tightly controlled dev environments, Synnax is deployed on
customer servers on all major operating systems.

We also release new versions several times a week, and have five different services that
are versioned independently. It's up to our users to ensure they are running the correct
versions of different services. Frequent releases and independent versioning makes it
difficult to manage and maintain Synnax.

**Deploying reliable software in user managed environments is more challenging than
deploying in a controlled cloud environment**. As a result, we need to be more
disciplined in our development and release processes.

## 2 - Proposed Workflow Summary

A new workflow should maintain adequate simplicity while adding quality control steps
that slow the pace of releases and increase the reliability of production software. I
propose a workflow that moves away from the modern GitHub Flow to a simplified version
of the more traditional
[Git Flow](https://www.atlassian.com/git/tutorials/comparing-workflows/gitflow-workflow).

### 2.0 - Branches

The updated workflow adds an intermediary `rc` branch that enables regulation of release
cycles and additional QA.

### 2.0.0 - Main

The `main` branch holds the current production release. It is protected from direct
commits, and can only be updated by an approved pull request from a release candidate
branch or a critical bug fix branch.

### 2.0.1 - RC

The `rc` branch holds the current release candidate. Instead of naming this branch `rc`,
I've chosen `rc` to indicate that the changes merged into this branch should be of
release quality.

When a new release is feature-ready, a pull request is opened from `rc` into `main`. At
this point, new features are no longer merged into the `rc` branch, and the candidate
undergoes QA.

#### 2.0.2 - Feature Branches

Feature branches are checked out from and pull requested back into the `rc` branch.
These branches maintain the same naming convention as the previous workflow i.e. they
are named according to the Linear issue they correspond to.

### 2.2 - Fix Branches

Fix branches can be checked out and pull requested into `rc` or into `main`. Critical
issues should be checked out of and pull requested into `main`. After a critical fix is
merged into `main`, it should be back-merged into `rc`. Like feature branches, fix
branches are named according to the Linear issue they correspond to.

# 3 - Candidate Builds

An issue with our current workflow is that we don't have a way to sustainably release
and install candidate builds. They are necessary for QA, and provide a good way for
developers of a certain service to have access to a candidate version of a different
service.

For example, when making changes to the most recent version of the Console, it would be
useful to have access to the latest server release candidate to test against.

### 3.0 - Server candidate builds

Server candidate builds are tagged and released with the tags `synnax-vX.Y.Z-rc`.
Additional installation instructions are added to the user facing documentation for easy
installation of candidate builds. The server release CI is automatically modified to
correctly tag and release builds merged to `rc` and `main`.

### 3.1 - Console candidate builds

Console candidate builds are tagged and released with the tags `console-vX.Y.Z-rc`.
Additional installation instructions are in the user facing documentation for easy
installation of candidate builds. Candidate builds have auto-update functionality
configured to receive new versions from the candidate builds. This is a good way to
easily get new versions and verify that the auto update functionality is working without
needing to wait until production.

## 4 - Versioning

### 4.0 - Prior to 1.0.0

For releases prior to 1.0.0, we follow a modified
[Semantic Versioning](https://semver.org/) scheme. We increment the minor version for
new features, and the patch version for bug fixes.

We have a number of different packages that are all versioned independently. This is
confusing for our users and our development team, as it's difficult to tell which
versions of which services are compatible with each other.

From now on, all public facing packages will maintain the same **minor** version number.
Internal packages are free to increment versions numbers as they see fit.

### 4.1 - After 1.0.0

To be determined.

## 5 - Release Cycles

A new Synnax release is published every two weeks, and is coordinated via Linear's
cycles. Both weeks of a cycle will be dedicated to feature development and low priority
bug fixes. On the first day of the subsequent cycle, QA begins on the release candidate,
and new features will no longer be merged. We'll test the candidate for a maximum of
three days, we'll release on the fourth day of the cycle.

## 6 - QA Process

The engineering team is in charge of QA, which consists of a checklist of manual
verifications and additional tests that need to be completed before a release candidate
can be merged into `main`. This checklist is available as a
[pull request template](/.github/PULL_REQUEST_TEMPLATE/rc.md).

## 7 - CI/CD Scheduling

We're currently running the majority of our CI/CD actions on push to any branch. This
adds a lot of noise to our CI/CD logs, and makes it difficult to determine relevant CI
passes/fails for pull requests and CD releases.

To remedy this, we'll only run CI/CD actions on pull requests, and run CD actions on
merges to `rc` and `main`. We'll open pull requests for release candidates at the time
of checkout, and run CI actions on all pushes to the branch.

Sometimes it is necessary to debug CI/CD workflows themselves. This is difficult do to
when we're not running CI actions on every push. To address this, we'll make it possible
to manually dispatch relevant workflows on any branch.

### 7.0 - Removal of Driver Build Workflow

The driver build workflow currently executes the same process as run in the server build
workflow. I propose we remove the driver build workflow, and instead rely only on the
server build workflow to build and release the driver.
