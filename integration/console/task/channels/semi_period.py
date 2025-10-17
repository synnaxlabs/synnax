#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import TYPE_CHECKING, Any, Literal, Optional

from console.task.channels.counter import Counter

if TYPE_CHECKING:
    from console.console import Console


class SemiPeriod(Counter):
    """
    Semi Period channel type for NI counter read tasks.

    Kwargs:
        port (int): Physical port number
        min_val (float): Minimum value
        max_val (float): Maximum value
        units (str): "Seconds", "Ticks", or "FromCustomScale"
    """

    def __init__(
        self,
        console: "Console",
        name: str,
        device: str,
        units: Optional[Literal["Seconds", "Ticks", "FromCustomScale"]] = None,
        **kwargs: Any,
    ) -> None:
        """Initialize semi period channel with configuration."""
        super().__init__(
            console=console,
            name=name,
            device=device,
            type="Semi Period",
            **kwargs,
        )

        # Scaled Units
        if units is not None:
            console.click_btn("Scaled Units")
            console.select_from_dropdown(units)
            self.form_values["Scaled Units"] = units
        else:
            self.form_values["Scaled Units"] = console.get_dropdown_value(
                "Scaled Units"
            )
