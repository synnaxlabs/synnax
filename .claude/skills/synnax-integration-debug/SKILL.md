---
name: synnax-integration-debug
description: Triage failures from the Synnax integration test suite (`uv run tc ...`). Use when a console/driver/arc test fails locally, when the user pastes a CI workflow URL for a `Test - Integration` run, or when they reference a debug bundle, `summary.json`, `trace.zip`, or a `run-<timestamp>/` directory. Walks the bundle in order: summary.json → per-test artifacts → trace inspection → server log.
allowed-tools: Bash, Read
---

# Synnax Integration Test Debugging

When the integration test conductor (`uv run tc ...`) finishes, it writes a
self-contained debug bundle to
`integration/tests/results/run-<UTC-timestamp>[-<name>]/`. A `latest` symlink in the
same dir points at the most recent run. **Always start here when triaging an integration
failure.** The bundle is identical between local and CI; in CI it's uploaded as a single
`test-results-*` artifact.

## From a CI failure link

When the user pastes a GitHub Actions URL for a failed `Test - Integration` run:

1. Extract the run ID from the URL (the digits after `/actions/runs/`).
2. List the workflow's artifacts and download the bundle:
   ```bash
   gh run view <run-id>                                  # confirm which jobs failed
   gh run download <run-id> --dir /tmp/synnax-bundle    # downloads all artifacts
   ```
   The integration suite uploads one artifact per OS/matrix entry named
   `test-results-<os>-<name>`. Each artifact contains exactly one `run-<ts>-*/`
   directory.
3. `cd /tmp/synnax-bundle/test-results-<os>-<name>/run-<ts>-*` and proceed with the
   Workflow below. The bundle layout is identical to local.
4. To see the conductor's stdout (including the `→ bundle:` breadcrumb and
   per-test error messages), use:
   ```bash
   gh run view <run-id> --log-failed                  # only failed steps
   gh run view <run-id> --log | grep -E "tc >|FAILED" # conductor lines
   ```

If `gh` is unavailable, the same artifact is downloadable from the workflow run page in
the browser under "Artifacts" at the bottom.

## Workflow

1. **Read `all-failures.md`** at the run root (present only when there were failures).
   Each failure has: error, Python traceback, source snippet at the failing line, last
   12 trace actions, browser console errors, network failures, and a tail of the sliced
   server log. This is the highest-signal-per-token entry point. Read it first.
2. **Cross-reference `summary.json`** for run-level stats and the exact per-test bundle
   path. Failures and timeouts are the ones to triage.
3. **For each failure still ambiguous after step 1, drill into `tests/<name>/`**
   (`trace.zip` + sliced `server.log` + screenshots/exports). The trace CLI cheatsheet
   below is for when you need to inspect DOM at a specific action or compare snapshots.
4. **Cross-reference the server log slice** for backend cause of any 4xx/5xx the trace
   shows.

Do not parse `trace.zip` directly. The `npx playwright trace` CLI is the supported
interactive interface; the batch-mode parsing already done by `all-failures.md` covers
most triage needs.

## Bundle layout

```
integration/tests/results/
├── latest -> run-<ts>-<name>/        # symlink to most recent run
└── run-<ts>-<name>/
    ├── README.md                      # short triage guide for cold landings
    ├── all-failures.md                # consolidated failure report (only on failures)
    ├── summary.json                   # run + per-test outcomes
    ├── server.log                     # full server log for this run (CI only by default)
    └── tests/
        └── <test-name>/
            ├── trace.zip              # Playwright trace, only on FAILED/TIMEOUT/KILLED
            ├── server.log             # byte-sliced to this test's wall-clock window
            ├── *.png                  # screenshots from `console.screenshot()`
            └── *.json, *.csv          # exports from `console.workspace.export()` etc.
```

Per-test artifacts only exist for tests that actually wrote them. A passing test
typically has an empty bundle dir.

## Reading `summary.json`

