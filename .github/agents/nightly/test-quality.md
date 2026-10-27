# Theme: test quality

Make weak tests prove what they claim. Read `docs/claude/testing.md` and the toolchain
guide of the language first. Pick one package each night, starting with the specs that
changed on `main` in the last 14 days.

Look for:

- **Weak assertions**: a test that checks only that a value exists, is not nil, or did
  not throw, when it can check the value itself. A test with no assertion. An error
  check with no check of the result.
- **Mocks where a real thing works**: `vi.mock` of a Synnax module, a hand-written fake
  of a Synnax interface, or a patched internal. `AGENTS.md` wants the real thing with
  test config (an in-memory DB, a fake cluster). Mock only at the external boundary.
- **Tests of style, not behavior**: assertions on CSS class names.
- **Shape faults**: a `Describe` with a single child, a test that depends on another
  test, and random data.

Change only tests. When a stronger assertion fails, the test found a bug: revert your
change to that test, and put the bug in the report. Run each changed test five times
before you push. One PR per package.
