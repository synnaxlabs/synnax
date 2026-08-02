#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import synnax as sy
from console.case import ConsoleCase


class RackCase(ConsoleCase):
    """ConsoleCase with a helper to create the NI test rack the form tests share."""

    def create_test_ni_rack(
        self, rack_name: str, device_name: str, device_key: str
    ) -> None:
        """Create a rack holding one NI 9229 module, both deleted in teardown."""
        rack = self.create_test_rack(rack_name)
        self.create_test_devices(
            [
                sy.ni.Device(
                    key=device_key,
                    rack=rack.key,
                    name=device_name,
                    model="NI 9229",
                    location=device_name,
                    identifier=f"{device_name}Mod1",
                )
            ]
        )
        sy.sleep(1)
