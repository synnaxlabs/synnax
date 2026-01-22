#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import uuid

import synnax as sy
from framework.utils import get_random_name
from console.case import ConsoleCase


class DeviceLifecycle(ConsoleCase):
    """Test the lifecycle of devices in the Console UI."""

    def setup(self) -> None:
        super().setup()
        self.rack_name = "Test Rack " + get_random_name()
        self.test_rack: sy.Rack = self.client.racks.create(name=self.rack_name)
        self.created_devices: list[str] = []

    def run(self) -> None:
        self.test_device_visible_in_toolbar()
        self.test_device_status_update()
        self.test_rename_device()
        self.test_delete_device()

    def cleanup(self) -> None:
        self.client.devices.delete(self.created_devices)
        self.client.racks.delete([self.test_rack.key])
        super().cleanup()

    def _create_test_device(self, name: str, make: str = "NI") -> sy.Device:
        """Create a test device and track it for cleanup."""
        device = self.client.devices.create(
            sy.Device(
                key=uuid.uuid4(),
                rack=self.test_rack.key,
                name=name,
                make=make,
                model=f"Test-{make}",
                location=f"loc_{self.rand_suffix}",
                identifier=f"id_{self.rand_suffix}",
            )
        )
        self.created_devices.append(device.key)
        return device

    def test_device_visible_in_toolbar(self) -> None:
        """Test that a device appears in the devices toolbar."""
        self.log("Testing: Device visible in toolbar")
        device_name = f"TestDevice_{self.rand_suffix}"
        self._create_test_device(device_name)

        self.page.wait_for_timeout(1000)
        exists = self.console.devices.exists(device_name)
        assert exists, f"Device '{device_name}' should be visible in devices toolbar"
        self.log(f"  - Device '{device_name}' is visible in toolbar")

    def test_device_status_update(self) -> None:
        """Test that device status indicator is visible."""
        self.log("Testing: Device status display")
        device_name = f"StatusDevice_{self.rand_suffix}"
        self._create_test_device(device_name)

        self.page.wait_for_timeout(1000)
        status = self.console.devices.get_status(device_name)
        assert status is not None, "Device should have a status indicator"
        self.log(f"  - Device status indicator visible: {status}")

    def test_rename_device(self) -> None:
        """Test renaming a device via context menu."""
        self.log("Testing: Rename device")
        device_name = f"RenameDevice_{self.rand_suffix}"
        self._create_test_device(device_name)

        self.page.wait_for_timeout(500)
        new_name = f"RenamedDevice_{self.rand_suffix}"
        self.console.devices.rename(device_name, new_name)

        assert self.console.devices.exists(
            new_name
        ), f"Device should exist with new name '{new_name}'"
        assert not self.console.devices.exists(
            device_name
        ), f"Device should not exist with old name '{device_name}'"
        self.log(f"  - Device renamed from '{device_name}' to '{new_name}'")

    def test_group_devices(self) -> None:
        """Test grouping multiple devices via context menu."""
        self.log("Testing: Group devices")
        device1_name = f"GroupDevice1_{self.rand_suffix}"
        device2_name = f"GroupDevice2_{self.rand_suffix}"
        self._create_test_device(device1_name)
        self._create_test_device(device2_name)

        self.page.wait_for_timeout(500)
        group_name = f"DeviceGroup_{self.rand_suffix}"
        self.console.devices.group([device1_name, device2_name], group_name)
        self.page.wait_for_timeout(500)

        group_item = self.page.locator(f"div[id^='group:']").filter(has_text=group_name)
        assert group_item.count() > 0, f"Group '{group_name}' should be created"
        self.log(f"  - Created group '{group_name}' with 2 devices")

    def test_delete_device(self) -> None:
        """Test deleting a device via context menu."""
        self.log("Testing: Delete device")
        device_name = f"DeleteDevice_{self.rand_suffix}"
        device = self._create_test_device(device_name)

        self.page.wait_for_timeout(500)
        assert self.console.devices.exists(
            device_name
        ), f"Device '{device_name}' should exist before deletion"

        self.console.devices.delete(device_name)
        self.created_devices.remove(device.key)

        assert not self.console.devices.exists(
            device_name
        ), f"Device '{device_name}' should be deleted"
        self.log(f"  - Device '{device_name}' deleted successfully")
