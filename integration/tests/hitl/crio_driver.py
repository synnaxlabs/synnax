#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from tests.hitl.crio_case import CRIOCase


class CRIODriver(CRIOCase):
    """Checks that the Driver on the cRIO enrolls a rack with NI, matches the version of
    the embedded Driver, and that the Core sees it stop and start again."""

    def run(self) -> None:
        if "ni" not in self.rack.integrations:
            raise AssertionError(
                f"Rack {self.rack.name} reports {self.rack.integrations}, not ni"
            )
        embedded = self.client.racks.retrieve_embedded_rack()
        if self.rack.version != embedded.version:
            raise AssertionError(
                f"cRIO Driver is {self.rack.version}, but the embedded Driver is "
                f"{embedded.version}"
            )
        self.log("Testing: Core sees the Driver stop")
        self.stop_driver()
        self.wait_for_rack("warning")
        self.log("Testing: Driver keeps its rack when it starts again")
        self.start_driver()
        restarted = self.wait_for_rack("success")
        if restarted.key != self.rack.key:
            raise AssertionError(
                f"Driver enrolled rack {restarted.key} instead of {self.rack.key}"
            )
