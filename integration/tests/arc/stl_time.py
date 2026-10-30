#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import re
from datetime import UTC, datetime, timedelta

import synnax as sy
from framework.utils import create_indexed_pair, create_virtual_channels
from tests.arc.arc import ArcCase

ARC_STL_TIME_SOURCE = """
import time
authority 200
// ──────────────────────────── time.now ───────────────────────────────
func write_now() {
    time_now_int_out = time.now()
    time_now_ts_out = time.now()
}
time_trigger -> write_now()

// time.now() as a flow node: triggered by any upstream, outputs timestamp.
time_now_flow_ts_trigger -> time.now() -> time_now_flow_ts_out
time_now_flow_int_trigger -> time.now() -> time_now_flow_int_out

// Formats a timestamp as ISO 8601 in UTC. The call is the same in a function
// body and in a flow statement; the flow form lowers to an expression node.
func write_format() {
    time_format_fn_out = time.format(time.now())
}
time_format_fn_trigger -> write_format()
time_format_flow_trigger -> time.format(time.now()) -> time_format_flow_out
// time.now() with a format.
func write_format_now_lit() {
    now_fmt_fn_out = time.format(time.now(), "%Y-%m-%d %H:%M:%S.%f")
}
now_fmt_trigger -> write_format_now_lit()
now_fmt_trigger -> time.format(time.now(), "%Y-%m-%d %H:%M:%S.%f") -> now_fmt_flow_out
// A stamp read from a channel, so the test can expect an exact string. It is
// formatted bare, with a literal format, and into an f-string through a local.
func write_format_fixed() {
    time_format_fixed_fn_out = time.format(time_format_ts_in)
    time_format_lit_fn_out = time.format(time_format_ts_in, "%Y-%m-%d %H:%M:%S.%f")
    clock := time.format(time_format_ts_in, "%H:%M:%S")
    time_fstr_out = f"at {clock}"
}
time_format_ts_in -> write_format_fixed()
time.format(time_format_ts_in) -> time_format_fixed_flow_out
time.format(time_format_ts_in, "%Y-%m-%d %H:%M:%S.%f") -> time_format_lit_flow_out
// A format read from a channel compiles through a different path than a literal.
func write_format_chan() {
    time_format_chan_fn_out = time.format(time_format_ts_in, time_format_in)
}
time_format_in -> write_format_chan()
time.format(time_format_ts_in, time_format_in) -> time_format_chan_flow_out

// ─────────────────────────── time.interval ──────────────────────────
// interval is inherently time-triggered (that's the point of the function).
// We use time.interval() (qualified syntax) to verify module syntax works.
// standalone
func count_intervals() {
    interval_count = interval_count + 1
}
interval(period=100ms) -> count_intervals()
// module-qualified
func count_intervals_mod() {
    interval_count_mod = interval_count_mod + 1
}
time.interval(period=100ms) -> count_intervals_mod()
// ──────────────────────────── time.wait ─────────────────────────────
// Regression test: wait{3s} previously took 5-7s due to a runtime bug.
// 3 seconds was chosen because it's long enough to measure accurately
// with wall-clock timers but short enough to keep test runtime low.
// The [2.5, 4.0]s tolerance window accounts for OS scheduling jitter
// and Arc's 5ms minimum timer tolerance.
// standalone
sequence main_standalone {
    stage stage1 {
        1 -> toggle_cmd
        wait(3s) => stage2
    }
    stage stage2 {
        0 -> toggle_cmd
    }
}
start_wait_cmd => main_standalone
// module-qualified
sequence main_module {
    stage stage1 {
        1 -> toggle_cmd_mod
        time.wait(3s) => stage2
    }
    stage stage2 {
        0 -> toggle_cmd_mod
    }
}
start_wait_mod_cmd => main_module
"""

# Wait timing bounds (seconds). See regression comment in Arc source above.
MAX_WAIT_DURATION = 4.0
MIN_WAIT_DURATION = 2.5

# Jan 1, 2020 00:00:00 UTC in nanoseconds — sanity floor for time.now().
# Catches unit bugs (seconds vs. nanos) or zero returns.
JAN_2020_NANOS = 1577836800000000000

