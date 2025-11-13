#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import TYPE_CHECKING, Any, Type

from playwright.sync_api import Page

from console.task.channels.counter import Counter
from console.task.channels.counter_input import (
    AngularPosition,
    AngularVelocity,
    DutyCycle,
    EdgeCount,
    Frequency,
    LinearPosition,
    LinearVelocity,
    Period,
    PulseWidth,
    SemiPeriod,
    TwoEdgeSeparation,
)

from .ni import NIChannel, NITask

if TYPE_CHECKING:
    from console.console import Console

# Valid channel types for NI Counter Read tasks
COUNTER_READ_CHANNEL_TYPES: dict[str, Type[Counter]] = {
    "Position Angular": AngularPosition,
    "Velocity Angular": AngularVelocity,
    "Duty Cycle": DutyCycle,
    "Edge Count": EdgeCount,
    "Frequency": Frequency,
    "Position Linear": LinearPosition,
    "Velocity Linear": LinearVelocity,
    "Period": Period,
    "Pulse Width": PulseWidth,
    "Semi Period": SemiPeriod,
    "Two Edge Separation": TwoEdgeSeparation,
}


class CounterRead(NITask):
    """NI Counter Read Task automation interface."""

    def __init__(self, page: Page, console: "Console") -> None:
        super().__init__(page, console)
        self.page_type = "NI Counter Read Task"

    def new(self) -> str:
        """Create a new NI CI task page."""
        return super().new()

    def add_channel(
        self,
        name: str,
        chan_type: str,
        device: str,
        dev_name: str | None = None,
        **kwargs: Any,
    ) -> NIChannel:
        """
        Add a counter read channel to the task.

        Args:
            name: Channel name
            chan_type: Channel type (must be valid for counter read tasks)
            device: Device identifier
            dev_name: Optional device name
            **kwargs: Additional channel-specific configuration

        Returns:
            The created channel instance

        Raises:
            ValueError: If channel type is not valid for counter read tasks
        """
        if chan_type not in COUNTER_READ_CHANNEL_TYPES:
            raise ValueError(
                f"Invalid channel type for NI Counter Read: {chan_type}. "
                f"Valid types: {list(COUNTER_READ_CHANNEL_TYPES.keys())}"
            )

        return self._add_channel_helper(
            name=name,
            device=device,
            dev_name=dev_name,
            channel_class=COUNTER_READ_CHANNEL_TYPES[chan_type],
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
        Set the parameters for the NI Counter Read task.

        Args:
            task_name: The name of the task.
            data_saving: Whether to save data to the core.
            auto_start: Whether to start the task automatically.
            **kwargs: Additional parameters including:
                sample_rate (float): Sample rate for the task.
                stream_rate (float): Stream rate for the task.
        """
        sample_rate = kwargs.pop("sample_rate", None)
        stream_rate = kwargs.pop("stream_rate", None)

        super().set_parameters(
            task_name=task_name,
            data_saving=data_saving,
            auto_start=auto_start,
            **kwargs,
        )

        if sample_rate is not None:
            self.console.fill_input_field("Sample Rate", str(sample_rate))

        if stream_rate is not None:
            self.console.fill_input_field("Stream Rate", str(stream_rate))
