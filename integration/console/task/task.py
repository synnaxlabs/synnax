#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import Any, Literal, Optional, Type

from console.console import Console
from console.task.channels.accelerometer import Accelerometer
from console.task.channels.analog import Analog
from console.task.channels.bridge import Bridge
from console.task.channels.voltage import Voltage


# Channel type registry for extensible factory pattern
CHANNEL_TYPES: dict[str, Type[Analog]] = {
    "Accelerometer": Accelerometer,
    "Bridge": Bridge,
    "Voltage": Voltage,
}


class Task:
    """NI Task automation interface for managing analog channels."""

    channels: list[Analog]
    console: Console
    name: str

    def __init__(
        self,
        console: Console,
        type: Literal[
            "NI Analog Read Task",
        ],
        name: Optional[str] = None,
    ) -> None:
        """Initialize a new task in the Console."""
        self.console = console
        self.channels = []

        page_type = "NI Analog Read Task"
        if name is None:
            name = page_type

        _, page_id = console.create_page(page_type, name)
        self.name = name

    def add_channel(
        self,
        name: str,
        type: Literal[*CHANNEL_TYPES.keys()],
        device: str,
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

        # Add first channel or subsequent channels
        if len(self.channels) == 0:
            console.click("Add a channel")
        else:
            console.page.locator(
                "header:has-text('Channels') .pluto-icon--add"
            ).click()

        # Click the channel in the list
        idx = len(self.channels)
        console.page.locator(".pluto-list__item").nth(idx).click()

        # Create channel using registry
        if type not in CHANNEL_TYPES:
            raise ValueError(
                f"Unknown channel type: {type}. "
                f"Available types: {list(CHANNEL_TYPES.keys())}"
            )

        channel_class = CHANNEL_TYPES[type]
        channel = channel_class(
            console=console, name=name, device=device, **kwargs
        )

        self.channels.append(channel)
        return channel
