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

ARC_STAT_SOURCE = """
stat_in -> avg{} -> stat_avg_out
stat_in -> min{} -> stat_min_out
stat_in -> max{} -> stat_max_out
stat_count_in -> avg{count=5} -> stat_avg_count_out
stat_count_in -> min{count=5} -> stat_min_count_out
stat_count_in -> max{count=5} -> stat_max_count_out
stat_dur_in -> avg{duration=500ms} -> stat_avg_dur_out
stat_dur_in -> min{duration=500ms} -> stat_min_dur_out
stat_dur_in -> max{duration=500ms} -> stat_max_dur_out
stat_neg_in -> avg{} -> stat_neg_avg_out
stat_neg_in -> min{} -> stat_neg_min_out
stat_neg_in -> max{} -> stat_neg_max_out
stat_edge_in -> avg{} -> stat_edge_avg_out
stat_edge_in -> min{} -> stat_edge_min_out
stat_edge_in -> max{} -> stat_edge_max_out
stat_mono_in -> avg{} -> stat_mono_avg_out
stat_mono_in -> min{} -> stat_mono_min_out
stat_mono_in -> max{} -> stat_mono_max_out
stat_deriv_in -> derivative{} -> stat_deriv_out
"""

VIRTUAL_INPUTS = [
    "stat_in",
    "stat_count_in",
    "stat_dur_in",
    "stat_neg_in",
    "stat_edge_in",
    "stat_mono_in",
    "stat_deriv_in",
]

INDEXED_OUTPUTS = [
    "stat_avg_out",
    "stat_min_out",
    "stat_max_out",
    "stat_avg_count_out",
    "stat_min_count_out",
    "stat_max_count_out",
    "stat_avg_dur_out",
    "stat_min_dur_out",
    "stat_max_dur_out",
    "stat_neg_avg_out",
    "stat_neg_min_out",
    "stat_neg_max_out",
    "stat_edge_avg_out",
    "stat_edge_min_out",
    "stat_edge_max_out",
    "stat_mono_avg_out",
    "stat_mono_min_out",
    "stat_mono_max_out",
    "stat_deriv_out",
]

ALL_CHANNELS = VIRTUAL_INPUTS + INDEXED_OUTPUTS


