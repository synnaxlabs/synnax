#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import Unpack

from console.layout import LayoutClient
from console.task.channels.analog import Analog, OutputOptions


class Voltage(Analog):
    """
    Voltage channel type for NI analog write/output tasks.

    Note: This is the output variant without terminal_config support.

    Base kwargs:
        port (int): Physical port number
        min_val (float): Minimum voltage value
        max_val (float): Maximum voltage value
        custom_scale (str): "none", "Linear", "Map", "Table"
    """

    def __init__(
        self,
        layout: LayoutClient,
        name: str,
        device: str,
        **kwargs: Unpack[OutputOptions],
    ) -> None:
        """Initialize voltage output channel with configuration."""
        super().__init__(
            layout=layout,
            name=name,
            device=device,
            chan_type="Voltage",
            **kwargs,
        )
