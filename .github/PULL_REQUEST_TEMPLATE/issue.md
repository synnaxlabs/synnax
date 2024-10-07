# Feature Pull Request Template

## Key Information

- **Linear Issue**: [SY-#](https://linear.app/synnax/issue/)

## Description

Please write a short (2-3 sentence) description describing the changes.

## Basic Readiness

- [ ] I have performed a self-review of my code.
- [ ] I have added relevant tests to cover the changes to CI.
- [ ] I have needed QA steps to the [release
      candidate](/.github/PULL_REQUEST_TEMPLATE/rc.md) template that cover these changes.
- [ ] I have updated in-code documentation to reflect the changes.
- [ ] I have updated user-facing documentation to reflect the changes.

## Backwards Compatibility

The following makes sure that this feature does not break backwards compatability.

### Data Structures

- [ ] Server - I have ensured that previous versions of stored data structures are
      properly migrated to new formats.
- [ ] Console - I have ensured that previous versions of stored data structures are
      properly migrated to new formats.

### API Changes

- [ ] Server - The server API is backwards-compatible
- The following client APIs are backwards-compatible:
  - [ ] C++
  - [ ] TypeScript
  - [ ] Python

### Breaking Changes

If anything in this section is not true, please list all breaking changes.
