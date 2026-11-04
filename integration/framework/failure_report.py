#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Pre-rendered per-run failure report (``all-failures.md``).

After the conductor finishes, this module reads each failing test's bundle
(``trace.zip`` + sliced ``server.log`` + traceback) and emits one consolidated markdown
file at the run-dir root. The format is shaped for autonomous LLM triage: stable
headings, code fences, and concise sections so an agent reading the file once can decide
what to drill into.

Trace parsing reads the ``trace.trace`` and ``trace.network`` JSONL streams and the
``trace.stacks`` JSON from the zip directly. The ``pnpm dlx playwright trace`` CLI is
the *interactive* fallback documented in the skill; this module is the *batch* path that
runs at exit.
"""

import re
import traceback as _tb
import zipfile
from dataclasses import dataclass, field
from pathlib import Path

from pydantic import BaseModel, Field, JsonValue, ValidationError

from framework.models import STATUS, Test

# Maximum entries to render per section. Beyond this an agent gets diminishing
# returns and the markdown becomes noisy.
_MAX_ACTIONS = 12
_MAX_CONSOLE_ERRORS = 10
_MAX_NETWORK_FAILURES = 10
_MAX_SERVER_LOG_LINES = 30
_CODE_SNIPPET_CONTEXT = 4

_FAILED_STATUSES = (STATUS.FAILED, STATUS.TIMEOUT, STATUS.KILLED)


@dataclass
class _TraceAction:
    """A single Playwright action with timing and any captured error."""

    call_id: str
    method: str
    params: dict[str, JsonValue]
    start_time: float
    duration_ms: float | None
    error: str | None


@dataclass
class _ConsoleError:
    text: str


@dataclass
class _NetworkFailure:
    method: str
    url: str
    status: int
    status_text: str


@dataclass
class _TraceSummary:
    actions: list[_TraceAction] = field(default_factory=list)
    console_errors: list[_ConsoleError] = field(default_factory=list)
    network_failures: list[_NetworkFailure] = field(default_factory=list)


class _TraceEvent(BaseModel):
    """A line of the ``trace.trace`` stream. Fields that a type of event does not
    carry keep their defaults."""

    type: str = ""
    call_id: str = Field(default="", alias="callId")
    class_name: str = Field(default="", alias="class")
    method: str = ""
    params: dict[str, JsonValue] = Field(default_factory=dict)
    start_time: float | None = Field(default=None, alias="startTime")
    end_time: float | None = Field(default=None, alias="endTime")
    error: JsonValue = None
    message_type: str = Field(default="", alias="messageType")
    text: str = ""


@dataclass
class _Call:
    before: _TraceEvent | None = None
    after: _TraceEvent | None = None


class _Request(BaseModel):
    method: str = ""
    url: str = ""


class _Response(BaseModel):
    status: int = 0
    status_text: str = Field(default="", alias="statusText")


class _Snapshot(BaseModel):
    request: _Request = Field(default_factory=_Request)
    response: _Response = Field(default_factory=_Response)


class _NetworkEvent(BaseModel):
    """A line of the ``trace.network`` stream."""

    snapshot: _Snapshot = Field(default_factory=_Snapshot)


class _Stacks(BaseModel):
    """The ``trace.stacks`` file. Each stack pairs a call ID with its frames, and the
    last item of each frame is the calling function."""

    stacks: list[tuple[str, list[list[JsonValue]]]] = Field(default_factory=list)


def _parse_trace(trace_zip: Path) -> _TraceSummary | None:
    """Parse a Playwright trace.zip into a structured summary.

    Returns None if the zip is missing or malformed. Tracing errors must never
    bring down report generation.
    """
    if not trace_zip.exists():
        return None
    summary = _TraceSummary()
    paired: dict[str, _Call] = {}
    try:
        with zipfile.ZipFile(trace_zip) as z:
            _parse_trace_events(z, paired, summary)
            _parse_network_events(z, summary)
            teardown_calls = _parse_teardown_calls(z)
    except (zipfile.BadZipFile, KeyError, OSError):
        return None

    for call_id, call in paired.items():
        before = call.before
        if before is None or call_id in teardown_calls:
            continue
        after = call.after
        duration = None
        if (
            after is not None
            and after.end_time is not None
            and before.start_time is not None
        ):
            duration = after.end_time - before.start_time
        summary.actions.append(
            _TraceAction(
                call_id=call_id,
                method=f"{before.class_name}.{before.method}".strip("."),
                params=before.params,
                start_time=before.start_time or 0.0,
                duration_ms=duration,
                error=None if after is None else _action_error(after.error),
            )
        )
    summary.actions.sort(key=lambda a: a.start_time)
    return summary


def _parse_trace_events(
    z: zipfile.ZipFile,
    paired: dict[str, _Call],
    summary: _TraceSummary,
) -> None:
    try:
        info = z.getinfo("trace.trace")
    except KeyError:
        return
    with z.open(info) as f:
        for raw in f:
            try:
                evt = _TraceEvent.model_validate_json(raw)
            except ValidationError:
                continue
            if evt.type == "before":
                paired.setdefault(evt.call_id, _Call()).before = evt
            elif evt.type == "after":
                paired.setdefault(evt.call_id, _Call()).after = evt
            elif evt.type == "console" and evt.message_type == "error":
                summary.console_errors.append(_ConsoleError(text=evt.text))


def _parse_teardown_calls(z: zipfile.ZipFile) -> set[str]:
    """Return the call IDs of actions made from a ``teardown`` method. The report leaves
    them out so its last actions end at the failure, not at cleanup."""
    try:
        info = z.getinfo("trace.stacks")
    except KeyError:
        return set()
    with z.open(info) as f:
        try:
            stacks = _Stacks.model_validate_json(f.read()).stacks
        except ValidationError:
            return set()
    return {
        call_id
        for call_id, frames in stacks
        if any(
            isinstance(caller, str) and caller.endswith(".teardown")
            for *_, caller in frames
        )
    }


def _parse_network_events(z: zipfile.ZipFile, summary: _TraceSummary) -> None:
    try:
        info = z.getinfo("trace.network")
    except KeyError:
        return
    with z.open(info) as f:
        for raw in f:
            try:
                snap = _NetworkEvent.model_validate_json(raw).snapshot
            except ValidationError:
                continue
            if snap.response.status >= 400:
                summary.network_failures.append(
                    _NetworkFailure(
                        method=snap.request.method,
                        url=snap.request.url,
                        status=snap.response.status,
                        status_text=snap.response.status_text,
                    )
                )


def _action_error(err: JsonValue) -> str | None:
    if err is None:
        return None
    if isinstance(err, dict):
        message = err.get("message") or err.get("name")
        return None if message is None else str(message)
    return str(err)


def _render_failure_report(run_dir: Path, tests: list[Test]) -> Path | None:
    """Write ``run_dir/all-failures.md`` if any tests failed. Returns the path."""
    failing = [t for t in tests if t.status in _FAILED_STATUSES]
    if not failing:
        return None

    total = len(tests)
    lines: list[str] = [
        f"# {run_dir.name}: {len(failing)} failed / {total} total",
        "",
        "Generated from each failure's `trace.zip`, sliced `server.log`, and",
        "Python traceback. Each section is anchored to its `bundle_dir`; navigate",
        "there for the raw artifacts.",
        "",
    ]
    for i, test in enumerate(failing, 1):
        lines.extend(_render_test(i, test, run_dir))
        lines.append("")

    out = run_dir / "all-failures.md"
    out.write_text("\n".join(lines) + "\n", encoding="utf-8")
    return out


def _render_test(idx: int, test: Test, run_dir: Path) -> list[str]:
    bundle = Path(test.bundle_dir) if test.bundle_dir else None
    rel_bundle = (
        bundle.relative_to(run_dir).as_posix()
        if bundle and bundle.is_absolute()
        else (test.bundle_dir or "<unknown>")
    )
    dur = _dur(test)
    duration = f"{dur:.2f}s" if dur is not None else "?"
    out: list[str] = [
        f"## {idx}. {test.test_name}",
        "",
        f"**Status:** `{test.status.name}`  **Duration:** {duration}  "
        f"**Bundle:** `{rel_bundle}/`",
        "",
    ]
    if test.error_message:
        out.append("### Error")
        out.append("")
        out.append("```")
        out.append(test.error_message)
        out.append("```")
        out.append("")
    if test.error_traceback:
        snippet = _code_snippet_from_traceback(test.error_traceback)
        if snippet:
            out.append("### Failing source")
            out.append("")
            out.append(snippet)
            out.append("")
        out.append("<details><summary>Python traceback</summary>")
        out.append("")
        out.append("```")
        out.append(test.error_traceback.rstrip())
        out.append("```")
        out.append("")
        out.append("</details>")
        out.append("")

    summary = _parse_trace(bundle / "trace.zip") if bundle else None
    if summary is not None:
        out.extend(_render_actions(summary))
        out.extend(_render_console(summary))
        out.extend(_render_network(summary))
    else:
        out.append("_No trace.zip recorded for this test._")
        out.append("")

    out.extend(_render_server_log_tail(bundle))
    return out


def _dur(test: Test) -> float | None:
    if test.started_at is None or test.ended_at is None:
        return None
    return test.ended_at - test.started_at


def _render_actions(summary: _TraceSummary) -> list[str]:
    if not summary.actions:
        return []
    actions = summary.actions[-_MAX_ACTIONS:]
    out: list[str] = ["### Last actions", ""]
    out.append("```")
    for a in actions:
        marker = "✗" if a.error else " "
        dur = f" ({a.duration_ms:.0f}ms)" if a.duration_ms is not None else ""
        target = _action_target(a)
        out.append(f"{marker} {a.method}{dur}  {target}".rstrip())
        if a.error:
            out.append(f"    ↳ {a.error}")
    out.append("```")
    out.append("")
    return out


def _action_target(action: _TraceAction) -> str:
    """One-line summary of what an action targeted (selector, url, etc.)."""
    p = action.params
    for key in ("selector", "url", "name", "value", "text"):
        v = p.get(key)
        if v:
            return f"{key}={_truncate(str(v), 80)}"
    return ""


def _render_console(summary: _TraceSummary) -> list[str]:
    if not summary.console_errors:
        return []
    out: list[str] = ["### Browser console errors", ""]
    out.append("```")
    out.extend(
        _truncate(e.text, 200) for e in summary.console_errors[:_MAX_CONSOLE_ERRORS]
    )
    if len(summary.console_errors) > _MAX_CONSOLE_ERRORS:
        out.append(f"... +{len(summary.console_errors) - _MAX_CONSOLE_ERRORS} more")
    out.append("```")
    out.append("")
    return out


def _render_network(summary: _TraceSummary) -> list[str]:
    if not summary.network_failures:
        return []
    out: list[str] = ["### Network failures", ""]
    out.append("```")
    out.extend(
        f"{f.status} {f.status_text:<15} {f.method} {_truncate(f.url, 100)}".rstrip()
        for f in summary.network_failures[:_MAX_NETWORK_FAILURES]
    )
    if len(summary.network_failures) > _MAX_NETWORK_FAILURES:
        out.append(f"... +{len(summary.network_failures) - _MAX_NETWORK_FAILURES} more")
    out.append("```")
    out.append("")
    return out


def _render_server_log_tail(bundle: Path | None) -> list[str]:
    if bundle is None:
        return []
    log = bundle / "server.log"
    if not log.exists() or log.stat().st_size == 0:
        return []
    try:
        text = log.read_text(encoding="utf-8", errors="replace")
    except OSError:
        return []
    lines = text.splitlines()
    if not lines:
        return []
    interesting = [
        line
        for line in lines
        if re.search(r"\b(ERROR|WARN|FATAL|panic)\b", line, re.IGNORECASE)
    ]
    rendered = interesting[-_MAX_SERVER_LOG_LINES:] or lines[-_MAX_SERVER_LOG_LINES:]
    label = (
        "Server log (matching ERROR/WARN/FATAL)" if interesting else "Server log (tail)"
    )
    out: list[str] = [f"### {label}", ""]
    out.append("```")
    out.extend(rendered)
    out.append("```")
    out.append("")
    return out


_FRAME_PATTERN = re.compile(
    r'^\s*File "(?P<file>[^"]+)", line (?P<line>\d+), in (?P<func>\S+)\s*$',
    re.MULTILINE,
)


def _code_snippet_from_traceback(tb_text: str) -> str | None:
    """Render a markdown snippet around the most user-actionable frame.

    Priority order, walking the traceback bottom-up:
      1. Deepest frame under ``integration/tests/`` (the test author's code).
      2. Deepest frame under ``integration/`` outside ``framework/`` (helpers).
      3. Deepest frame outside ``site-packages`` and Playwright.

    Paths are resolved before classification so that ``framework/../tests/``
    style references (produced when the conductor imports test modules by
    relative path) compare equal to their canonical ``integration/tests/`` form.
    """
    test_frame: dict[str, str] | None = None
    helper_frame: dict[str, str] | None = None
    other_frame: dict[str, str] | None = None
    for match in _FRAME_PATTERN.finditer(tb_text):
        raw = match.group("file")
        try:
            resolved = str(Path(raw).resolve())
        except OSError:
            resolved = raw
        if "site-packages" in resolved or "/playwright/" in resolved:
            continue
        if "/integration/framework/" in resolved:
            continue
        frame = {**match.groupdict(), "file": resolved}
        if "/integration/tests/" in resolved:
            test_frame = frame
        elif "/integration/" in resolved:
            helper_frame = frame
        else:
            other_frame = frame
    target = test_frame or helper_frame or other_frame
    if target is None:
        return None
    file_path = Path(target["file"])
    if not file_path.exists():
        return None
    try:
        lineno = int(target["line"])
    except ValueError:
        return None
    try:
        source = file_path.read_text(encoding="utf-8", errors="replace").splitlines()
    except OSError:
        return None
    start = max(0, lineno - 1 - _CODE_SNIPPET_CONTEXT)
    end = min(len(source), lineno + _CODE_SNIPPET_CONTEXT)
    width = len(str(end))
    fenced: list[str] = ["```python"]
    fenced.append(f"# {file_path}:{lineno} in {target['func']}")
    for i in range(start, end):
        marker = ">" if (i + 1) == lineno else " "
        fenced.append(f"{marker} {str(i + 1).rjust(width)}  {source[i]}")
    fenced.append("```")
    return "\n".join(fenced)


def _truncate(text: str, limit: int) -> str:
    if len(text) <= limit:
        return text
    return text[: limit - 1] + "…"


def safe_render(run_dir: Path, *, tests: list[Test]) -> Path | None:
    """Render the report, swallowing any unexpected error.

    Failure-report generation must never break the run's exit path. On any
    unexpected exception the conductor logs the failure but the bundle is still
    valid.
    """
    try:
        return _render_failure_report(run_dir, tests)
    except Exception:
        # Surface the traceback into the bundle so the next debugging session
        # can find it, rather than silently losing it.
        try:
            (run_dir / "all-failures.error.txt").write_text(
                _tb.format_exc(), encoding="utf-8"
            )
        except OSError:
            pass
        return None
