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
from console.case import ConsoleCase
from console.task.analog_read import AnalogRead
from framework.hardware_case import HardwareCase

HARDWARE = "NI hardware"


class ConnectHardware(HardwareCase, ConsoleCase):
    """Find and configure NI hardware through each way into the find view.

    The test racks report the NI integration but run no Driver, so the find view lists
    the devices this case creates on every platform.
    """

    def setup(self) -> None:
        super().setup()
        self.suffix = random.randint(1000, 9999)
        self.rack = self.create_test_rack(
            f"Find rack {self.suffix}", integrations=["ni"]
        )
        other_rack = self.create_test_rack(
            f"Other rack {self.suffix}", integrations=["ni"]
        )
        self.toolbar_dev, self.task_dev, self.footer_dev, self.other_dev = (
            self.create_test_devices(
                [
                    self._create_ni_device(self.rack, "Toolbar"),
                    self._create_ni_device(self.rack, "Task"),
                    self._create_ni_device(self.rack, "Footer"),
                    self._create_ni_device(other_rack, "Other"),
                ]
            )
        )

    def _create_ni_device(self, rack: sy.Rack, name: str) -> sy.Device:
        return sy.ni.Device(
            rack=rack.key,
            name=f"{name} {self.suffix}",
            model="NI 9229",
            location=f"{name}Mod1",
            identifier=f"{name}Mod1",
        )

    @staticmethod
    def _reads_from(task: sy.Task, device: str) -> bool:
        channels = task.config.get("channels")
        return isinstance(channels, list) and any(
            isinstance(ch, dict) and ch.get("device") == device for ch in channels
        )

    def run(self) -> None:
        self.test_find_from_toolbar()
        self.test_create_task_from_command()
        self.test_find_from_rack()
        self.test_connect_from_task_form()

    def test_find_from_toolbar(self) -> None:
        """The "+" picker leads to a find view of every rack, and Back returns."""
        self.log("Testing: Find hardware from the toolbar")
        devices = self.console.devices
        devices.open_connect()
        devices.pick_integration(HARDWARE)
        for dev in (self.toolbar_dev, self.other_dev):
            devices.found(dev.name).wait_for(state="visible", timeout=5000)
        devices.modal.get_by_role("button", name="Back", exact=True).click()
        devices.modal.get_by_text("Connect by address", exact=True).wait_for(
            state="visible", timeout=5000
        )
        self.console.layout.press_escape()
        devices.modal.wait_for(state="hidden", timeout=5000)

    def test_create_task_from_command(self) -> None:
        """The command opens the find view. Configuring with "Save and create read
        task" saves the device and opens a read task on it."""
        self.log("Testing: Configure a found device and create a read task")
        devices = self.console.devices
        self.console.layout.command_palette(f"Connect {HARDWARE}")
        devices.modal.wait_for(state="visible", timeout=5000)
        name = f"Read device {self.suffix}"
        devices.configure_found(
            self.task_dev.name,
            device_name=name,
            identifier="readdev",
            action="Save and create read task",
        )
        devices.modal.wait_for(state="hidden", timeout=5000)
        self.console.layout.wait_for_tab("NI analog read task")

        tasks = [
            t
            for t in self.client.tasks.retrieve(types=["ni_analog_read"])
            if self._reads_from(t, self.task_dev.key)
        ]
        self.track_test_tasks(tasks)
        assert len(tasks) == 1, f"One read task should use '{name}', got {len(tasks)}"

        dev = self.client.devices.retrieve(key=self.task_dev.key)
        assert dev.configured, f"'{name}' should be configured"
        assert dev.name == name, f"Device name should be '{name}', got '{dev.name}'"
        identifier = dev.properties.get("identifier")
        assert identifier == "readdev", (
            f"Identifier should be 'readdev', got {identifier}"
        )

    def test_find_from_rack(self) -> None:
        """A rack's "Connect device" lists only that rack. Save returns to the list,
        which shows the device as configured."""
        self.log("Testing: Find hardware from a rack")
        devices = self.console.devices
        devices.connect_from_rack(self.rack.name)
        devices.pick_integration(HARDWARE)
        devices.found(self.toolbar_dev.name).wait_for(state="visible", timeout=5000)
        assert devices.found(self.other_dev.name).count() == 0, (
            f"'{self.rack.name}' should not list '{self.other_dev.name}'"
        )
        name = f"Rack device {self.suffix}"
        devices.configure_found(
            self.toolbar_dev.name, device_name=name, identifier="rackdev"
        )
        devices.found(name).get_by_text("Configured", exact=True).wait_for(
            state="visible", timeout=5000
        )
        self.console.layout.press_escape()
        devices.modal.wait_for(state="hidden", timeout=5000)
        assert self.client.devices.retrieve(key=self.toolbar_dev.key).configured, (
            f"'{name}' should be configured"
        )

    def test_connect_from_task_form(self) -> None:
        """The "Connect NI hardware" action of a device dropdown opens the find view,
        and the device configured there becomes the selected device."""
        self.log("Testing: Connect hardware from a task form")
        task_name = f"Connect read {self.suffix}"
        page = self.console.pages.create(AnalogRead, task_name)
        page.connect_device()
        name = f"Form device {self.suffix}"
        self.console.devices.configure_found(
            self.footer_dev.name, device_name=name, identifier="formdev"
        )
        self.console.devices.modal.wait_for(state="hidden", timeout=5000)
        self.console.layout.wait_for_dropdown_value("Device", name)
        self.track_test_tasks(self.client.tasks.retrieve(names=[task_name]))
