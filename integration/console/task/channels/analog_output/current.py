#  Copyright 2026 Synnax Labs, Inc.
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


class Current(Analog):
    """
    Current channel type for NI analog write/output tasks.

    Note: This is the output variant without terminal_config or shunt_resistor support.

    Base kwargs:
        port (int): Physical port number
        min_val (float): Minimum current value
        max_val (float): Maximum current value
        custom_scale (str): "None", "Linear", "Map", "Table"
    """

    def __init__(
        self,
        console: "Console",
        name: str,
        device: str,
        **kwargs: Any,
    ) -> None:
        """Initialize current output channel with configuration."""
        # Remove input-only parameters if accidentally passed
        kwargs.pop("terminal_config", None)
        kwargs.pop("shunt_resistor", None)
        kwargs.pop("resistance", None)

        super().__init__(
            console=console,
            name=name,
            device=device,
            chan_type="Current",
            **kwargs,
        )
