#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import TYPE_CHECKING, Any, Optional, Type

from playwright.sync_api import Page

from console.task.channels.pulse_output import PulseOutput

from .ni import NITask

if TYPE_CHECKING:
    from console.console import Console

# Valid channel types for NI Counter Write tasks
COUNTER_WRITE_CHANNEL_TYPES: dict[str, Type[PulseOutput]] = {
    "Pulse Output": PulseOutput,
}


class CounterWrite(NITask):
    """NI Counter Write Task automation interface."""

    def __init__(self, page: Page, console: "Console") -> None:
        super().__init__(page, console)
        self.page_type = "NI Counter Write Task"

    def new(self) -> str:
        """Create a new NI CO task page."""
        return super().new()

    def add_channel(
        self,
        name: str,
        type: str,
        device: str,
        dev_name: Optional[str] = None,
        **kwargs: Any,
    ) -> PulseOutput:
        """
        Add a counter write channel to the task.

        Args:
            name: Channel name
            type: Channel type (must be valid for counter write tasks)
            device: Device identifier
            dev_name: Optional device name
            **kwargs: Additional channel-specific configuration

        Returns:
            The created channel instance

        Raises:
            ValueError: If channel type is not valid for counter write tasks
        """
        if type not in COUNTER_WRITE_CHANNEL_TYPES:
            raise ValueError(
                f"Invalid channel type for NI Counter Write: {type}. "
                f"Valid types: {list(COUNTER_WRITE_CHANNEL_TYPES.keys())}"
            )

        return self._add_channel_helper(
            name=name,
            type=type,
            device=device,
            dev_name=dev_name,
            channel_class=COUNTER_WRITE_CHANNEL_TYPES[type],
            **kwargs,
        )

    def set_parameters(
        self,
        task_name: Optional[str] = None,
        data_saving: Optional[bool] = None,
        auto_start: Optional[bool] = None,
        **kwargs: Any,
    ) -> None:
        """
        Set the parameters for the NI CO task.

        Args:
            task_name: The name of the task.
            state_update_rate: The state update rate for the CO task.
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
