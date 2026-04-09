#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""
Arc Short Circuit Integration Test

Verifies two properties of the Arc scheduler:

1. Transition priority by declaration order: when multiple stage transition
   conditions are simultaneously true, the first-written transition wins and
   subsequent statements in that stage pass are not executed. Statements after
   a winning transition (including channel writes) must be skipped.

2. Loop timing: the on/pause cycle runs on a 1-second interval. Each loop
   iteration must complete within a tight timing window to confirm the scheduler
   is not drifting or accumulating delay across stage transitions.

Phase 1: ss_temp_a=200, ss_temp_b=400
    Only the second condition (ss_temp_b > 300) is true, so the sequence loops
    on => pause => on at 1-second intervals.

Phase 2: ss_temp_a=400
    Both conditions become true simultaneously. The first-written transition
    (=> off) must win. The second transition (=> pause) must never execute.
"""

import threading

import synnax as sy
from framework.utils import create_virtual_channel
from tests.arc.arc_case import ArcConsoleCase

SHORT_CIRCUIT_SOURCE = """
func count{c_chan chan u8}() {
    n u8 $= 0
    n = n + 1
    c_chan = n
}

func noop{}(input u8) u8 {
    return input
}

ss_start_cmd => main

sequence main {
    stage on {
        "on" -> ss_stage_str,
        count{c_chan = ss_count_on},
        0 -> ss_sim_stage,
        1 -> ss_heater_cmd,
        // Priority is declaration order, not statement size.
        interval{1s} -> (ss_temp_a > 290 and ss_temp_b > 290) -> noop{} -> noop{} -> noop{} => off,
        interval{1s} -> ss_temp_b > 300 => pause,
    }
    stage pause {
        "pause" -> ss_stage_str,
        count{c_chan = ss_count_pause},
        2 -> ss_sim_stage,
        0 -> ss_heater_cmd,
        wait{1s} => on,
    }
    stage off {
        "off" -> ss_stage_str,
        3 -> ss_sim_stage,
        0 -> ss_heater_cmd,
        ss_start_cmd => on,
    }
}
"""


class ChannelCollector:
    """Accumulates streamed values for a set of channels."""

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


class ShortCircuit(ArcConsoleCase):
    """Test that first-written transition short-circuits later statements."""

    arc_source = SHORT_CIRCUIT_SOURCE
    arc_name_prefix = "ArcShortCircuit"
    start_cmd_channel = "end_test_cmd"  # Wrong on purpose so we can trigger manually.
    subscribe_channels = [
        "ss_stage_str",
        "ss_sim_stage",
        "ss_heater_cmd",
        "ss_temp_a",
        "ss_temp_b",
        "ss_count_on",
        "ss_count_pause",
    ]

    def setup(self) -> None:
        client = self.client

        create_virtual_channel(client, "ss_stage_str", sy.DataType.STRING)
        create_virtual_channel(client, "ss_sim_stage", sy.DataType.UINT8)
        create_virtual_channel(client, "ss_start_cmd", sy.DataType.UINT8)

        for time_ch, data_ch in [
            ("ss_count_on_time", "ss_count_on"),
            ("ss_count_pause_time", "ss_count_pause"),
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

        create_virtual_channel(client, "ss_heater_cmd", sy.DataType.UINT8)

        self.ss_sensor_time = client.channels.create(
            name="ss_sensor_time",
            is_index=True,
            retrieve_if_name_exists=True,
        )

        client.channels.create(
            name="ss_temp_a",
            index=self.ss_sensor_time.key,
            data_type=sy.DataType.FLOAT32,
            retrieve_if_name_exists=True,
        )
        client.channels.create(
            name="ss_temp_b",
            index=self.ss_sensor_time.key,
            data_type=sy.DataType.FLOAT32,
            retrieve_if_name_exists=True,
        )

        super().setup()

    def verify_sequence_execution(self) -> None:
        stream_channels = [
            "ss_stage_str",
            "ss_sim_stage",
            "ss_heater_cmd",
            "ss_count_on_time",
            "ss_count_pause_time",
        ]
        with ChannelCollector(self.client, stream_channels) as collected:
            with self.client.open_writer(
                start=sy.TimeStamp.now(),
                channels=["ss_sensor_time", "ss_temp_a", "ss_temp_b"],
                name="Short Circuit Sensor Writer",
            ) as self._sensor_writer:
                self._ss_temp_a = 200.0
                self._ss_temp_b = 400.0
                self._write_sensors()

                self.writer.write("ss_start_cmd", 1)

                self._verify_on_pause_loop()
                self._verify_off_transition()

        self._assert_loop_writes(collected)

    def _write_sensors(self) -> None:
        now = sy.TimeStamp.now()
        self._sensor_writer.write(
            {
                "ss_sensor_time": now,
                "ss_temp_a": self._ss_temp_a,
                "ss_temp_b": self._ss_temp_b,
            }
        )

    def _verify_on_pause_loop(self) -> None:
        self.log("Phase 1: Verifying on/pause loop")
        for i in range(1, 4):
            self.wait_for_eq("ss_count_on", i, is_virtual=False)
            self.wait_for_eq("ss_count_pause", i, is_virtual=False)
            self._write_sensors()
        # Wait for the 4th on stage to begin before transitioning to phase 2,
        # so the collector captures the full on/pause/on/pause/on/pause/on sequence.
        self.wait_for_eq("ss_count_on", 4, is_virtual=False)
        self.log("Phase 1 complete")

    def _verify_off_transition(self) -> None:
        self.log("Phase 2: Setting ss_temp_a=400")
        self._ss_temp_a = 400.0
        self._write_sensors()
        self.wait_for_eq("ss_stage_str", "off", is_virtual=True, timeout=10.0)
        self.wait_for_eq("ss_sim_stage", 3, is_virtual=True, timeout=5.0)
        self.log("Phase 2 complete: first transition won, later statements skipped")

    def _assert_loop_writes(
        self, collected: dict[str, list[int | float | str]]
    ) -> None:
        for ch in ("ss_count_on_time", "ss_count_pause_time"):
            times = [int(t) for t in collected[ch]]
            deltas_s = [(times[i + 1] - times[i]) / 1e9 for i in range(len(times) - 1)]
            self.log(f"{ch} deltas (s): {[f'{d:.3f}' for d in deltas_s]}")
            for d in deltas_s:
                assert 0.950 <= d <= 1.050, (
                    f"{ch}: delta {d:.3f}s out of [0.950, 1.050]"
                )

        expected: dict[str, list[int | float | str]] = {
            "ss_stage_str": ["on", "pause", "on", "pause", "on", "pause", "on", "off"],
            "ss_sim_stage": [0, 2, 0, 2, 0, 2, 0, 3],
            "ss_heater_cmd": [1, 0, 1, 0, 1, 0, 1, 0],
        }
        for ch, seq in expected.items():
            cast = str if ch == "ss_stage_str" else int
            actual = [cast(v) for v in collected[ch]]
            assert actual[: len(seq)] == seq, f"{ch}: expected {seq} - got {actual}"
