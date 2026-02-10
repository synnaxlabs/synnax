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


class Frequency(Counter):
    """
    Frequency channel type for NI counter read tasks.

    Kwargs:
        port (int): Physical port number
        min_val (float): Minimum value
        max_val (float): Maximum value
        edge (str): "Rising" or "Falling"
        units (str): "Hz", "Ticks", or "Custom"
        terminal (str): Input terminal (e.g., "PFI0", "PFI1", etc.)
        meas_method (str): "One Counter (Low Frequency)", "Two Counters (High Frequency)", "Two Counters (Large Range)", "Dynamic Averaging"
    """

    def __init__(
        self,
        layout: LayoutClient,
        name: str,
        device: str,
        edge: Optional[Literal["Rising", "Falling"]] = None,
        units: Optional[Literal["Hz", "Ticks", "Custom"]] = None,
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
        """Initialize frequency channel with configuration."""
        super().__init__(
            layout=layout,
            name=name,
            device=device,
            chan_type="Frequency",
            **kwargs,
        )

        self._configure_dropdown("Starting Edge", edge)
        self._configure_dropdown("Units", units)
        self._configure_dropdown("Input Terminal", terminal)
        self._configure_dropdown("Measurement Method", meas_method)
