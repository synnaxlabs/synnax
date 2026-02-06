#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import Any

from console.layout import LayoutClient
from console.task.channels.analog import Analog


class Voltage(Analog):
    """
    Voltage channel type for NI analog read tasks.

    Base kwargs:
        port (int): Physical port number
        terminal_config (str): "Default", "Differential", etc.
        min_val (float): Minimum voltage value
        max_val (float): Maximum voltage value
        custom_scale (str): "None", "Linear", "Map", "Table"
    """

    def __init__(
        self,
        layout: LayoutClient,
        name: str,
        device: str,
        **kwargs: Any,
    ) -> None:
        """Initialize voltage channel with configuration."""
        super().__init__(
            layout=layout,
            name=name,
            device=device,
            chan_type="Voltage",
            **kwargs,
        )
