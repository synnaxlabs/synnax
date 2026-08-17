#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import Any, Literal

from console.layout import LayoutClient
from console.task.channels.counter import Counter


class LinearPosition(Counter):
    """
    Linear Position channel type for NI counter read tasks.

    Kwargs:
        port (int): Physical port number
        units (str): "Meters", "Inches", or "Ticks"
        decoding_type (str): "X1", "X2", "X4", or "Two pulse"
        dist_per_pulse (float): Distance per pulse
        initial_pos (float): Initial position value
        z_index_enable (bool): Enable Z index
        z_index_val (float): Z index value
        z_index_phase (str): "A high B high", "A high B low", "A low B high", or "A low B low"
        terminal_a (str): Input Terminal A
        terminal_b (str): Input Terminal B
        terminal_z (str): Input Terminal Z
    """

    def __init__(
        self,
        layout: LayoutClient,
        name: str,
        device: str,
        units: Literal["Meters", "Inches", "Ticks"] | None = None,
        decoding_type: Literal["X1", "X2", "X4", "Two pulse"] | None = None,
        dist_per_pulse: float | None = None,
        initial_pos: float | None = None,
        z_index_enable: bool | None = None,
        z_index_val: float | None = None,
        z_index_phase: (
            Literal["A high B high", "A high B low", "A low B high", "A low B low"]
            | None
        ) = None,
        terminal_a: str | None = None,
        terminal_b: str | None = None,
        terminal_z: str | None = None,
        **kwargs: Any,
    ) -> None:
        """Initialize linear position channel with configuration."""
        super().__init__(
            layout=layout,
            name=name,
            device=device,
            chan_type="Position linear",
            **kwargs,
        )

        self._configure_dropdown("Units", units)
        self._configure_input("Initial position", initial_pos)
        self._configure_input("Distance / Pulse", dist_per_pulse)
        self._configure_dropdown("Decoding type", decoding_type)
        self._configure_toggle("Z index enable", z_index_enable)
        self._configure_input("Value", z_index_val)
        self._configure_dropdown("Phase", z_index_phase)
        self._configure_dropdown("Input terminal A", terminal_a)
        self._configure_dropdown("Input terminal B", terminal_b)
        self._configure_dropdown("Input terminal Z", terminal_z)