class Stat(ArcConsoleCase):
    """Test Arc STL stat functions: avg, min, max, derivative.
    Writes controlled values to virtual input channels and verifies
    each stat pipeline produces the expected output."""

    arc_source = ARC_STAT_SOURCE
    arc_name_prefix = "ArcStat"
    start_cmd_channel = "start_stat_cmd"
    subscribe_channels = ALL_CHANNELS

    def setup(self) -> None:
        for name in VIRTUAL_INPUTS:
            create_virtual_channel(self.client, name, sy.DataType.FLOAT64)
        for name in INDEXED_OUTPUTS:
            idx = self.client.channels.create(
                name=f"{name}_time",
                is_index=True,
                data_type=sy.DataType.TIMESTAMP,
                retrieve_if_name_exists=True,
            )
            self.client.channels.create(
                name=name,
                data_type=sy.DataType.FLOAT64,
                index=idx.key,
                retrieve_if_name_exists=True,
            )
        super().setup()

    def _write(self, channel: str, value: float) -> None:
        self.writer.write(channel, value)

    def _write_many(self, channel: str, values: list[float]) -> None:
        """Write multiple values through a single writer session."""
        for val in values:
            self.writer.write(channel, val)

    def _write_spaced(
        self,
        channel: str,
        values: list[float],
        dt_ms: int = 20,
    ) -> None:
        """Write values to a virtual channel with wall-clock spacing.
        Sleeps dt_ms between each write so the Arc runtime assigns
        predictable timestamps for derivative computation.
        """
        for i, val in enumerate(values):
            self.writer.write(channel, val)
            if i < len(values) - 1:
                sy.sleep(dt_ms / 1000.0)

    def verify_sequence_execution(self) -> None:
        self._test_basic_stats()
        self._test_count_window()
        self._test_count_window_all_negative()
        self._test_duration_window()
        self._test_duration_window_reset()
        self._test_negative_values()
        self._test_large_input()
        self._test_all_identical()
        self._test_zeros()
        self._test_large_magnitudes()
        self._test_derivative_constant()
        self._test_derivative_alternating()
        self._test_monotonic_increasing()

    def _test_basic_stats(self) -> None:
        self.log("Basic stats: [10, 20, 30]")
        self._write_many("stat_in", [10.0, 20.0, 30.0])

        self.log("[avg] Expecting 20.0")
        self.wait_for_eq("stat_avg_out", 20.0)

        self.log("[min] Expecting 10.0")
        self.wait_for_eq("stat_min_out", 10.0)

        self.log("[max] Expecting 30.0")
        self.wait_for_eq("stat_max_out", 30.0)

        # 2 samples, diff=1.0, 20ms spacing: rate ≈ 1.0/0.02 = 50
        self._write_spaced("stat_deriv_in", [0.0, 1.0])
        self.log("[deriv] Expecting ≈ 50")
        self.wait_for_near("stat_deriv_out", 50.0, tolerance=25.0)

    def _test_count_window(self) -> None:
        self.log("Count window (count=5): 15 samples, 3 windows")

        # Window 1: [10, 20, 30, 40, 50] -> avg=30, min=10, max=50
        # Window 2: [100, 200, 300, 400, 500] -> avg=300, min=100, max=500
        # Window 3: [1, 2, 3, 4, 5] -> avg=3, min=1, max=5
        self._write_many(
            "stat_count_in",
            [
                10.0,
                20.0,
                30.0,
                40.0,
                50.0,
                100.0,
                200.0,
                300.0,
                400.0,
                500.0,
                1.0,
                2.0,
                3.0,
                4.0,
                5.0,
            ],
        )

        # Last window's values: [1, 2, 3, 4, 5]
        self.wait_for_eq("stat_avg_count_out", 3.0)
        self.wait_for_eq("stat_min_count_out", 1.0)
        self.wait_for_eq("stat_max_count_out", 5.0)

    def _test_count_window_all_negative(self) -> None:
        self.log("Count window: all negative [-10,-20,-30,-40,-50]")
        self._write_many(
            "stat_count_in",
            [-10.0, -20.0, -30.0, -40.0, -50.0],
        )

        self.wait_for_eq("stat_avg_count_out", -30.0)
        self.wait_for_eq("stat_min_count_out", -50.0)
        self.wait_for_eq("stat_max_count_out", -10.0)

    def _test_duration_window(self) -> None:
        self.log("Duration window (500ms): batch, wait 600ms, batch")
        # Batch 1 at t≈0
        self._write_many("stat_dur_in", [1.0, 2.0, 3.0, 4.0, 5.0])
        self.wait_for_eq("stat_max_dur_out", 5.0)

        # Wait 600ms so the 500ms window expires
        sy.sleep(0.6)

        # Batch 2 — window has reset, only these samples count
        self._write_many("stat_dur_in", [100.0, 200.0, 300.0])

        self.log("[dur_max] Expecting 300.0")
        self.wait_for_eq("stat_max_dur_out", 300.0)

        self.log("[dur_min] Expecting 100.0 (batch 1 expired)")
        self.wait_for_eq("stat_min_dur_out", 100.0)

    def _test_duration_window_reset(self) -> None:
        self.log("Duration window reset: another 600ms gap")
        sy.sleep(0.6)

        self._write_many("stat_dur_in", [900.0, 910.0, 920.0])

        self.wait_for_eq("stat_max_dur_out", 920.0)
        self.wait_for_eq("stat_min_dur_out", 900.0)

    def _test_negative_values(self) -> None:
        self.log("Negative values: [-50, -10, -200, 100, -300, 75]")
        self._write_many(
            "stat_neg_in",
            [-50.0, -10.0, -200.0, 100.0, -300.0, 75.0],
        )

        # avg = -385/6 ≈ -64.17
        self.log("[neg_avg] Expecting ≈ -64.17")
        self.wait_for_near("stat_neg_avg_out", -64.17, tolerance=0.01)

        self.log("[neg_min] Expecting -300.0")
        self.wait_for_eq("stat_neg_min_out", -300.0)

        self.log("[neg_max] Expecting 100.0")
        self.wait_for_eq("stat_neg_max_out", 100.0)

        # 2 samples, diff=2.0, 20ms: rate ≈ 2.0/0.02 = 100 (positive)
        self._write_spaced("stat_deriv_in", [-1.0, 1.0])
        self.log("[neg_deriv] Expecting ≈ 100")
        self.wait_for_near("stat_deriv_out", 100.0, tolerance=50.0)

    def _test_large_input(self) -> None:
        values = [
            5.0,
            -3.0,
            12.0,
            -8.0,
            20.0,
            -15.0,
            7.0,
            -1.0,
            25.0,
            -20.0,
            30.0,
            -25.0,
            3.0,
            18.0,
            -12.0,
            50.0,
            -40.0,
            35.0,
            -30.0,
            100.0,
        ]
        self.log(f"Large input: {len(values)} samples")
        self._write_many("stat_neg_in", values)

        # Running stats accumulate on top of _test_negative_values.
        # Global min = -300 (from previous), max = 100 (tied)
        self.log("[large_min] Expecting -300.0 (global min)")
        self.wait_for_eq("stat_neg_min_out", -300.0)

        self.log("[large_max] Expecting 100.0 (global max)")
        self.wait_for_eq("stat_neg_max_out", 100.0)

    def _test_all_identical(self) -> None:
        self.log("All identical: [42, 42, 42, 42, 42]")
        self._write_many("stat_edge_in", [42.0, 42.0, 42.0, 42.0, 42.0])

        self.wait_for_eq("stat_edge_avg_out", 42.0)
        self.wait_for_eq("stat_edge_min_out", 42.0)
        self.wait_for_eq("stat_edge_max_out", 42.0)

    def _test_zeros(self) -> None:
        self.log("Zeros: writing 5 zeros")
        self._write_many("stat_edge_in", [0.0, 0.0, 0.0, 0.0, 0.0])

        self.wait_for_eq("stat_edge_min_out", 0.0)
        self.wait_for_eq("stat_edge_max_out", 42.0)

    def _test_large_magnitudes(self) -> None:
        self.log("Large magnitudes: [1e12, 2e12, 3e12]")
        self._write_many("stat_edge_in", [1e12, 2e12, 3e12])

        self.wait_for_eq("stat_edge_max_out", 3e12)

    def _test_derivative_constant(self) -> None:
        self.log("Derivative constant: 2 x 5.0 at 20ms")
        self._write_spaced("stat_deriv_in", [5.0, 5.0])

        self.log("[edge_deriv] Expecting rate = 0")
        self.wait_for_eq("stat_deriv_out", 0.0)

    def _test_derivative_alternating(self) -> None:
        # Positive rate: diff=1.0, 20ms: rate ≈ 1.0/0.02 = 50
        self.log("Derivative positive: [0, 1.0] at 20ms")
        self._write_spaced("stat_deriv_in", [0.0, 1.0])

        self.log("[edge_deriv] Expecting ≈ 50")
        self.wait_for_near("stat_deriv_out", 50.0, tolerance=25.0)

        # Negative rate: diff=-1.0, 20ms: rate ≈ -1.0/0.02 = -50
        self.log("Derivative negative: [1.0, 0] at 20ms")
        self._write_spaced("stat_deriv_in", [1.0, 0.0])

        self.log("[edge_deriv] Expecting ≈ -50")
        self.wait_for_near("stat_deriv_out", -50.0, tolerance=25.0)

    def _test_monotonic_increasing(self) -> None:
        self.log("Monotonic increasing: 1..50")
        self._write_many("stat_mono_in", [float(i) for i in range(1, 51)])

        # avg(1..50) = 25.5
        self.wait_for_eq("stat_mono_avg_out", 25.5)
        self.wait_for_eq("stat_mono_min_out", 1.0)
        self.wait_for_eq("stat_mono_max_out", 50.0)
