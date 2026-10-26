# Explore the Console as a QA tester

Use the Synnax Console like a demanding QA tester for a fixed time, and file one report
of the bugs, bad UI, and bad UX you find. Nobody watches this session, so do not ask
questions. Change no code and open no pull request.

## Environment

`integration/qa/up.sh` has already run. If `curl -s localhost:9222/json/version` gives
nothing, run it yourself (it needs `SYNNAX_LICENSE_KEY`). It gives you:

- A Core with the web Console at `http://localhost:9090`, login `synnax` / `seldon`.
- The sample tank project open: a schematic, a line plot, and two Arc programs. Click
  **Start** on the schematic and the tank pressurizes, so the plot moves.
- Channels `qa_sine`, `qa_noise`, and `qa_square` on index `qa_time`: one hour of
  history and live data at 50 Hz. The range `QA history` covers the history.
- A headless Chromium on CDP port 9222, already logged in. Drive it with short Python
  Playwright scripts that you run with `uv run python` from `integration/`:

```python
from playwright.sync_api import sync_playwright

with sync_playwright() as p:
    page = p.chromium.connect_over_cdp("http://localhost:9222").contexts[0].pages[0]
    page.on("console", lambda m: m.type == "error" and print("console:", m.text))
    page.set_viewport_size({"width": 1440, "height": 900})
    page.get_by_role("button", name="...").click()
    page.screenshot(path="/tmp/console-qa/images/step.png")
```

Never close the browser or the page: state carries over between scripts. The viewport
size does not, so set it in each script. Look at every screenshot you take. The pixels
are the evidence, not the DOM. The schematic opens in edit mode: click **Disable
editing** before you click its buttons. Drags from a list to a plot do not work through
CDP, so add channels through the visualization toolbar.

## What to try

Try everything a Console user can do, and push each one past the happy path:

- Layout: split, drag, move, rename, and close mosaic tabs; the nav drawers; the command
  palette; keyboard shortcuts; windows opened from the palette.
- Visualizations: line plots (axes, ranges, legends, tooltips, zoom and pan, live data),
  schematics (symbols, edit mode, control chips, the sample's buttons), tables, and
  logs. Create, edit, rename, and delete each one.
- Resources: ranges, channels (and calculated channels), devices, tasks, workspaces or
  projects, labels, users, and statuses. Search, filter, and context menus.
- Import and export of each kind of resource that has it.
- Settings, and the light and dark themes. Check contrast in both themes.
- Error states: bad input in every form, a deleted channel on a plot, an empty search, a
  Core that stops (`kill` the Core PID in `/tmp/console-qa/pids` near the end only, then
  start it again with the same command from `up.sh`).
- Sizes: do the main pages at 1280x720, 1440x900, 1920x1080, 2560x1440, 800x900, and
  480x900.

Look for overlapped, clipped, or cut-off elements; misalignment; unreadable contrast;
spacing or names that do not agree from page to page; slow or jumpy interactions;
confusing flows; console errors; and states that do not recover.

## Findings

Report a finding only after you make it happen twice. For each one, record:

- **Severity.** High: wrong data, a crash, lost work, or a flow that cannot finish.
  Medium: a broken layout at a common size, a confusing flow, or a console error with a
  visible effect. Low: spacing, alignment, naming, or small contrast faults.
- **Steps**: exact, numbered, from a logged-in Console with the sample project open,
  with the window size.
- **Expected** and **actual** behavior, in one line each.
- **Screenshot**: a PNG in `/tmp/console-qa/images/` with a short descriptive name. Crop
  or zoom so the fault is clear.
- Console errors, verbatim, if any.

If `gh issue list --label console-qa --state open` works, read the newest issue and mark
a finding that is already in it as "Seen before" instead of describing it again.

## Report

Explore for 60 minutes, or for the time budget at the end of this prompt if it gives
one. Then write `/tmp/console-qa/report.md` in ASD-STE100 Simplified Technical English:

- Line 1: `# Console QA <YYYY-MM-DD>: <N> findings (<H> high)`.
- A table: number, severity, area, one-line title. Highest severity first.
- One `## <number>. <title>` section per finding, with the fields above. Link its
  screenshot as `![<title>](images/<name>.png)`.
- `## Coverage`: what you tried, and what you could not try and why.

Then publish it. This is the one GitHub write you may do:

```sh
uv run --project .github/scripts python .github/scripts/publish_qa_report.py \
  /tmp/console-qa
```

Add `--dry-run` when this prompt says dry run. Your last message is the issue URL (or
the `issue.md` path in a dry run) and the line-1 title of the report.
