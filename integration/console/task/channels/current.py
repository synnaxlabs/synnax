#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import TYPE_CHECKING, Any, Literal

from console.task.channels.analog import Analog

if TYPE_CHECKING:
    from console.console import Console


class Current(Analog):
    """
    Current channel type for NI analog read tasks.

    Supported kwargs (in addition to Analog base kwargs):
        shunt_resistor (str): "Default", "Internal", "External"
        resistance (float): Shunt resistance value

    Base kwargs from Analog:
        port (int): Physical port number
        terminal_config (str): "Default", "Differential", etc.
        min_val (float): Minimum value
        max_val (float): Maximum value
        custom_scale (str): "None", "Linear", "Map", "Table"
    """

    def __init__(
        self,
        console: "Console",
        name: str,
        device: str,
        shunt_resistor: Literal["Default" "Internal" "External"] | None = None,
        resistance: float | None = None,
        **kwargs: Any,
    ) -> None:

        # Initialize base analog channel (remaining kwargs passed through)
        super().__init__(
            console=console,
            name=name,
            device=device,
            type="Current",
            **kwargs,
        )

        if shunt_resistor is not None:
            console.click_btn("Shunt Resistor Location")
            console.select_from_dropdown(shunt_resistor)

        if resistance is not None:
            console.fill_input_field("Shunt Resistance", str(resistance))
