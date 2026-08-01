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
    """ConsoleCase that deletes the test racks and devices it creates in teardown.

    A leftover rack keeps the rack monitor warning for the rest of the run, and its
    devices keep the Driver health-checking dead endpoints.
    """

    def create_test_rack(self, name: str) -> sy.Rack:
        """Create a rack that teardown deletes."""
        rack = self.client.racks.create(name=name)
        self._rack_keys = [*getattr(self, "_rack_keys", []), rack.key]
        return rack

    def create_test_devices(self, devices: list[sy.Device]) -> list[sy.Device]:
        """Create devices that teardown deletes."""
        created: list[sy.Device] = self.client.devices.create(devices)
        keys = [d.key for d in created]
        self._device_keys = [*getattr(self, "_device_keys", []), *keys]
        return created

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

    def teardown(self) -> None:
        """Delete the created devices and racks so the rack monitor stops warning."""
        with self._try_to("delete test devices"):
            device_keys = getattr(self, "_device_keys", [])
            if device_keys:
                self.client.devices.delete(device_keys)
        with self._try_to("delete test racks"):
            rack_keys = getattr(self, "_rack_keys", [])
            if rack_keys:
                self.client.racks.delete(rack_keys)
        super().teardown()