```bash
# Quick failure overview from any worktree:
jq '.tests[] | select(.status != "PASSED" and .status != "FLAKY")' \
  integration/tests/results/latest/summary.json
```

Fields:

- `case` -> registry path like `console/pages/open_close`
- `name` -> display name (used as the bundle subdir slug)
- `status` -> `PASSED`, `FLAKY`, `FAILED`, `TIMEOUT`, `KILLED`
- `started_at` / `ended_at` -> ISO UTC; use these to correlate with external logs
- `duration_s` -> wall-clock seconds
- `error_message` -> populated when the test called `self.fail(msg)`
- `bundle_dir` -> relative path under the run dir

## Inspecting the trace

The Playwright trace CLI is bundled with the Python `playwright` package and is
invokable via npx (matches the `@playwright/test` shipped CLI). Stay in shell. Don't
open the GUI viewer for autonomous triage.

```bash
# 1. Open the trace
npx -y -p @playwright/test playwright trace open \
    integration/tests/results/latest/tests/<name>/trace.zip

# 2. List all actions with their IDs (failures only)
npx -y -p @playwright/test playwright trace actions --errors-only

# 3. Drill into a specific action: params, error, log, source, snapshots
npx -y -p @playwright/test playwright trace action <id>

# 4. Page state at that moment
npx -y -p @playwright/test playwright trace snapshot <id>
npx -y -p @playwright/test playwright trace snapshot <id> -- eval "document.querySelector('.error').textContent"

# 5. Network failures
npx -y -p @playwright/test playwright trace requests --failed

# 6. Console errors
npx -y -p @playwright/test playwright trace console --errors-only

# 7. All trace errors with stacks
npx -y -p @playwright/test playwright trace errors

# 8. Cleanup
npx -y -p @playwright/test playwright trace close
```

All commands after `open` operate on the currently-extracted trace. Opening a new trace
replaces the previous one.

## Reading the sliced server log

Per-test `server.log` is the byte range of the full server log captured between this
test's `started_at` and `ended_at`. Format depends on how the server was launched (text
or JSON; check the first line). Greppable as-is.

```bash
# Recent ERRORs in the failing test's window:
grep -i "ERROR\|WARN" integration/tests/results/latest/tests/<name>/server.log

# JSON format: pull error-level entries:
jq -c 'select(.level == "error")' \
   integration/tests/results/latest/tests/<name>/server.log
```

For asynchronous test sequences, the slice contains the union of all parallel tests in
that wall-clock window. Filter by request path or trace fields to attribute lines.

## Common gotchas

- **Local dev has no `server.log` by default.** The conductor reads from
  `$SYNNAX_SERVER_LOG` (default `~/synnax-data/synnax-core.log`). CI's `start_core.sh`
  redirects there automatically. Locally, the user must redirect:
  `synnax start -mi > ~/synnax-data/synnax-core.log 2>&1`. Absent the file, the bundle
  is still valid; just no server log slice.
- **`trace.zip` only exists for failures.** A passing test's bundle dir is intentionally
  empty; `tracing.stop()` discards on pass.
- **`bundle_dir` field is relative.** Resolve against the run dir, not the integration
  root.
- **`latest` symlink may be absent on Windows/restricted FS.** Use the most recent
  `run-<ts>-*` dir by sort order in that case.
- **Pre-bundle artifacts.** Anything written before the conductor created its run dir
  (or by code that bypasses `get_results_path`) lands at the flat results root. Rare.

## Next steps after triage

- Stale locator / element-not-found → check `npx playwright trace snapshot <id>` and
  look for renamed/moved elements in nearby DOM.
- Backend 500 / 4xx → server log slice will show the handler's error stack; check
  `core/pkg/service/...` for the relevant endpoint.
- Timeout with no failed actions → likely a Playwright `wait_for_*` exceeded its budget;
  the action just before the timeout shows what was being waited on.
- "EXCEPTION" in `error_message` with a Python traceback → setup/teardown failure, not a
  UI failure. The trace is partial but the test's own log (printed inline by `tc`) has
  the stack.
