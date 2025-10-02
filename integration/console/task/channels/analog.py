#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import TYPE_CHECKING, Literal, Optional

import synnax as sy

if TYPE_CHECKING:
    from console.console import Console


class Analog:
    """Base class for analog channel types in NI tasks."""

    name: str
    console: "Console"
    device: str

    def __init__(
        self,
        console: "Console",
        device: str,
        type: str,
        port: Optional[int] = None,
        terminal_config: Optional[
            Literal[
                "Default",
                "Differential",
                "Pseudo-Differential",
                "Referenced Single Ended",
                "Non-Referenced Single Ended",
            ]
        ] = None,
        min_val: Optional[float] = None,
        max_val: Optional[float] = None,
        custom_scale: Optional[
            Literal[
                "None",
                "Linear",
                "Map",
                "Table",
            ]
        ] = None,
    ) -> None:
        """
        Initialize analog channel with common configuration.

        Args:
            console: Console automation interface
            name: Channel name
            device: Device identifier
            type: Channel type (e.g., "Voltage", "Accelerometer")
            port: Physical port number
            terminal_config: "Default", "Differential", "Pseudo-Differential",
                           "Referenced Single Ended", "Non-Referenced Single Ended"
            min_val: Minimum value
            max_val: Maximum value
            custom_scale: "None", "Linear", "Map", "Table"
        """
        self.console = console
        self.device = device

        # Configure channel type
        console.click_btn("Channel Type")
        console.select_from_dropdown(type)

        # Optional configurations
        if port is not None:
            console.fill_input_field("Port", str(port))

        if terminal_config is not None:
            console.click_btn("Terminal Configuration")
            console.select_from_dropdown(terminal_config)

        if min_val is not None:
            console.fill_input_field("Minimum Value", str(min_val))

        if max_val is not None:
            console.fill_input_field("Maximum Value", str(max_val))

        if custom_scale is not None:
            console.click_btn("Custom Scaling")
            console.select_from_dropdown(custom_scale)
