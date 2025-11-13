#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import TYPE_CHECKING, Any

from console.task.channels.analog import Analog

if TYPE_CHECKING:
    from console.console import Console


class Voltage(Analog):
    """
    Voltage channel type for NI analog read tasks.

    Base kwargs:
        port (int): Physical port number
        terminal_config (str): "Default", "Differential", etc.
        min_val (float): Minimum voltage value
        max_val (float): Maximum voltage value
        custom_scale (str): "None", "Linear", "Map", "Table"
    """

    def __init__(
        self,
        console: "Console",
        name: str,
        device: str,
        **kwargs: Any,
    ) -> None:
        """Initialize voltage channel with configuration."""
        super().__init__(
            console=console,
            name=name,
            device=device,
            chan_type="Voltage",
            **kwargs,
        )
