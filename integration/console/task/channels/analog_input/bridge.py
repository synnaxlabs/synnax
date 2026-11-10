#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import Unpack

import synnax as sy
from console.layout import LayoutClient
from console.task.channels.analog import Analog, Options


class Bridge(Analog):
    """
    Bridge channel type for NI analog read tasks.

    Supported kwargs (in addition to Analog base kwargs):
        units (str): "mVoltsPerVolt" or "VoltsPerVolt"
        configuration (str): "FullBridge", "HalfBridge", or "QuarterBridge"
        resistance (float): Nominal bridge resistance
        excitation_source (str): "Internal", "External", or "None"
        excitation_value (float): Voltage excitation value

    Base kwargs from Analog:
        port (int): Physical port number
        terminal_config (str): "Cfg_Default", "Diff", etc.
        min_val (float): Minimum value
        max_val (float): Maximum value
        custom_scale (str): "none", "linear", "map", "table"
    """

    def __init__(
        self,
        layout: LayoutClient,
        name: str,
        device: str,
        *,
        units: sy.ni.ElectricalUnits | None = None,
        configuration: sy.ni.BridgeConfig | None = None,
        resistance: float | None = None,
        excitation_source: sy.ni.ExcitationSource | None = None,
        excitation_value: float | None = None,
        **kwargs: Unpack[Options],
    ) -> None:
        super().__init__(
            layout=layout,
            name=name,
            device=device,
            chan_type="Bridge",
            **kwargs,
        )

        self._configure_dropdown("Electrical units", units)
        self._configure_dropdown("Bridge configuration", configuration)
        self._configure_input("Nominal bridge resistance", resistance)
        self._configure_dropdown("Voltage excitation source", excitation_source)
        self._configure_input("Voltage excitation value", excitation_value)
