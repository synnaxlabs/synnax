#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""The first scan of an EtherCAT slave that the Core does not hold."""

from collections.abc import Iterator

from pydantic import ValidationError

import synnax as sy
from framework.test_case import TestCase
from tests.hitl.ethercat import RESCANNED, prepare_bus

_SCAN_TIMEOUT = 30 * sy.TimeSpan.SECOND
PASS_GAP = 2 * sy.TimeSpan.SECOND
"""The Driver scans every 5 seconds, so a status this long after its device came from a
later scan."""
_DEVICES = "sy_device_set"
_STATUSES = "sy_status_set"


class EtherCATFirstScanStatus(TestCase):
    """Deletes a 32xDO from the Core, and checks that the scan that creates it again
    also sends its status."""

    def setup(self) -> None:
        self.key = prepare_bus(self.client)[RESCANNED].key

    def run(self) -> None:
        owner = sy.device.resource_id(self.key)
        created: sy.TimeStamp | None = None
        matched: list[sy.TimeStamp] = []
        with self.client.open_streamer([_DEVICES, _STATUSES]) as streamer:
            # Each pass sends the status of every device. No device set comes before
            # the delete, so this is not the status that the Driver sends after one.
            old = next((st for _, st in _statuses(streamer, owner)), None)
            if old is None:
                self.fail(f"Expected the scan to send a status for slave {RESCANNED}")
            self.client.devices.delete([self.key])
            for at, frame in _frames(streamer):
                if (
                    created is None
                    and _DEVICES in frame
                    and any(
                        isinstance(raw, dict) and raw.get("key") == self.key
                        for raw in frame[_DEVICES]
                    )
                ):
                    created = at
                matched += [
                    at for st in _owned(frame, owner) if st.message == old.message
                ]
                # The create carries a status, which the Core can send before the
                # device. The second status is the one that the pass sends.
                if created is not None and len(matched) >= 2:
                    break
        if created is None:
            self.fail(f"Expected the scan to create slave {RESCANNED} again")
        if len(matched) < 2:
            self.fail(
                f"Expected the create and the scan to send {old.message!r} for slave "
                f"{RESCANNED}"
            )
        gap = sy.TimeSpan(abs(int(matched[1]) - int(created)))
        if gap > PASS_GAP:
            self.fail(
                f"Expected the status in the scan that created it, not {gap} later"
            )


def _frames(streamer: sy.Streamer) -> Iterator[tuple[sy.TimeStamp, sy.Frame]]:
    """Yields each frame with its arrival time, until _SCAN_TIMEOUT."""
    timer = sy.Timer()
    while timer.elapsed() < _SCAN_TIMEOUT:
        frame = streamer.read(timeout=1)
        if frame is not None:
            yield sy.TimeStamp.now(), frame


def _owned(frame: sy.Frame, owner: sy.resource.ID) -> Iterator[sy.device.Status]:
    if _STATUSES not in frame:
        return
    for raw in frame[_STATUSES]:
        try:
            status = sy.device.Status.model_validate(raw)
        except ValidationError:
            continue
        if status.owner == owner:
            yield status


def _statuses(
    streamer: sy.Streamer, owner: sy.resource.ID
) -> Iterator[tuple[sy.TimeStamp, sy.device.Status]]:
    for at, frame in _frames(streamer):
        for status in _owned(frame, owner):
            yield at, status
