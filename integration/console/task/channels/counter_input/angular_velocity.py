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


class AngularVelocity(Counter):
    """
    Angular Velocity channel type for NI counter read tasks.

    Kwargs:
        port (int): Physical port number
        min_val (float): Minimum value
        max_val (float): Maximum value
        units (str): "RPM", "Radians/s", or "Degrees/s"
        decoding_type (str): "X1", "X2", "X4", or "Two Pulse"
        pulses_per_rev (int): Pulses per revolution
        terminal_a (str): Input Terminal A
        terminal_b (str): Input Terminal B
    """

    def __init__(
        self,
        layout: LayoutClient,
        name: str,
        device: str,
        units: Literal["RPM", "Radians/s", "Degrees/s"] | None = None,
        decoding_type: Literal["X1", "X2", "X4", "Two Pulse"] | None = None,
        pulses_per_rev: int | None = None,
        terminal_a: str | None = None,
        terminal_b: str | None = None,
        **kwargs: Any,
    ) -> None:
        """Initialize angular velocity channel with configuration."""
        super().__init__(
            layout=layout,
            name=name,
            device=device,
            chan_type="Velocity Angular",
            **kwargs,
        )

        self._configure_dropdown("Scaled Units", units)
        self._configure_input("Pulses / Rev", pulses_per_rev)
        self._configure_dropdown("Decoding Type", decoding_type)
        self._configure_dropdown("Input Terminal A", terminal_a)
        self._configure_dropdown("Input Terminal B", terminal_b)
