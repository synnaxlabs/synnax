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


class Period(Counter):
    """
    Period channel type for NI counter read tasks.

    Kwargs:
        port (int): Physical port number
        min_val (float): Minimum value
        max_val (float): Maximum value
        starting_edge (str): "Rising" or "Falling"
        units (str): "Seconds", "Ticks", or "Custom"
        terminal (str): Input terminal (e.g., "PFI0", "PFI1", etc.)
        meas_method (str): "One Counter (Low Frequency)", "Two Counters (High Frequency)", "Two Counters (Large Range)", "Dynamic Averaging"
    """

    def __init__(
        self,
        layout: LayoutClient,
        name: str,
        device: str,
        starting_edge: Optional[Literal["Rising", "Falling"]] = None,
        units: Optional[Literal["Seconds", "Ticks", "Custom"]] = None,
        terminal: Optional[str] = None,
        meas_method: Optional[
            Literal[
                "One Counter (Low Frequency)",
                "Two Counters (High Frequency)",
                "Two Counters (Large Range)",
                "Dynamic Averaging",
            ]
        ] = None,
        **kwargs: Any,
    ) -> None:
        """Initialize period channel with configuration."""
        super().__init__(
            layout=layout,
            name=name,
            device=device,
            chan_type="Period",
            **kwargs,
        )

        self._configure_dropdown("Starting Edge", starting_edge)
        self._configure_dropdown("Units", units)
        self._configure_dropdown("Input Terminal", terminal)
        self._configure_dropdown("Measurement Method", meas_method)
