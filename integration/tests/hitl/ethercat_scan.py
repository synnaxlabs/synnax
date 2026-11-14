#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""The first scan of an EtherCAT slave that the Core does not hold."""

import synnax as sy
from framework.test_case import TestCase
from tests.hitl.ethercat import RESCANNED, prepare_bus

_SCAN_TIMEOUT = 30 * sy.TimeSpan.SECOND
PASS_GAP = 2 * sy.TimeSpan.SECOND
"""The Driver scans every 5 seconds, so a status this long after its device came from a
later scan."""


class EtherCATFirstScanStatus(TestCase):
    """Deletes a 32xDO from the Core, and checks that the scan that creates it again
    also sends its status."""

    def setup(self) -> None:
        self.key = prepare_bus(self.client)[RESCANNED].key

    def run(self) -> None:
        old = self.client.devices.retrieve(key=self.key, include_status=True).status
        if old is None:
            self.fail(f"Expected a status for slave {RESCANNED}")
        self.client.devices.delete([self.key])
        created: sy.TimeStamp | None = None

        def scanned() -> sy.Device | None:
            nonlocal created
            found = self.client.devices.retrieve(keys=[self.key], include_status=True)
            if not found:
                return None
            if created is None:
                created = sy.TimeStamp.now()
            status = found[0].status
            return found[0] if status is not None and status.time > old.time else None

        dev = sy.poll(scanned, timeout=_SCAN_TIMEOUT, interval=0.1)
        if dev is None or dev.status is None or created is None:
            self.fail(f"Expected the scan to create slave {RESCANNED} with a status")
        gap = sy.TimeSpan(sy.TimeStamp.now() - created)
        if gap > PASS_GAP:
            self.fail(
                f"Expected the status in the scan that created it, not {gap} later"
            )
        status = dev.status
        if (status.variant, status.message) != (old.variant, old.message):
            self.fail(
                f"Expected {old.variant}: {old.message}, but got "
                f"{status.variant}: {status.message}"
            )
