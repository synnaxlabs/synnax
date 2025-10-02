#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import TYPE_CHECKING, Any, Literal, Optional, Type

import synnax as sy
from playwright.sync_api import Page

from console.task.channels.accelerometer import Accelerometer
from console.task.channels.analog import Analog
from console.task.channels.bridge import Bridge
from console.task.channels.current import Current
from console.task.channels.force_bridge_table import ForceBridgeTable
from console.task.channels.force_bridge_two_point_linear import (
    ForceBridgeTwoPointLinear,
)
from console.task.channels.force_iepe import ForceIEPE
from console.task.channels.microphone import Microphone
from console.task.channels.pressure_bridge_table import PressureBridgeTable
from console.task.channels.pressure_bridge_two_point_linear import (
    PressureBridgeTwoPointLinear,
)
from console.task.channels.resistance import Resistance
from console.task.channels.rtd import RTD
from console.task.channels.strain_gauge import StrainGauge
from console.task.channels.temperature_built_in_sensor import TemperatureBuiltInSensor
from console.task.channels.thermocouple import Thermocouple
from console.task.channels.torque_bridge_table import TorqueBridgeTable
from console.task.channels.torque_bridge_two_point_linear import (
    TorqueBridgeTwoPointLinear,
)
from console.task.channels.velocity_iepe import VelocityIEPE
from console.task.channels.voltage import Voltage

from ..page import ConsolePage

if TYPE_CHECKING:
    from console.console import Console


# Channel type registry for extensible factory pattern
CHANNEL_TYPES: dict[str, Type[Analog]] = {
    "Accelerometer": Accelerometer,
    "Bridge": Bridge,
    "Current": Current,
    "Force Bridge Table": ForceBridgeTable,
    "Force Bridge Two Point Linear": ForceBridgeTwoPointLinear,
    "Force IEPE": ForceIEPE,
    "Microphone": Microphone,
    "Pressure Bridge Table": PressureBridgeTable,
    "Pressure Bridge Two-Point Linear": PressureBridgeTwoPointLinear,
    "Resistance": Resistance,
    "RTD": RTD,
    "Strain Gauge": StrainGauge,
    "Temperature Built-In Sensor": TemperatureBuiltInSensor,
    "Thermocouple": Thermocouple,
    "Torque Bridge Table": TorqueBridgeTable,
    "Torque Bridge Two-Point Linear": TorqueBridgeTwoPointLinear,
    "Velocity IEPE": VelocityIEPE,
    "Voltage": Voltage,
}


class Task(ConsolePage):
    """NI Task automation interface for managing analog channels."""

    channels: list[Analog]
    task_name: str

    def __init__(self, page: Page, console: "Console") -> None:
        super().__init__(page, console)
        self.page_type = "NI Analog Read Task"
        self.pluto_label = ".ni_ai_somethingsomething"
        self.channels = []

    def add_channel(
        self,
        name: str,
        type: str,
        device: str,
        dev_name: Optional[str] = None,
        **kwargs: Any,
    ) -> Analog:
        """
        Add a channel to the task using factory pattern.

        Args:
            name: Channel name
            type: Channel type (must be registered in CHANNEL_TYPES)
            device: Device identifier
            **kwargs: Additional channel-specific configuration

        Returns:
            The created channel instance
        """
        console = self.console
        console.close_all_notifications()

        # Add first channel or subsequent channels
        if len(self.channels) == 0:
            console.click("Add a channel")
        else:
            console.page.locator("header:has-text('Channels') .pluto-icon--add").click()

        # Click the channel in the list
        idx = len(self.channels)
        console.page.locator(".pluto-list__item").nth(idx).click()

        # Configure device
        console.click_btn("Device")
        console.select_from_dropdown(device)

        if dev_name is None:
            dev_name = name[:12]
        # Handle device creation modal if it appears
        sy.sleep(0.2)  # Give modal time to appear
        if console.check_for_modal():
            console.close_all_notifications()
            sy.sleep(0.3)
            console.fill_input_field("Name", dev_name)
            console.click_btn("Next")
            sy.sleep(0.3)
            console.fill_input_field("Identifier", dev_name)
            console.click_btn("Save")
            sy.sleep(0.3)

        if console.check_for_modal():
            raise RuntimeError("Blocking modal is open")

        # Create channel using registry
        if type not in CHANNEL_TYPES:
            raise ValueError(
                f"Unknown channel type: {type}. "
                f"Available types: {list(CHANNEL_TYPES.keys())}"
            )

        channel_class = CHANNEL_TYPES[type]
        channel = channel_class(console=console, device=device, **kwargs)

        self.channels.append(channel)
        return channel

    def set_parameters(
        self,
        task_name: Optional[str] = None,
        sample_rate: Optional[float] = None,
        stream_rate: Optional[float] = None,
        data_saving: Optional[bool] = None,
        auto_start: Optional[bool] = None,
    ) -> None:
        """
        Set the parameters for the task.

        Args:
            sample_rate: The sample rate for the task.
            stream_rate: The stream rate for the task.
            data_saving: Whether to save data to the core.
            auto_start: Whether to start the task automatically.
        """
        console = self.console

        if task_name is not None:
            console.fill_input_field("Name", task_name)
            console.ENTER

        if sample_rate is not None:
            console.fill_input_field("Sample Rate", str(sample_rate))

        if stream_rate is not None:
            console.fill_input_field("Stream Rate", str(stream_rate))

        if data_saving is not None:
            if data_saving != console.get_toggle("Data Saving"):
                console.click_checkbox("Data Saving")

        if auto_start is not None:
            if auto_start != console.get_toggle("Auto Start"):
                console.click_checkbox("Auto Start")

    def configure(self) -> None:
        # Notifications will block the configure channel.
        # Another mitigation is to snap the task page left.
        self.console.close_all_notifications()
        self.console.page.get_by_role("button", name="Configure", exact=True).click(
            force=True
        )

    def run(self) -> None:
        sy.sleep(0.2)
        self.console.page.locator("button .pluto-icon--play").locator("..").click(
            force=True
        )
        sy.sleep(0.2)

    def status(self) -> dict[str, str]:
        """
        Get the current status information from the task status box.

        Returns:
            Dictionary containing:
                - text: The status message (e.g., "Task has not been configured")
                - level: The alert level (e.g., "disabled", "info", "success", "error")
                - name: Status field name
                - time: Timestamp if available
        """
        sy.sleep(0.2)
        status_element = self.console.page.locator(
            ".console-task-state p.pluto-status__text, .console-task-state p.pluto-text"
        ).first

        # status
        class_attr = status_element.get_attribute("class") or ""
        level = "unknown"
        for cls in class_attr.split():
            if cls.startswith("pluto--status-"):
                level = cls.replace("pluto--status-", "")
                break

        msg = status_element.inner_text()

        return {
            "msg": msg,
            "level": level,
        }
