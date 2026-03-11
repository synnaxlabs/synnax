#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""
Arc Statement Ordering Integration Test

Verifies that when multiple stage transition conditions are true simultaneously,
the first-written condition takes priority.

Phase 1: cf_temp_a=200, cf_temp_b=400
    Only the second condition is true (cf_temp_b > 300), so we loop on/pause.

Phase 2: set cf_temp_a=400
    Both conditions become true. The first condition (=> off) should win.
"""

import threading

import synnax as sy
from examples.simulators import PressSimDAQ

from tests.arc.arc_case import ArcConsoleCase

ARC_CHECKFAIL_SOURCE = """
func count{c_chan chan u8}() {
    n u8 $= 0
    n = n + 1
    c_chan = n
}

func noop{}(input u8) u8 {
    return input
}
cf_start_cmd => main

sequence main {
    stage on {
        "on" -> cf_stage_str,
        count{c_chan = cf_count_on},
        0 -> cf_sim_stage,
        1 -> cf_heater_cmd,
        // This needs fixing, but not now. Keep here for visibility.
        interval{period=1s} -> (cf_temp_a > 290 and cf_temp_b > 290) -> noop{} -> noop{} -> noop{} => off,
        interval{period=1s} -> cf_temp_b > 300 => pause,
    }
    stage pause {
        "pause" -> cf_stage_str,
        count{c_chan = cf_count_pause},
        2 -> cf_sim_stage,
        0 -> cf_heater_cmd,
        wait{duration=1s} => on,
    }
    stage off {
        "off" -> cf_stage_str,
        3 -> cf_sim_stage,
        0 -> cf_heater_cmd,
        cf_start_cmd => on,
    }
}
"""


class ChannelCollector:
    """Accumulates all streamed values for a set of channels.

    Usage:
        with ChannelCollector(client, ["ch_a", "ch_b"]) as data:
            # data["ch_a"] and data["ch_b"] grow as frames arrive
        # assert on data after exiting
    """

    def __init__(self, client: sy.Synnax, channels: list[str]) -> None:
        self._client = client
        self._channels = channels
        self._stop = threading.Event()
        self.data: dict[str, list[int | float | str]] = {ch: [] for ch in channels}

    def __enter__(self) -> dict[str, list[int | float | str]]:
        self._thread = threading.Thread(target=self._run, daemon=True)
        self._thread.start()
        return self.data

    def __exit__(self, *_: object) -> None:
        self._stop.set()
        self._thread.join(timeout=3)

    def _run(self) -> None:
        with self._client.open_streamer(self._channels) as s:
            while not self._stop.is_set():
                frame = s.read(timeout=sy.TimeSpan.SECOND)
                if frame is None:
                    continue
                for ch in self._channels:
                    if ch in frame:
                        self.data[ch].extend(frame[ch])


class ArcCheckfailOrdering(ArcConsoleCase):
    """Test that first-written transition takes priority and skips later statements."""

    arc_source = ARC_CHECKFAIL_SOURCE
    arc_name_prefix = "ArcCheckfailOrdering"
    start_cmd_channel = "end_test_cmd"  # Wrong on purpose so we can trigger manually.
    end_cmd_channel = "end_test_cmd"
    subscribe_channels = [
        "cf_stage_str",
        "cf_sim_stage",
        "cf_heater_cmd",
        "cf_temp_a",
        "cf_temp_b",
        "cf_count_on",
        "cf_count_pause",
        "end_test_cmd",
    ]
    sim_daq_class = PressSimDAQ

    def setup(self) -> None:
        client = self.client

        for name in ["cf_stage_str", "cf_sim_stage"]:
            client.channels.create(
                name=name,
                data_type=(
                    sy.DataType.STRING if name == "cf_stage_str" else sy.DataType.UINT8
                ),
                virtual=True,
                retrieve_if_name_exists=True,
            )
        client.channels.create(
            name="cf_start_cmd",
            data_type=sy.DataType.UINT8,
            virtual=True,
            retrieve_if_name_exists=True,
        )
        for time_ch, data_ch in [
            ("cf_count_on_time", "cf_count_on"),
            ("cf_count_pause_time", "cf_count_pause"),
        ]:
            idx = client.channels.create(
                name=time_ch, is_index=True, retrieve_if_name_exists=True
            )
            client.channels.create(
                name=data_ch,
                index=idx.key,
                data_type=sy.DataType.UINT8,
                retrieve_if_name_exists=True,
            )

        client.channels.create(
            name="cf_heater_cmd",
            data_type=sy.DataType.UINT8,
            virtual=True,
            retrieve_if_name_exists=True,
        )

        self.cf_sensor_time = client.channels.create(
            name="cf_sensor_time",
            is_index=True,
            retrieve_if_name_exists=True,
        )
        client.channels.create(
            name="cf_temp_a",
            index=self.cf_sensor_time.key,
            data_type=sy.DataType.FLOAT32,
            retrieve_if_name_exists=True,
        )
        client.channels.create(
            name="cf_temp_b",
            index=self.cf_sensor_time.key,
            data_type=sy.DataType.FLOAT32,
            retrieve_if_name_exists=True,
        )

        super().setup()

    def verify_sequence_execution(self) -> None:
        stream_channels = [
            "cf_stage_str",
            "cf_sim_stage",
            "cf_heater_cmd",
            "cf_count_on_time",
            "cf_count_pause_time",
        ]
        with ChannelCollector(self.client, stream_channels) as collected:
            with self.client.open_writer(
                start=sy.TimeStamp.now(),
                channels=["cf_sensor_time", "cf_temp_a", "cf_temp_b"],
                name="Checkfail Sensor Writer",
            ) as self._sensor_writer:
                self._cf_temp_a = 200.0
                self._cf_temp_b = 400.0
                self._write_sensors()

                with self.client.open_writer(sy.TimeStamp.now(), "cf_start_cmd") as w:
                    w.write("cf_start_cmd", 1)

                self._verify_on_pause_loop()
                self._verify_off_transition()

        self._assert_loop_writes(collected)

    def _write_sensors(self) -> None:
        now = sy.TimeStamp.now()
        self._sensor_writer.write(
            {
                "cf_sensor_time": now,
                "cf_temp_a": self._cf_temp_a,
                "cf_temp_b": self._cf_temp_b,
            }
        )

    def _verify_on_pause_loop(self) -> None:
        self.log("Phase 1: Verifying on/pause loop")
        for i in range(1, 4):
            self.wait_for_eq("cf_count_on", float(i), is_virtual=False)
            self.wait_for_eq("cf_count_pause", float(i), is_virtual=False)
            self._write_sensors()
        self.log("Phase 1 complete")

    def _verify_off_transition(self) -> None:
        self.log("Phase 2: Setting cf_temp_a=400")
        self._cf_temp_a = 400.0
        self._write_sensors()
        self.wait_for_eq("cf_stage_str", "off", is_virtual=True, timeout=10.0)
        self.wait_for_eq("cf_sim_stage", 3, is_virtual=True, timeout=5.0)
        self.log("Phase 2 complete: first transition won, later statements skipped")

    def _assert_loop_writes(
        self, collected: dict[str, list[int | float | str]]
    ) -> None:
        """Assert each channel produced the exact expected sequence."""
        for ch in ("cf_count_on_time", "cf_count_pause_time"):
            times = [int(t) for t in collected[ch]]
            deltas_s = [(times[i + 1] - times[i]) / 1e9 for i in range(len(times) - 1)]
            self.log(f"{ch} deltas (s): {[f'{d:.3f}' for d in deltas_s]}")
            for d in deltas_s:
                assert 1.0 <= d <= 1.005, f"{ch}: delta {d:.3f}s out of [1.000, 1.005]"

        expected: dict[str, list[int | float | str]] = {
            "cf_stage_str": ["on", "pause", "on", "pause", "on", "pause", "on", "off"],
            "cf_sim_stage": [0, 2, 0, 2, 0, 2, 0, 3],
            "cf_heater_cmd": [1, 0, 1, 0, 1, 0, 1, 0],
        }
        for ch, seq in expected.items():
            cast = str if ch == "cf_stage_str" else int
            actual = [cast(v) for v in collected[ch]]
            assert actual[: len(seq)] == seq, f"{ch}: expected {seq} — got {actual}"
