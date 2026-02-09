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


class LinearVelocity(Counter):
    """
    Linear Velocity channel type for NI counter read tasks.

    Kwargs:
        port (int): Physical port number
        min_val (float): Minimum value
        max_val (float): Maximum value
        units (str): "m/s" or "in/s"
        decoding_type (str): "X1", "X2", "X4", or "Two Pulse"
        dist_per_pulse (float): Distance per pulse
        terminal_a (str): Input Terminal A
        terminal_b (str): Input Terminal B
    """

    def __init__(
        self,
        layout: LayoutClient,
        name: str,
        device: str,
        units: Literal["m/s", "in/s"] | None = None,
        decoding_type: Literal["X1", "X2", "X4", "Two Pulse"] | None = None,
        dist_per_pulse: float | None = None,
        terminal_a: str | None = None,
        terminal_b: str | None = None,
        **kwargs: Any,
    ) -> None:
        """Initialize linear velocity channel with configuration."""
        super().__init__(
            layout=layout,
            name=name,
            device=device,
            chan_type="Velocity Linear",
            **kwargs,
        )

        self._configure_dropdown("Scaled Units", units)
        self._configure_input("Distance / Pulse", dist_per_pulse)
        self._configure_dropdown("Decoding Type", decoding_type)
        self._configure_dropdown("Input Terminal A", terminal_a)
        self._configure_dropdown("Input Terminal B", terminal_b)