# time.format output: ISO 8601 UTC, nanosecond fraction with trailing zeros trimmed
# and omitted when zero, Z suffix.
ISO_8601_UTC = re.compile(r"^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d{1,9})?Z$")

# Allowed gap between a formatted time.now() and the wall clock when it is read.
MAX_FORMAT_DRIFT = timedelta(seconds=2)

# The format the Arc source above uses as a literal, and the shape of its output.
PYTHON_FORMAT = "%Y-%m-%d %H:%M:%S.%f"
PYTHON_FORMAT_OUTPUT = re.compile(r"^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}\.\d{6}$")

# (ns since the Unix epoch, ISO 8601, PYTHON_FORMAT). Each stamp is written to
# time_format_ts_in and formatted in both contexts. The f-string output takes the
# clock part of the ISO form.
FORMAT_CASES = (
    (
        1_326_789_125_000_000_000,
        "2012-01-17T08:32:05Z",
        "2012-01-17 08:32:05.000000",
    ),
    (
        1_326_789_125_250_000_000,
        "2012-01-17T08:32:05.25Z",
        "2012-01-17 08:32:05.250000",
    ),
    (
        1_709_251_199_999_999_999,
        "2024-02-29T23:59:59.999999999Z",
        "2024-02-29 23:59:59.999999",
    ),
    (
        946_684_800_000_000_000,
        "2000-01-01T00:00:00Z",
        "2000-01-01 00:00:00.000000",
    ),
)

# Tuesday, January 17, 2012 at 08:32:05.25 UTC, in ISO week 3.
STAMP = 1_326_789_125_250_000_000

# (format, expected for STAMP), each written to time_format_in and formatted in both
# contexts. Every supported code, then composites and text, then what the runtime
# copies through unchanged. Consecutive expectations differ, so each wait proves a
# new write landed.
CODE_CASES = (
    ("%Y", "2012"),
    ("%y", "12"),
    ("%m", "01"),
    ("%d", "17"),
    ("%H", "08"),
    ("%e", "17"),
    ("%I", "08"),
    ("%M", "32"),
    ("%S", "05"),
    ("%j", "017"),
    ("%f", "250000"),
    ("%N", "250000000"),
    ("%u", "2"),
    ("%U", "03"),
    ("%w", "2"),
    ("%W", "03"),
    ("%G", "2012"),
    ("%V", "03"),
    ("%z", "+0000"),
    ("%Z", "UTC"),
    ("%F", "2012-01-17"),
    ("%T", "08:32:05"),
    ("%%", "%"),
    ("%Y-%m-%d %H:%M:%S.%f", "2012-01-17 08:32:05.250000"),
    ("%H:%M:%S.%N", "08:32:05.250000000"),
    ("T+%S s", "T+05 s"),
    ("%Q %", "%Q %"),
)

# (ns since the Unix epoch, format, expected) at the calendar edges. Week numbers,
# the 12-hour clock, ISO week rollover, the epoch, and a stamp before it.
EDGE_CASES = (
    # Sunday, January 1, 2012 at midnight.
    (1_325_376_000_000_000_000, "%u %w %U %W %I %e %j", "7 0 01 00 12  1 001"),
    # Noon and one o'clock on January 17, 2012.
    (1_326_801_600_000_000_000, "%I %H", "12 12"),
    (1_326_805_200_000_000_000, "%H %I", "13 01"),
    # January 1, 2021 falls in ISO week 53 of 2020.
    (1_609_459_200_000_000_000, "%G-%V %Y", "2020-53 2021"),
    # December 30, 2024 falls in ISO week 1 of 2025.
    (1_735_516_800_000_000_000, "%G-%V %Y", "2025-01 2024"),
    # The last nanosecond of leap day 2024.
    (1_709_251_199_999_999_999, "%F %T.%N %j", "2024-02-29 23:59:59.999999999 060"),
    (0, "%F %T %Z", "1970-01-01 00:00:00 UTC"),
    (-1_000_000_000_000_000_000, "%F %T", "1938-04-24 22:13:20"),
)

# Codes Python documents for every platform. The rest are checked by hand above.
PYTHON_CODES = set("YymdHIMSjfUWwzZ%")


