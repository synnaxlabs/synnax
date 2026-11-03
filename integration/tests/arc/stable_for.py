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
# at the cycle stamp. The passthroughs to sf_seen and sf_vseen prove the runtime
# processed a batch.
SOURCE = """
import stable

func below_limit() {v f64} bool {
    return v < 300.0
}

// ──── stored input, stamped by the test ────
sf_pt -> below_limit() -> stable.for(duration=200ms) -> sf_out
sf_pt -> sf_seen

// ──── virtual input, stamped by the runtime at read time ────
sf_vpt -> below_limit() -> stable.for(duration=200ms) -> sf_vout
sf_vpt -> sf_vseen
"""

# Every stored batch lands this long after its stamps, so a close against the cycle
# clock would fire on the first batch.
DELIVERY_DELAY = 300 * sy.TimeSpan.MILLISECOND
# The virtual flow's window, and the wall-clock wait that proves a hold past it.
VIRTUAL_WINDOW = 200 * sy.TimeSpan.MILLISECOND
VIRTUAL_HOLD = 300 * sy.TimeSpan.MILLISECOND
READ_TIMEOUT = 5.0


class StableFor(ArcCase):
    """stable.for measures its window from sample stamp to sample stamp.

    The stored flow stamps every sf_pt sample itself, 300 ms in the past, so
    the delivery delay is fixed and large. A batch older than the window but spanning
    less than it must not close the window; a same-value sample stamped a window after
    the last change closes it exactly once.

    The virtual flow has no index, so the runtime stamps each batch at the cycle that
    reads it, 1 ns apart inside a batch. A hold is then wall time between batches.
    """

    arc_source = SOURCE
    arc_name_prefix = "ArcStableFor"
    start_cmd_channel = "sf_start_cmd"
    subscribe_channels = ["sf_seen", "sf_vseen"]

    def setup(self) -> None:
        create_indexed_pair(self.client, "sf_pt", data_type=sy.DataType.FLOAT64)
        create_virtual_channel(self.client, "sf_vpt", data_type=sy.DataType.FLOAT64)
        create_virtual_channel(self.client, "sf_seen", data_type=sy.DataType.FLOAT64)
        create_virtual_channel(self.client, "sf_vseen", data_type=sy.DataType.FLOAT64)
        create_virtual_channel(self.client, "sf_out", data_type=sy.DataType.BOOLEAN)
        create_virtual_channel(self.client, "sf_vout", data_type=sy.DataType.BOOLEAN)
        super().setup()

    def _settle(
        self,
        stream: sy.Streamer,
        marker: str,
        value: float,
        out: str,
        emitted: list[bool],
    ) -> None:
        """Reads stream until marker shows value, collecting every out sample.

        The marker and out are written by the same task in the same cycle, so one
        stream delivers an emit no later than the marker of its batch.
        """
        seen: float | None = None
        while seen != value:
            frame = stream.read(timeout=READ_TIMEOUT)
            if frame is None:
                self.fail(f"{marker} never showed {value}")
            if out in frame:
                emitted.extend(bool(v) for v in frame[out])
            if marker in frame:
                seen = float(frame[marker].to_numpy()[-1])

    def _expect(self, emitted: list[bool], want: list[bool], why: str) -> None:
        if emitted != want:
            self.fail(f"{why}: expected {want}, got {emitted}")

    def verify_sequence_execution(self) -> None:
        self._verify_stored()
        self._verify_virtual()

    def _verify_stored(self) -> None:
        t0 = sy.TimeStamp.now()
        self.log(f"Waiting until the wall clock is {DELIVERY_DELAY} past t0")
        while sy.TimeStamp.now() < t0 + DELIVERY_DELAY:
            sy.sleep(0.05)

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
                self._settle(stream, "sf_seen", values[-1], "sf_out", emitted)

            self.log("[stored] Batch stamped t0..t0+20ms: a 20ms hold, no emit")
            ingest([100.0, 101.0, 102.0], [0, 10, 20])
            self._expect(emitted, [], "fired on a batch older than the window")

            self.log("[stored] Value changes back inside the window: no emit")
            ingest([400.0, 103.0], [100, 110])
            self._expect(emitted, [], "fired across a change")

            self.log("[stored] 140ms since the change at t0+110ms: no emit")
            ingest([104.0], [250])
            self._expect(emitted, [], "fired before the window closed")

            self.log("[stored] 210ms since the change: one emit")
            ingest([105.0], [320])
            self._expect(emitted, [True], "one true emit after 210ms held")

            self.log("[stored] Same value again: no second emit")
            ingest([106.0], [400])
            self._expect(emitted, [True], "emitted the same value twice")

            self.log("[stored] Change to false late in a batch: no emit")
            ingest([107.0, 500.0, 501.0], [700, 710, 720])
            self._expect(emitted, [True], "fired on an unconfirmed change")

            self.log("[stored] False confirmed by the first sample, then true: false")
            ingest([502.0, 108.0, 109.0], [1000, 1010, 1020])
            self._expect(emitted, [True, False], "false confirmed in the batch")

            self.log("[stored] True confirmed by the first sample, then false: true")
            ingest([110.0, 503.0], [1300, 1310])
            self._expect(emitted, [True, False, True], "true confirmed in the batch")

    def _verify_virtual(self) -> None:
        emitted: list[bool] = []
        with (
            self.client.open_streamer(["sf_vseen", "sf_vout"]) as stream,
            self.client.open_writer(
                start=sy.TimeStamp.now(),
                channels=["sf_vpt"],
                enable_auto_commit=True,
            ) as writer,
        ):
            last_write = sy.TimeStamp.now()

            def ingest(values: list[float], *, hold: bool = False) -> None:
                nonlocal last_write
                if hold:
                    while sy.TimeStamp.now() < last_write + VIRTUAL_HOLD:
                        sy.sleep(0.02)
                writer.write({"sf_vpt": values})
                last_write = sy.TimeStamp.now()
                self._settle(stream, "sf_vseen", values[-1], "sf_vout", emitted)

            self.log("[virtual] One batch, 1ns apart: no emit")
            ingest([100.0, 101.0, 102.0])
            self._expect(emitted, [], "fired inside one batch")

            self.log("[virtual] Second batch at once: no emit")
            ingest([103.0, 104.0])
            self._expect(emitted, [], "fired before the window closed")

            self.log("[virtual] Batch after the hold, then a change to false: true")
            ingest([105.0, 400.0, 401.0], hold=True)
            self._expect(emitted, [True], "true confirmed after the hold")

            self.log("[virtual] False at once: no emit")
            ingest([402.0])
            self._expect(emitted, [True], "fired before the false held")

            self.log("[virtual] False after the hold, then a change to true: false")
            ingest([403.0, 106.0], hold=True)
            self._expect(emitted, [True, False], "false confirmed after the hold")

            self.log("[virtual] True after the hold: true")
            ingest([107.0, 108.0], hold=True)
            self._expect(emitted, [True, False, True], "true confirmed again")
