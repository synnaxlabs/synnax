#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import Any, Literal, Optional

from console.layout import LayoutClient
from console.task.channels.counter import Counter


class SemiPeriod(Counter):
    """
    Semi Period channel type for NI counter read tasks.

    Kwargs:
        port (int): Physical port number
        min_val (float): Minimum value
        max_val (float): Maximum value
        units (str): "Seconds", "Ticks", or "Custom"
    """

    def __init__(
        self,
        layout: LayoutClient,
        name: str,
        device: str,
        units: Optional[Literal["Seconds", "Ticks", "Custom"]] = None,
        **kwargs: Any,
    ) -> None:
        """Initialize semi period channel with configuration."""
        super().__init__(
            layout=layout,
            name=name,
            device=device,
            chan_type="Semi Period",
            **kwargs,
        )
        layout = self.layout

        # Scaled Units
        if units is not None:
            layout.click_btn("Scaled Units")
            layout.select_from_dropdown(units)
            self.form_values["Scaled Units"] = units
        else:
            self.form_values["Scaled Units"] = layout.get_dropdown_value("Scaled Units")
