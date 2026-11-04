#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import numpy as np

import synnax as sy
from framework.utils import create_indexed_pair, create_virtual_channel
from tests.arc.arc import ArcCase
from x import random_name

ARC_STALE_BATCH_ABORT = """
import stable
import time

func is_low() {{v f64}} bool {{
    return v < 300.0
}}

start_stale_batch_abort_cmd => main

sequence main {{
    stage arm {{
        "arm" -> sba_log
        sba_go == 1 => next
    }}
    stage run {{
        "run" -> sba_log
        time.wait(500ms) -> sba_ready
        {sensor} -> sba_sensor_seen
        {sensor} -> is_low() -> stable.for(duration=400ms) -> {hold}
        {sensor} -> is_low() -> stable.for(duration=400ms) => aborted
    }}
    stage aborted {{
        "aborted" -> sba_log
    }}
}}
"""

WINDOW = 400 * sy.TimeSpan.MILLISECOND
STALE_SAMPLES = 50
STALE_FROM = 6 * sy.TimeSpan.SECOND
STALE_SPAN = 2 * sy.TimeSpan.SECOND
IN_STAGE_SAMPLES = 50
IN_STAGE_FROM = 50 * sy.TimeSpan.MILLISECOND
IN_STAGE_SPAN = 200 * sy.TimeSpan.MILLISECOND
PROBE_VALUE = 400.0
COLLECT_TIMEOUT = 5 * sy.TimeSpan.SECOND


class StaleBatchAbort(ArcCase):
    """A stage check must act only on samples taken inside the stage (SY-5456)
    and measure its hold in sample time (SY-5457).

    One batch lands 500 ms after entry: 2 s of low samples from before entry,
    then 200 ms from inside it. The 400 ms window must not close. A probe above
    the threshold must then pass through."""

    arc_source = ""
    arc_name_prefix = "StaleBatchAbort"
    start_cmd_channel = "start_stale_batch_abort_cmd"
    subscribe_channels = ["sba_log", "sba_ready"]

    def setup(self) -> None:
        # Fresh indexed channels per run: the batch is written into the past, and
        # a repeat run within seconds would overlap the previous domain.
        suffix = random_name()
        self._sensor = f"sba_sensor_{suffix}"
        self._hold = f"sba_hold_{suffix}"
        self.arc_source = ARC_STALE_BATCH_ABORT.format(
            sensor=self._sensor, hold=self._hold
        )
        create_indexed_pair(self.client, self._sensor, data_type=sy.DataType.FLOAT64)
        create_indexed_pair(self.client, self._hold, data_type=sy.DataType.BOOLEAN)
        create_virtual_channel(
            self.client, "sba_sensor_seen", data_type=sy.DataType.FLOAT64
        )
        create_virtual_channel(self.client, "sba_log", data_type=sy.DataType.STRING)
        create_virtual_channel(self.client, "sba_go", data_type=sy.DataType.UINT8)
        create_virtual_channel(self.client, "sba_ready", data_type=sy.DataType.UINT8)
        super().setup()

    def verify_sequence_execution(self) -> None:
        self.wait_for_eq("sba_log", "arm")
        t_go = sy.TimeStamp.now()
        self.writer.write("sba_go", 1)
        self.wait_for_eq("sba_log", "run")
        t_after = sy.TimeStamp.now()

        # The stage reports ready 500 ms after entry on the runtime's own clock, so
        # the batch lands after the window has passed on the wall clock.
        self.wait_for_eq("sba_ready", 1)

        first = t_go - STALE_FROM
        stale_step = STALE_SPAN / STALE_SAMPLES
        in_stage_step = IN_STAGE_SPAN / IN_STAGE_SAMPLES
        stamps = [first + stale_step * i for i in range(STALE_SAMPLES)]
        stamps += [
            t_after + IN_STAGE_FROM + in_stage_step * i for i in range(IN_STAGE_SAMPLES)
        ]
        values = [100.0 + i for i in range(len(stamps))]
        probe = sy.TimeStamp.now()
        self.log(
            f"straddling batch: {STALE_SAMPLES} stale, {IN_STAGE_SAMPLES} in stage, "
            "then probe"
        )
        hold_time = f"{self._hold}_time"
        seen: list[float] = []
        hold_stamps: list[sy.TimeStamp] = []
        with (
            self.client.open_streamer(["sba_sensor_seen", hold_time]) as stream,
            self.client.open_writer(
                first,
                [f"{self._sensor}_time", self._sensor],
                enable_auto_commit=True,
            ) as w,
        ):
            w.write({f"{self._sensor}_time": stamps, self._sensor: values})
            w.write({f"{self._sensor}_time": probe, self._sensor: PROBE_VALUE})
            timer = sy.Timer()
            while timer.elapsed() < COLLECT_TIMEOUT and PROBE_VALUE not in seen:
                frame = stream.read(timeout=0.5)
                if frame is None:
                    continue
                if "sba_sensor_seen" in frame:
                    seen.extend(
                        frame["sba_sensor_seen"].to_numpy().astype(np.float64).tolist()
                    )
                if hold_time in frame:
                    hold_stamps.extend(
                        sy.TimeStamp(int(v))
                        for v in frame[hold_time].to_numpy().astype(np.int64)
                    )

        if PROBE_VALUE in seen:
            self.wait_for_eq("sba_log", "run")
            return
        self.fail(self._diagnose(seen, hold_stamps, t_after))

    def _diagnose(
        self,
        seen: list[float],
        hold_stamps: list[sy.TimeStamp],
        t_after: sy.TimeStamp,
    ) -> str:
        """Name the mechanism behind an abort from what the stage processed."""
        admitted = len(seen) > IN_STAGE_SAMPLES
        lines = [
            "the probe never reached the stage",
            f"samples the stage processed: {len(seen)}, first "
            f"{seen[0] if seen else None}",
        ]
        if admitted:
            lines.append("  -> samples taken before entry were admitted (SY-5456)")
        if hold_stamps:
            since_entry = sy.TimeSpan(hold_stamps[0] - t_after)
            lines.append(f"check fired with stamp {since_entry} after entry")
            if since_entry > WINDOW:
                lines.append("  -> window closed against the wall clock (SY-5457)")
            else:
                lines.append("  -> window opened on samples taken before entry")
        else:
            lines.append("check did not fire")
        return "\n".join(lines)
