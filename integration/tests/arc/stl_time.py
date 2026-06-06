#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

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
time_trigger -> write_now{}

// time.now{} as a flow node: triggered by any upstream, outputs timestamp.
time_now_flow_ts_trigger -> time.now{} -> time_now_flow_ts_out
time_now_flow_int_trigger -> time.now{} -> time_now_flow_int_out

// ─────────────────────────── time.interval ──────────────────────────
// interval is inherently time-triggered (that's the point of the function).
// We use time.interval{} (qualified syntax) to verify module syntax works.
// standalone
func count_intervals() {
    interval_count = interval_count + 1
}
interval{period=100ms} -> count_intervals{}
// module-qualified
func count_intervals_mod() {
    interval_count_mod = interval_count_mod + 1
}
time.interval{period=100ms} -> count_intervals_mod{}
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
        wait{3s} => stage2
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
        time.wait{3s} => stage2
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
    """Test time module with qualified syntax: time.now(), time.interval{},
    time.wait{}.

    time.now() is verified by checking the returned timestamp is a valid
    nanosecond value. time.interval{} is verified by measuring its firing
    rate over 1 second. time.wait{} is a regression test migrated from
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
                ("interval_count", sy.DataType.INT64),
                ("interval_count_mod", sy.DataType.INT64),
                ("start_wait_cmd", sy.DataType.UINT8),
                ("start_wait_mod_cmd", sy.DataType.UINT8),
            ],
        )
        for name in ("toggle_cmd", "toggle_cmd_mod"):
            create_indexed_pair(self.client, name, sy.DataType.UINT8)
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
        self.log("=== time.now{} [Flow] ===")
        self.writer.write("time_now_flow_ts_trigger", 1.0)
        self.wait_for_gt("time_now_flow_ts_out", 0)
        ts = self.read_tlm("time_now_flow_ts_out", 0)
        now = int(sy.TimeStamp.now())
        drift = abs(ts - now)
        max_drift = 500 * int(sy.TimeSpan.MILLISECOND)
        self.log(
            f"time.now{{}} returned {ts}, current time {now}, "
            f"drift {drift / 1e6:.1f}ms (max 500ms)"
        )
        if drift > max_drift:
            self.fail(f"time.now{{}} drift {drift / 1e6:.1f}ms exceeds 500ms tolerance")
        self.log(
            f"Expecting time_now_flow_int_out > {JAN_2020_NANOS} (Jan 1, 2020 nanos)"
        )
        self.writer.write("time_now_flow_int_trigger", 1.0)
        self.wait_for_gt("time_now_flow_int_out", JAN_2020_NANOS)

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
        self._test_interval_rate()
        self._test_wait_timing()
