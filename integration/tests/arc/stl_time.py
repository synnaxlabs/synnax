#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import synnax as sy
from framework.utils import create_virtual_channel
from tests.arc.arc_case import ArcConsoleCase

ARC_STL_TIME_SOURCE = """
authority 200

// ──────────────────────────── time.now ───────────────────────────────

func write_now() {
    time_now_out = time.now()
}
time_trigger -> write_now{}

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
        1 -> toggle_cmd,
        wait{3s} => stage2,
    }
    stage stage2 {
        0 -> toggle_cmd,
    }
}
start_wait_cmd => main_standalone

// module-qualified
sequence main_module {
    stage stage1 {
        1 -> toggle_cmd_mod,
        time.wait{3s} => stage2,
    }
    stage stage2 {
        0 -> toggle_cmd_mod,
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


class StlTime(ArcConsoleCase):
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
        "time_now_out",
        "interval_count",
        "interval_count_mod",
        "toggle_cmd",
        "toggle_cmd_mod",
    ]

    def setup(self) -> None:
        create_virtual_channel(self.client, "time_trigger", sy.DataType.FLOAT64)
        create_virtual_channel(self.client, "time_now_out", sy.DataType.INT64)
        create_virtual_channel(self.client, "interval_count", sy.DataType.INT64)
        create_virtual_channel(self.client, "interval_count_mod", sy.DataType.INT64)
        create_virtual_channel(self.client, "start_wait_cmd", sy.DataType.UINT8)
        create_virtual_channel(self.client, "start_wait_mod_cmd", sy.DataType.UINT8)
        for name in ("toggle_cmd", "toggle_cmd_mod"):
            idx = self.client.channels.create(
                name=f"{name}_time",
                is_index=True,
                data_type=sy.DataType.TIMESTAMP,
                retrieve_if_name_exists=True,
            )
            self.client.channels.create(
                name=name,
                data_type=sy.DataType.UINT8,
                index=idx.key,
                retrieve_if_name_exists=True,
            )
        super().setup()

    def _test_now(self) -> None:
        self.log("=== time.now() ===")
        self.writer.write("time_trigger", 1.0)
        self.log(f"Expecting time_now_out > {JAN_2020_NANOS} (Jan 1, 2020 nanos)")
        self.wait_for_gt("time_now_out", JAN_2020_NANOS, is_virtual=True)
        self.log("time.now() returned a valid timestamp")

    def _check_interval_rate(self, channel: str, label: str) -> None:
        baseline = self.read_tlm(channel, 0)
        self.log(f"[{label}] Baseline {channel} = {baseline}")
        sy.sleep(1.0)
        current = self.read_tlm(channel, 0)
        delta = current - baseline
        self.log(
            f"[{label}] After 1s: {channel} = {current}, "
            f"delta = {delta} (expected ~10 at 100ms period)"
        )
        if delta < 5 or delta > 15:
            self.fail(
                f"[{label}] interval fired {delta} times in 1s, "
                f"expected 5-15 (nominal 10)"
            )

    def _test_interval_rate(self) -> None:
        self.log("=== time.interval ===")
        self._check_interval_rate("interval_count", "standalone")
        self._check_interval_rate("interval_count_mod", "module")

    def _check_wait_timing(self, channel: str, label: str) -> None:
        self.log(f"[{label}] Waiting for {channel}=1 (stage1 entered)...")
        self.wait_for_eq(channel, 1, timeout=5 * sy.TimeSpan.SECOND)
        timer = sy.Timer()
        self.log(f"[{label}] {channel}=1 observed, starting 3s wait measurement")

        self.log(f"[{label}] Waiting for {channel}=0 (stage2 entered)...")
        self.wait_for_eq(channel, 0, timeout=8 * sy.TimeSpan.SECOND)

        wait_duration = timer.elapsed() / sy.TimeSpan.SECOND
        self.log(f"[{label}] Wait duration: {wait_duration:.2f}s (expected ~3.0s)")

        if wait_duration > MAX_WAIT_DURATION:
            self.fail(
                f"[{label}] wait{{3s}} took {wait_duration:.2f}s, "
                f"exceeding maximum of {MAX_WAIT_DURATION}s."
            )
            return

        if wait_duration < MIN_WAIT_DURATION:
            self.fail(
                f"[{label}] wait{{3s}} took {wait_duration:.2f}s, "
                f"below minimum of {MIN_WAIT_DURATION}s."
            )
            return

        self.log(
            f"[{label}] Wait timing verified: {wait_duration:.2f}s is within "
            f"[{MIN_WAIT_DURATION}, {MAX_WAIT_DURATION}]s"
        )

    def _test_wait_timing(self) -> None:
        self.log("=== time.wait ===")
        self.writer.write("start_wait_cmd", 1)
        self._check_wait_timing("toggle_cmd", "standalone")
        self.writer.write("start_wait_mod_cmd", 1)
        self._check_wait_timing("toggle_cmd_mod", "module")

    def verify_sequence_execution(self) -> None:
        self._test_now()
        self._test_interval_rate()
        self._test_wait_timing()
