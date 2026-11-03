#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import synnax as sy
from framework.utils import create_indexed_pair, create_virtual_channel
from tests.arc.arc import ArcCase

# Top-level flows only, so stage entry plays no part. The flow-fed func runs once per
# sample and carries the sample stamps; an inline expression would run once per batch
# at the cycle stamp. The passthrough to sf_seen proves the runtime processed a batch.
SOURCE = """
import stable

func below_limit() {v f64} bool {
    return v < 300.0
}

sf_pt -> below_limit() -> stable.for(duration=200ms) -> sf_out
sf_pt -> sf_seen
"""

WINDOW = 200 * sy.TimeSpan.MILLISECOND
# Every batch lands this long after its stamps, so a close against the cycle clock
# would fire on the first batch.
DELIVERY_DELAY = 500 * sy.TimeSpan.MILLISECOND
READ_TIMEOUT = 5.0


class StableFor(ArcCase):
    """stable.for measures its window from sample stamp to sample stamp.

    The test stamps every sf_pt sample itself, half a second in the past, so the
    delivery delay is fixed and large. A batch older than the window but spanning
    less than it must not close the window; a same-value sample stamped a window
    after the last change closes it exactly once.
    """

    arc_source = SOURCE
    arc_name_prefix = "ArcStableFor"
    start_cmd_channel = "sf_start_cmd"
    subscribe_channels = ["sf_seen"]

    def setup(self) -> None:
        create_indexed_pair(self.client, "sf_pt", data_type=sy.DataType.FLOAT64)
        create_virtual_channel(self.client, "sf_seen", data_type=sy.DataType.FLOAT64)
        create_virtual_channel(self.client, "sf_out", data_type=sy.DataType.BOOLEAN)
        super().setup()

    def verify_sequence_execution(self) -> None:
        t0 = sy.TimeStamp.now()
        self.log(f"Waiting until the wall clock is {DELIVERY_DELAY} past t0")
        while sy.TimeStamp.now() < t0 + DELIVERY_DELAY:
            sy.sleep(0.05)

        # sf_seen and sf_out are written by the same task in the same cycle, so one
        # stream delivers an emit no later than the marker of its batch.
        emitted: list[bool] = []
        with (
            self.client.open_streamer(["sf_seen", "sf_out"]) as stream,
            self.client.open_writer(
                start=t0,
                channels=["sf_pt_time", "sf_pt"],
                enable_auto_commit=True,
            ) as writer,
        ):

            def ingest(values: list[float], offsets_ms: list[int]) -> None:
                stamps = [t0 + off * sy.TimeSpan.MILLISECOND for off in offsets_ms]
                writer.write({"sf_pt_time": stamps, "sf_pt": values})
                seen: float | None = None
                while seen != values[-1]:
                    frame = stream.read(timeout=READ_TIMEOUT)
                    if frame is None:
                        self.fail(f"sf_seen never showed {values[-1]}")
                    if "sf_out" in frame:
                        emitted.extend(bool(v) for v in frame["sf_out"])
                    if "sf_seen" in frame:
                        seen = float(frame["sf_seen"].to_numpy()[-1])

            self.log("Batch stamped t0..t0+20ms: a 20ms hold, no emit")
            ingest([100.0, 101.0, 102.0], [0, 10, 20])
            if emitted:
                self.fail(f"stable.for fired on a batch 1s old: {emitted}")

            self.log("Value changes back inside the window: no emit")
            ingest([400.0, 103.0], [100, 110])
            if emitted:
                self.fail(f"stable.for fired across a change: {emitted}")

            self.log("140ms since the change at t0+110ms: no emit")
            ingest([104.0], [250])
            if emitted:
                self.fail(f"stable.for fired before the window closed: {emitted}")

            self.log("210ms since the change: one emit")
            ingest([105.0], [320])
            if emitted != [True]:
                self.fail(f"expected one true emit after 210ms held, got {emitted}")

            self.log("Same value again: no second emit")
            ingest([106.0], [400])
            if emitted != [True]:
                self.fail(f"stable.for emitted the same value twice: {emitted}")

            self.log("Change to false late in a batch: no emit")
            ingest([107.0, 500.0, 501.0], [700, 710, 720])
            if emitted != [True]:
                self.fail(f"stable.for fired on an unconfirmed change: {emitted}")

            self.log("False confirmed by the first sample, then a change back: false")
            ingest([502.0, 108.0, 109.0], [1000, 1010, 1020])
            if emitted != [True, False]:
                self.fail(f"expected the false confirmed in the batch, got {emitted}")

            self.log("True confirmed by the first sample, then a change: true")
            ingest([110.0, 503.0], [1300, 1310])
            if emitted != [True, False, True]:
                self.fail(f"expected the true confirmed in the batch, got {emitted}")
