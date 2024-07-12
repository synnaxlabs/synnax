# Feature Pull Request Template

## Key Information

- **Linear Issue**: [SY-###]()

## Description

Maximum 2-3 sentence description describing the changes.

## Basic Readiness Checklist

- [ ] I have performed a self-review of my code.
- [ ] I have added relevant tests to cover the changes to CI.
- [ ] I have updated user facing documentation accordingly.
- [ ] I have verified code coverage targets are met.

## Migrations

- [ ] Console - I have ensured that previous versions of stored data structures are 
properly migrated to new formats.
- [ ] Server - I have ensured that previous versions of stored data structures are 
properly migrated to new formats.

# Additional Notes
- [ ] These changes deal with concurrency
- [ ] These changes affect UI

## Manual QA Additions

- [ ] I have updated the [Release Candidate](/.github/PULL_REQUEST_TEMPLATE/rc.md) template
with necessary manual QA steps to test my change.

## Breaking Changes

Please list any breaking changes to public or internal packages.

## Reviwer Checklist
- [ ] Sufficient test coverage of new additions.
- [ ] Verified all steps in the Readiness checklists.
- [ ] UI changes have been tested.
- [ ] Style and formatting is consistent.
