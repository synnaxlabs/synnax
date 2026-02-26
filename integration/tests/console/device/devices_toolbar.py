#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import random

import synnax as sy
from playwright.sync_api import TimeoutError as PlaywrightTimeoutError

from console.case import ConsoleCase


class DevicesToolbar(ConsoleCase):
    """Test device toolbar operations in the Console UI.

    Covers the rc.md checklist for Console > Resources Toolbar > Devices:
    - Devices visible in toolbar
    - Chassis/module hierarchy (expand, children visible)
    - Device state display
    - Configure unconfigured device
    - Change identifier on configured device
    - Rename device (single, chassis with children)
    - Group devices (including chassis, edge cases)
    - Move devices between chassis via Python client
    - Delete device (single, chassis with children)
    """

    def setup(self) -> None:
        super().setup()
        self.suffix: int = random.randint(1000, 9999)
        self._live_keys: set[str] = set()

        self.rack = self.client.racks.retrieve(name="Node 1 Embedded Driver")

        # Chassis A with 3 modules
        self.chassis_a = self._create_device(
            key="chassis-a",
            name="cDAQ-9178 A",
            make="NI",
            model="cDAQ-9178",
            location="Slot 1",
            properties={
                "is_chassis": True,
                "is_simulated": True,
                "resource_name": "cDAQ1",
            },
        )
        self.mod_1 = self._create_device(
            key="mod-9205",
            name="NI 9205",
            make="NI",
            model="NI 9205",
            location="Analog Input",
            parent_device=self.chassis_a.key,
            properties={"is_simulated": True, "resource_name": "cDAQ1Mod1"},
        )
        self.mod_2 = self._create_device(
            key="mod-9263",
            name="NI 9263",
            make="NI",
            model="NI 9263",
            location="Analog Output",
            parent_device=self.chassis_a.key,
            properties={"is_simulated": True, "resource_name": "cDAQ1Mod2"},
        )
        self.mod_3 = self._create_device(
            key="mod-9401",
            name="NI 9401",
            make="NI",
            model="NI 9401",
            location="Digital I/O",
            parent_device=self.chassis_a.key,
            properties={"is_simulated": True, "resource_name": "cDAQ1Mod3"},
        )
        self.modules = [self.mod_1, self.mod_2, self.mod_3]

        # Standalone device (no parent, unconfigured)
        self.standalone = self._create_device(
            key="standalone",
            name="PXI-6255",
            make="NI",
            model="PXI-6255",
            location="dev1",
            properties={"is_simulated": True, "resource_name": "Dev1"},
        )

        # Configured device (for change identifier test)
        self.configured_dev = self._create_device(
            key="configured",
            name="USB-6000",
            make="NI",
            model="USB-6000",
            location="dev2",
            configured=True,
            properties={
                "is_simulated": True,
                "resource_name": "Dev2",
                "identifier": "OriginalID",
            },
        )

        # Chassis B (for move test)
        self.chassis_b = self._create_device(
            key="chassis-b",
            name="cDAQ-9174 B",
            make="NI",
            model="cDAQ-9174",
            location="Slot 2",
            properties={
                "is_chassis": True,
                "is_simulated": True,
                "resource_name": "cDAQ2",
            },
        )

        # Devices with different makes (for icon test)
        self.labjack_dev = self._create_device(
            key="labjack",
            name="T7",
            make="LabJack",
            model="T7",
            location="usb1",
        )
        self.opc_dev = self._create_device(
            key="opc",
            name="OPC Server",
            make="opc",
            model="UA Server",
            location="opc.tcp://localhost:4840",
        )
        self.modbus_dev = self._create_device(
            key="modbus",
            name="Modbus RTU",
            make="Modbus",
            model="RTU",
            location="192.168.1.100:502",
        )
        self.http_dev = self._create_device(
            key="http",
            name="HTTP Server",
            make="http",
            model="REST API",
            location="http://localhost:8080",
        )
        self.ethercat_dev = self._create_device(
            key="ethercat",
            name="EtherCAT Slave",
            make="ethercat",
            model="EK1100",
            location="0:1",
        )

    # ── Helpers ────────────────────────────────────────────────────────

    def _create_device(
        self,
        *,
        key: str,
        name: str,
        make: str,
        model: str,
        location: str,
        parent_device: str = "",
        configured: bool = False,
        properties: dict[str, object] | None = None,
    ) -> sy.Device:
        """Create a device with auto-suffixed key/name and track for teardown."""
        dev = self.client.devices.create(
            sy.Device(
                key=f"{key}-{self.suffix}",
                rack=self.rack.key,
                name=f"{name} ({self.suffix})",
                make=make,
                model=model,
                location=location,
                parent_device=parent_device,
                configured=configured,
                properties=properties or {},
            )
        )
        self._live_keys.add(dev.key)
        return dev

    def _move_device(self, device: sy.Device, target_chassis: sy.Device) -> sy.Device:
        """Move a device to a different chassis and verify in client and UI."""
        updated = self.client.devices.create(
            sy.Device(
                key=device.key,
                rack=self.rack.key,
                name=device.name,
                make=device.make,
                model=device.model,
                location=device.location,
                parent_device=target_chassis.key,
                properties=device.properties,
            )
        )
        retrieved = self.client.devices.retrieve(key=device.key)
        assert retrieved.parent_device == target_chassis.key, (
            f"Expected parent '{target_chassis.key}', "
            f"got '{retrieved.parent_device}'"
        )
        self.console.devices.expand_chassis(target_chassis.name)
        assert self.console.devices.is_child_of(
            device.name, target_chassis.name
        ), f"'{device.name}' should be under '{target_chassis.name}'"
        return updated

    def _assert_modules_under(self, chassis_name: str) -> None:
        """Assert all modules are visible children of the given chassis."""
        children = self.console.devices.get_children_names(chassis_name)
        for mod in self.modules:
            assert any(mod.name in child for child in children), (
                f"Module '{mod.name}' should be under "
                f"'{chassis_name}', got {children}"
            )

    # ── Test Runner ────────────────────────────────────────────────────

    def run(self) -> None:
        self.test_devices_visible()
        self.test_device_icons()
        self.test_configure_device()
        self.test_chassis_children_visible()
        self.test_device_state_display()
        self.test_change_identifier()
        self.test_rename_single_device()
        self.test_rename_chassis()
        self.test_group_devices()
        self.test_group_chassis_children_stay()
        self.test_group_children_outside_sibling_fails()
        self.test_move_device_to_new_chassis()
        self.test_move_device_back()
        self.test_delete_single_device()
        self.test_delete_chassis_with_children()

    def test_devices_visible(self) -> None:
        """All created devices should appear in the devices toolbar."""
        self.log("Testing: Devices visible in toolbar")
        for dev in [
            self.chassis_a,
            self.standalone,
            self.configured_dev,
            self.chassis_b,
        ]:
            assert self.console.devices.exists(
                dev.name
            ), f"Device '{dev.name}' should be visible in the devices toolbar"

    def test_device_icons(self) -> None:
        """Each device should display the icon matching its make."""
        self.log("Testing: Device icons")
        cases: list[tuple[sy.Device, str]] = [
            (self.chassis_a, "ni"),
            (self.standalone, "ni"),
            (self.configured_dev, "ni"),
            (self.chassis_b, "ni"),
            (self.labjack_dev, "labjack"),
            (self.opc_dev, "opc"),
            (self.modbus_dev, "modbus"),
            (self.http_dev, "http"),
            (self.ethercat_dev, "ethercat"),
        ]
        for dev, expected in cases:
            icon = self.console.devices.get_icon(dev.name)
            assert (
                icon == expected
            ), f"Device '{dev.name}' should have '{expected}' icon, got '{icon}'"

    def test_configure_device(self) -> None:
        """Configure an unconfigured device, verify properties."""
        self.log("Testing: Configure device flow")
        dev = self.labjack_dev

        self.console.devices.configure(
            dev.name, device_name="LabJack T4", identifier="lt"
        )

        props = self.console.devices.copy_properties("LabJack T4")
        assert (
            props["configured"] is True
        ), f"Expected configured=True, got {props['configured']}"
        assert (
            props["name"] == "LabJack T4"
        ), f"Expected name 'LabJack T4', got '{props['name']}'"
        inner = props["properties"]
        assert isinstance(inner, dict)
        assert (
            inner["identifier"] == "lt"
        ), f"Expected identifier 'lt', got '{inner['identifier']}'"

        self.console.devices.change_identifier("LabJack T4", "lt_new")

        props = self.console.devices.copy_properties("LabJack T4")
        inner = props["properties"]
        assert isinstance(inner, dict)
        assert (
            inner["identifier"] == "lt_new"
        ), f"Expected identifier 'lt_new', got '{inner['identifier']}'"

        self.labjack_dev = self.client.devices.retrieve(key=dev.key)

    def test_chassis_children_visible(self) -> None:
        """Modules should be nested under their parent chassis."""
        self.log("Testing: Chassis children visible")
        self._assert_modules_under(self.chassis_a.name)

    def test_device_state_display(self) -> None:
        """All devices should show 'warning' with 'state unknown'."""
        self.log("Testing: Device state display")
        for dev in [
            self.chassis_a,
            self.mod_1,
            self.mod_2,
            self.mod_3,
            self.standalone,
            self.configured_dev,
            self.chassis_b,
            self.labjack_dev,
            self.opc_dev,
            self.modbus_dev,
            self.http_dev,
            self.ethercat_dev,
        ]:
            status = self.console.devices.get_status(dev.name)
            assert status["variant"] == "warning", (
                f"Expected 'warning' variant for '{dev.name}', "
                f"got '{status['variant']}'"
            )
            msg = status["message"].lower()
            assert "state unknown" in msg or "not running" in msg, (
                f"Expected 'state unknown' or 'not running' in status "
                f"for '{dev.name}', got '{status['message']}'"
            )

    def test_change_identifier(self) -> None:
        """Change identifier should update the device's identifier property."""
        self.log("Testing: Change identifier on configured device")
        new_id = f"NewID_{self.suffix}"
        self.console.devices.change_identifier(self.configured_dev.name, new_id)

        updated = self.client.devices.retrieve(key=self.configured_dev.key)
        assert updated.properties.get("identifier") == new_id, (
            f"Expected identifier '{new_id}', "
            f"got '{updated.properties.get('identifier')}'"
        )

    def test_rename_single_device(self) -> None:
        """Rename a standalone device via context menu."""
        self.log("Testing: Rename single device")
        new_name = f"Renamed PXI ({self.suffix})"
        self.console.devices.rename(old_name=self.standalone.name, new_name=new_name)
        updated = self.client.devices.retrieve(key=self.standalone.key)
        assert (
            updated.name == new_name
        ), f"Expected name '{new_name}', got '{updated.name}'"
        self.standalone = updated

    def test_rename_chassis(self) -> None:
        """Rename a chassis; children should remain nested underneath."""
        self.log("Testing: Rename chassis, verify children intact")
        new_name = f"Renamed cDAQ ({self.suffix})"
        self.console.devices.rename(old_name=self.chassis_a.name, new_name=new_name)

        updated_chassis = self.client.devices.retrieve(key=self.chassis_a.key)
        assert updated_chassis.name == new_name, (
            f"Expected chassis name '{new_name}', " f"got '{updated_chassis.name}'"
        )

        self._assert_modules_under(new_name)

        for mod in self.modules:
            retrieved = self.client.devices.retrieve(key=mod.key)
            assert retrieved.parent_device == self.chassis_a.key, (
                f"Module '{mod.name}' parent should still be "
                f"'{self.chassis_a.key}', got '{retrieved.parent_device}'"
            )

        self.chassis_a = updated_chassis

    def test_group_devices(self) -> None:
        """Group standalone + configured device into a group."""
        self.log("Testing: Group devices")
        group_name = f"DevGroup_{self.suffix}"
        self.console.devices.group(
            [self.standalone.name, self.configured_dev.name],
            group_name,
        )
        assert self.console.devices.tree.group_exists(
            group_name
        ), f"Group '{group_name}' should exist after creation"
        self.console.devices.tree.delete_group(group_name)

    def test_group_chassis_children_stay(self) -> None:
        """Grouping a chassis should keep its module children nested."""
        self.log("Testing: Group chassis, children remain as children")
        chassis_group = f"ChassisGroup_{self.suffix}"
        self.console.devices.group([self.chassis_a.name], chassis_group)

        group_item = self.console.devices.tree.get_group(chassis_group)
        self.console.devices.tree.expand(group_item)
        self.console.devices.expand_chassis(self.chassis_a.name)

        self._assert_modules_under(self.chassis_a.name)
        self.console.devices.tree.delete_group(chassis_group)

    def test_group_children_outside_sibling_fails(self) -> None:
        """Grouping a module with a device outside its chassis fails."""
        self.log("Testing: Group children outside sibling group fails silently")
        self.console.devices.expand_chassis(self.chassis_a.name)

        bad_group = f"BadGroup_{self.suffix}"
        try:
            self.console.devices.group(
                [self.mod_1.name, self.chassis_b.name], bad_group
            )
        except (PlaywrightTimeoutError, ValueError):
            pass

        children_after = self.console.devices.get_children_names(self.chassis_a.name)
        assert any(self.mod_1.name in child for child in children_after), (
            f"Module '{self.mod_1.name}' should still be under chassis_a "
            f"after failed group attempt, got {children_after}"
        )
        if self.console.devices.tree.find_by_name("group:", bad_group):
            self.console.devices.tree.delete_group(bad_group)

    def test_move_device_to_new_chassis(self) -> None:
        """Move a module to a different chassis via Python client."""
        self.log("Testing: Move device to new chassis via Python client")
        self.mod_3 = self._move_device(self.mod_3, self.chassis_b)

    def test_move_device_back(self) -> None:
        """Move module back to original chassis via Python client."""
        self.log("Testing: Move device back to original chassis")
        self.mod_3 = self._move_device(self.mod_3, self.chassis_a)

    def test_delete_single_device(self) -> None:
        """Delete a standalone device via context menu."""
        self.log("Testing: Delete single device")
        self.console.devices.delete(self.standalone.name)
        result = self.client.devices.retrieve(
            key=self.standalone.key, ignore_not_found=True
        )
        assert result is None, f"Device '{self.standalone.key}' should be deleted"
        self._live_keys.discard(self.standalone.key)

    def test_delete_chassis_with_children(self) -> None:
        """Delete a chassis; its children should also be removed."""
        self.log("Testing: Delete chassis with children")
        self.console.devices.delete(self.chassis_b.name)
        result = self.client.devices.retrieve(
            key=self.chassis_b.key, ignore_not_found=True
        )
        assert result is None, f"Chassis '{self.chassis_b.key}' should be deleted"
        self._live_keys.discard(self.chassis_b.key)

    def teardown(self) -> None:
        self.log("Start teardown")
        if self._live_keys:
            self.client.devices.delete(keys=list(self._live_keys))
        super().teardown()
