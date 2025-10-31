#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import TYPE_CHECKING, Any

from playwright.sync_api import Page

from console.task.channels.analog import Analog
from console.task.channels.current import Current
from console.task.channels.voltage import Voltage

from .ni import NITask

if TYPE_CHECKING:
    from console.console import Console

# Channel type registry for NI Analog Output
AO_CHANNEL_TYPES: dict[str, type[Analog]] = {
    "Voltage": Voltage,
    "Current": Current,
}


class AnalogWrite(NITask):
    """NI Analog Write/Output Task automation interface."""

    def __init__(self, page: Page, console: "Console") -> None:
        super().__init__(page, console)
        self.page_type = "NI Analog Write Task"

    def new(self) -> str:
        """Create a new NI AO task page."""
        return super().new()

    def add_channel(
        self,
        name: str,
        chan_type: str,
        device: str,
        dev_name: str | None = None,
        **kwargs: Any,
    ) -> Analog:
        """
        Add a channel to the NI AO task. Only Voltage and Current types are allowed.
        Terminal configuration and shunt resistor parameters are not supported for AO tasks.

        Args:
            name: Channel name
            chan_type: Channel type (must be "Voltage" or "Current")
            device: Device identifier
            dev_name: Optional device name
            **kwargs: Additional channel-specific configuration

        Returns:
            The created channel instance

        Raises:
            ValueError: If channel type is not valid for analog write tasks
        """
        if chan_type not in AO_CHANNEL_TYPES:
            raise ValueError(
                f"Invalid channel type for NI Analog Write: {chan_type}. "
                f"Valid types: {list(AO_CHANNEL_TYPES.keys())}"
            )

        # Remove parameters not supported for AO tasks
        kwargs.pop("terminal_config", None)
        kwargs.pop("shunt_resistor", None)
        kwargs.pop("resistance", None)

        return self._add_channel_helper(
            name=name,
            device=device,
            dev_name=dev_name,
            channel_class=AO_CHANNEL_TYPES[chan_type],
            **kwargs,
        )

    def set_parameters(
        self,
        task_name: str | None = None,
        data_saving: bool | None = None,
        auto_start: bool | None = None,
        **kwargs: Any,
    ) -> None:
        """
        Set the parameters for the NI AO task.

        Args:
            task_name: The name of the task.
            state_update_rate: The state update rate for the AO task.
            data_saving: Whether to save data to the core.
            auto_start: Whether to start the task automatically.
            **kwargs: Additional parameters.
        """
        state_update_rate = kwargs.pop("state_update_rate", None)

        super().set_parameters(
            task_name=task_name,
            data_saving=data_saving,
            auto_start=auto_start,
            **kwargs,
        )

        if state_update_rate is not None:
            self.console.fill_input_field("State Update Rate", str(state_update_rate))
