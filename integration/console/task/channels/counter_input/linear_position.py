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
        decoding_type (str): "X1", "X2", "X4", or "Two Pulse"
        dist_per_pulse (float): Distance per pulse
        initial_pos (float): Initial position value
        z_index_enable (bool): Enable Z index
        z_index_val (float): Z index value
        z_index_phase (str): "A High B High", "A High B Low", "A Low B High", or "A Low B Low"
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
        decoding_type: Literal["X1", "X2", "X4", "Two Pulse"] | None = None,
        dist_per_pulse: float | None = None,
        initial_pos: float | None = None,
        z_index_enable: bool | None = None,
        z_index_val: float | None = None,
        z_index_phase: (
            Literal["A High B High", "A High B Low", "A Low B High", "A Low B Low"]
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
            chan_type="Position Linear",
            **kwargs,
        )

        self._configure_dropdown("Units", units)
        self._configure_input("Initial Position", initial_pos)
        self._configure_input("Distance / Pulse", dist_per_pulse)
        self._configure_dropdown("Decoding Type", decoding_type)
        self._configure_toggle("Z Index Enable", z_index_enable)
        self._configure_input("Value", z_index_val)
        self._configure_dropdown("Phase", z_index_phase)
        self._configure_dropdown("Input Terminal A", terminal_a)
        self._configure_dropdown("Input Terminal B", terminal_b)
        self._configure_dropdown("Input Terminal Z", terminal_z)
