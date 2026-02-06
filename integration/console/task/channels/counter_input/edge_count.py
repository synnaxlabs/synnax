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


class EdgeCount(Counter):
    """
    Edge Count channel type for NI counter read tasks.

    Kwargs:
        port (int): Physical port number
        active_edge (Literal["Rising", "Falling"]): Active edge
        count_direction (Literal["Count Up", "Count Down", "Externally Controlled"]): Count direction
        terminal (str): Input terminal (e.g., "PFI0", "PFI1", etc.)
        initial_count (int): Initial count value
    """

    def __init__(
        self,
        layout: LayoutClient,
        name: str,
        device: str,
        active_edge: Optional[Literal["Rising", "Falling"]] = None,
        count_direction: Optional[
            Literal["Count Up", "Count Down", "Externally Controlled"]
        ] = None,
        terminal: Optional[str] = None,
        initial_count: Optional[int] = None,
        **kwargs: Any,
    ) -> None:
        """Initialize edge count channel with configuration."""
        super().__init__(
            layout=layout,
            name=name,
            device=device,
            chan_type="Edge Count",
            **kwargs,
        )

        self._configure_dropdown("Active Edge", active_edge)
        self._configure_dropdown("Count Direction", count_direction)
        self._configure_dropdown("Input Terminal", terminal)
        self._configure_input("Initial Count", initial_count)