def python_strftime(ns: int, fmt: str) -> str | None:
    """Python's rendering of format at ns, or None when format uses a code Python
    does not support on every platform."""
    if any(code not in PYTHON_CODES for code in re.findall(r"%(.)", fmt)):
        return None
    stamp = datetime(1970, 1, 1, tzinfo=UTC) + timedelta(microseconds=ns // 1000)
    return stamp.strftime(fmt)


# (channel, label) for the two free-running interval counters.
INTERVAL_CHANNELS = (
    ("interval_count", "standalone"),
    ("interval_count_mod", "module"),
)

# (start_cmd, toggle, label) for the two wait sequences measured concurrently.
WAIT_CASES = (
    ("start_wait_cmd", "toggle_cmd", "standalone"),
    ("start_wait_mod_cmd", "toggle_cmd_mod", "module"),
)


class StlTime(ArcCase):
    """Test time module with qualified syntax: time.now(), time.format(),
    time.interval(), time.wait().

    time.now() is verified by checking the returned timestamp is a valid
    nanosecond value. time.format() is verified by parsing its output as ISO
    8601 and checking it against the wall clock, in both a function body and
    a flow statement, then with every format code on fixed stamps, from a
    literal and from a channel. time.interval() is verified by measuring its firing
    rate over 1 second. time.wait() is a regression test migrated from
    wait_timing.py verifying that a 3-second wait completes in ~3 seconds.
    """

    arc_source = ARC_STL_TIME_SOURCE
    arc_name_prefix = "ArcStlTime"
    start_cmd_channel = "start_stl_time_cmd"
    subscribe_channels = [
        "time_now_int_out",
        "time_now_ts_out",
        "time_now_flow_ts_out",
        "time_now_flow_int_out",
        "time_format_fn_out",
        "time_format_flow_out",
        "now_fmt_fn_out",
        "now_fmt_flow_out",
        "time_format_fixed_fn_out",
        "time_format_fixed_flow_out",
        "time_format_lit_fn_out",
        "time_format_lit_flow_out",
        "time_fstr_out",
        "time_format_chan_fn_out",
        "time_format_chan_flow_out",
        "interval_count",
        "interval_count_mod",
        "toggle_cmd",
        "toggle_cmd_mod",
    ]

    def setup(self) -> None:
        create_virtual_channels(
            self.client,
            [
                ("time_trigger", sy.DataType.FLOAT64),
                ("time_now_int_out", sy.DataType.INT64),
                ("time_now_ts_out", sy.DataType.TIMESTAMP),
                ("time_now_flow_ts_trigger", sy.DataType.FLOAT64),
                ("time_now_flow_ts_out", sy.DataType.TIMESTAMP),
                ("time_now_flow_int_trigger", sy.DataType.FLOAT64),
                ("time_now_flow_int_out", sy.DataType.INT64),
                ("time_format_fn_trigger", sy.DataType.FLOAT64),
                ("time_format_fn_out", sy.DataType.STRING),
                ("time_format_flow_trigger", sy.DataType.FLOAT64),
                ("time_format_flow_out", sy.DataType.STRING),
                ("time_format_ts_in", sy.DataType.TIMESTAMP),
                ("now_fmt_trigger", sy.DataType.FLOAT64),
                ("now_fmt_fn_out", sy.DataType.STRING),
                ("now_fmt_flow_out", sy.DataType.STRING),
                ("time_format_fixed_fn_out", sy.DataType.STRING),
                ("time_format_fixed_flow_out", sy.DataType.STRING),
                ("time_format_lit_fn_out", sy.DataType.STRING),
                ("time_format_lit_flow_out", sy.DataType.STRING),
                ("time_fstr_out", sy.DataType.STRING),
                ("time_format_in", sy.DataType.STRING),
                ("time_format_chan_fn_out", sy.DataType.STRING),
                ("time_format_chan_flow_out", sy.DataType.STRING),
                ("interval_count", sy.DataType.INT64),
                ("interval_count_mod", sy.DataType.INT64),
                ("start_wait_cmd", sy.DataType.UINT8),
                ("start_wait_mod_cmd", sy.DataType.UINT8),
            ],
        )
        for name in ("toggle_cmd", "toggle_cmd_mod"):
            create_indexed_pair(self.client, name, data_type=sy.DataType.UINT8)
        super().setup()

    def _test_now(self) -> None:
        self.log("=== time.now() [WASM] ===")
        self.writer.write("time_trigger", 1.0)
        self.log(f"Expecting time_now_int_out > {JAN_2020_NANOS} (Jan 1, 2020 nanos)")
        self.wait_for_gt("time_now_int_out", JAN_2020_NANOS)
        self.log("time.now() returned a valid timestamp (int64 channel)")
        self.log(f"Expecting time_now_ts_out > {JAN_2020_NANOS} (Jan 1, 2020 nanos)")
        self.wait_for_gt("time_now_ts_out", JAN_2020_NANOS)
        self.log("time.now() wrote successfully to a timestamp channel")

    def _test_now_flow(self) -> None:
        self.log("=== time.now() [Flow] ===")
        self.writer.write("time_now_flow_ts_trigger", 1.0)
        self.wait_for_gt("time_now_flow_ts_out", 0)
        ts = self.read_tlm("time_now_flow_ts_out")
        if not isinstance(ts, sy.TimeStamp):
            self.fail(f"time.now{{}} wrote {ts!r}, not a TimeStamp")
        now = sy.TimeStamp.now()
        drift = abs(now - ts)
        self.log(
            f"time.now{{}} returned {ts}, current time {now}, "
            f"drift {drift.milliseconds:.1f}ms (max 500ms)"
        )
        if drift > 500 * sy.TimeSpan.MILLISECOND:
            self.fail(
                f"time.now{{}} drift {drift.milliseconds:.1f}ms exceeds 500ms tolerance"
            )
        self.log(
            f"Expecting time_now_flow_int_out > {JAN_2020_NANOS} (Jan 1, 2020 nanos)"
        )
        self.writer.write("time_now_flow_int_trigger", 1.0)
        self.wait_for_gt("time_now_flow_int_out", JAN_2020_NANOS)

    def _assert_format(self, channel: str, label: str, fmt: str | None) -> None:
        """Wait for channel to hold a formatted time.now() and check it against the
        wall clock. A None format expects ISO 8601, else the output of PYTHON_FORMAT.
        """
        pattern = ISO_8601_UTC if fmt is None else PYTHON_FORMAT_OUTPUT
        self._wait_for_condition(
            channel,
            lambda v: isinstance(v, str) and pattern.match(v) is not None,
            f"to match {pattern.pattern}",
        )
        text = self.read_tlm(channel, "")
        if fmt is None:
            stamp = datetime.fromisoformat(text)
        else:
            stamp = datetime.strptime(text, fmt).replace(tzinfo=UTC)
        drift = abs(datetime.now(UTC) - stamp)
        self.log(
            f"[{label}] time.format wrote {text!r}, "
            f"drift {drift.total_seconds():.3f}s (max {MAX_FORMAT_DRIFT.seconds}s)"
        )
        if drift > MAX_FORMAT_DRIFT:
            self.fail(
                f"[{label}] time.format wrote {text!r}, "
                f"{drift.total_seconds():.3f}s from the wall clock"
            )

    def _test_format(self) -> None:
        self.log("=== time.format() [WASM] ===")
        self.writer.write("time_format_fn_trigger", 1.0)
        self._assert_format("time_format_fn_out", "function", None)

    def _test_format_flow(self) -> None:
        self.log("=== time.format() [Flow] ===")
        self.writer.write("time_format_flow_trigger", 1.0)
        self._assert_format("time_format_flow_out", "flow", None)

    def _test_format_now_lit(self) -> None:
        self.log("=== time.format() [time.now() with a format] ===")
        self.writer.write("now_fmt_trigger", 1.0)
        self._assert_format("now_fmt_fn_out", "function", PYTHON_FORMAT)
        self._assert_format("now_fmt_flow_out", "flow", PYTHON_FORMAT)

    def _check_python(self, ns: int, fmt: str, expected: str) -> None:
        """Fail when Python renders format at ns differently from the table."""
        rendered = python_strftime(ns, fmt)
        if rendered is not None and rendered != expected:
            self.fail(
                f"Python renders {fmt!r} as {rendered!r}, table says {expected!r}"
            )

    def _test_format_fixed(self) -> None:
        self.log("=== time.format() [fixed stamps, literal formats] ===")
        for ns, iso, with_format in FORMAT_CASES:
            self._check_python(ns, PYTHON_FORMAT, with_format)
            self.writer.write("time_format_ts_in", ns)
            self.log(f"Expecting {iso!r} and {with_format!r} for {ns} ns")
            self.wait_for_eq("time_format_fixed_fn_out", iso)
            self.wait_for_eq("time_format_fixed_flow_out", iso)
            self.wait_for_eq("time_format_lit_fn_out", with_format)
            self.wait_for_eq("time_format_lit_flow_out", with_format)
            self.wait_for_eq("time_fstr_out", f"at {iso[11:19]}")

    def _check_format(self, ns: int, fmt: str, expected: str) -> None:
        """Write ns and format, then expect expected from both contexts."""
        self._check_python(ns, fmt, expected)
        self.writer.write("time_format_ts_in", ns)
        self.writer.write("time_format_in", fmt)
        self.log(f"Expecting {expected!r} for {fmt!r} at {ns} ns")
        self.wait_for_eq("time_format_chan_fn_out", expected)
        self.wait_for_eq("time_format_chan_flow_out", expected)

    def _test_format_codes(self) -> None:
        self.log("=== time.format() [every format code, format from a channel] ===")
        for fmt, expected in CODE_CASES:
            self._check_format(STAMP, fmt, expected)

    def _test_format_edges(self) -> None:
        self.log("=== time.format() [calendar edges] ===")
        for ns, fmt, expected in EDGE_CASES:
            self._check_format(ns, fmt, expected)

    def _test_interval_rate(self) -> None:
        self.log("=== time.interval ===")
        baselines = {ch: self.read_tlm(ch, 0) for ch, _ in INTERVAL_CHANNELS}
        sy.sleep(1.0)
        for ch, label in INTERVAL_CHANNELS:
            delta = self.read_tlm(ch, 0) - baselines[ch]
            self.log(f"[{label}] {ch} fired {delta} times in 1s (expected ~10)")
            if delta < 5 or delta > 15:
                self.fail(
                    f"[{label}] interval fired {delta} times in 1s, "
                    f"expected 5-15 (nominal 10)"
                )

    def _assert_wait_duration(self, duration: sy.TimeSpan, label: str) -> None:
        wait_s = duration / sy.TimeSpan.SECOND
        self.log(f"[{label}] Wait duration: {wait_s:.2f}s (expected ~3.0s)")
        if wait_s > MAX_WAIT_DURATION:
            self.fail(
                f"[{label}] wait{{3s}} took {wait_s:.2f}s, "
                f"exceeding maximum of {MAX_WAIT_DURATION}s."
            )
        if wait_s < MIN_WAIT_DURATION:
            self.fail(
                f"[{label}] wait{{3s}} took {wait_s:.2f}s, "
                f"below minimum of {MIN_WAIT_DURATION}s."
            )
        self.log(
            f"[{label}] Wait timing verified: {wait_s:.2f}s is within "
            f"[{MIN_WAIT_DURATION}, {MAX_WAIT_DURATION}]s"
        )

    def _test_wait_timing(self) -> None:
        self.log("=== time.wait ===")
        for start_cmd, _, _ in WAIT_CASES:
            self.writer.write(start_cmd, 1)
        measurements: list[tuple[sy.Timer, str, str]] = []
        for _, toggle, label in WAIT_CASES:
            self.wait_for_eq(toggle, 1, timeout=5 * sy.TimeSpan.SECOND)
            self.log(f"[{label}] {toggle}=1 observed, measuring 3s wait")
            measurements.append((sy.Timer(), toggle, label))
        for timer, toggle, label in measurements:
            self.wait_for_eq(toggle, 0, timeout=8 * sy.TimeSpan.SECOND)
            self._assert_wait_duration(timer.elapsed(), label)

    def verify_sequence_execution(self) -> None:
        self._test_now()
        self._test_now_flow()
        self._test_format()
        self._test_format_flow()
        self._test_format_now_lit()
        self._test_format_fixed()
        self._test_format_codes()
        self._test_format_edges()
        self._test_interval_rate()
        self._test_wait_timing()
