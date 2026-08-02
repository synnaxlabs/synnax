#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Tracked rack/device lifecycle mixin.

Designed for multiple inheritance with TestCase subclasses.
"""

import synnax as sy
from framework.models import SynnaxConnection
from framework.test_case import TestCase


class HardwareCase(TestCase):
    """TestCase whose teardown deletes the racks and devices it creates.

    A leftover device keeps the Driver health-checking a dead endpoint, and a leftover
    rack keeps the rack monitor warning for the rest of the run.
    """

    def __init__(
        self,
        synnax_connection: SynnaxConnection = SynnaxConnection(),
        *,
        name: str,
        **params: object,
    ) -> None:
        super().__init__(synnax_connection, name=name, **params)
        self._test_rack_keys: list[int] = []
        self._test_device_keys: list[str] = []

    def create_test_rack(self, name: str) -> sy.Rack:
        """Create a rack that teardown deletes."""
        rack = self.client.racks.create(name=name)
        self._test_rack_keys.append(rack.key)
        return rack

    def create_test_devices(self, devices: list[sy.Device]) -> list[sy.Device]:
        """Create devices that teardown deletes."""
        created: list[sy.Device] = self.client.devices.create(devices)
        self._test_device_keys.extend(d.key for d in created)
        return created

    def teardown(self) -> None:
        """Delete tracked devices then racks, then run the remaining teardown."""
        with self._try_to("delete test devices"):
            if self._test_device_keys:
                self.client.devices.delete(self._test_device_keys)
        with self._try_to("delete test racks"):
            if self._test_rack_keys:
                self.client.racks.delete(self._test_rack_keys)
        super().teardown()
