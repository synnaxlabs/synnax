# Feature Pull Request Template

## Key Information

<!-- Edit the list below with the proper issue number and link -->

- **Linear Issue**: [SY-####](https://linear.app/synnax/issue/)

## Description

<!-- Please write a short (2-3 sentence) description describing the changes. -->

## Basic Readiness

- [ ] I have performed a self-review of my code.
- [ ] I have added relevant tests to cover the changes to CI.
- [ ] I have needed QA steps to the [release candidate](/synnaxlabs/synnax/blob/main/.github/PULL_REQUEST_TEMPLATE/rc.md) template that cover these changes.
- [ ] I have updated in-code documentation to reflect the changes.
- [ ] I have updated user-facing documentation to reflect the changes.

## Backwards Compatibility

### Data Structures

I have ensured that previous versions of stored data structures are properly migrated to new formats in the following projects:

- [ ] Server
- [ ] Console

### API Changes

The following projects have backwards-compatible APIs:

- [ ] Python Client
- [ ] Server
- [ ] TypeScript Client

### Breaking Changes

<!-- If anything in this section is not true, please list all breaking changes. -->
