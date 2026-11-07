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


class StrainGauge(Analog):
    """
    Strain Gauge channel type for NI analog read tasks.

    Supported kwargs (in addition to Analog base kwargs):
        strain_configuration (str): "FullBridgeI", "FullBridgeII",
            "Full bridge III", "Half bridge I", "Half bridge II", "Quarter bridge I"
        excitation_source (str): "Internal", "External", "None"
        excitation_value (float): Voltage excitation value
        gage_factor (float): Gage factor value
        initial_bridge_voltage (float): Initial bridge voltage
        nominal_gage_resistance (float): Nominal gage resistance
        poisson_ratio (float): Poisson's ratio
        lead_wire_resistance (float): Lead wire resistance

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
        strain_configuration: sy.ni.StrainConfig | None = None,
        excitation_source: sy.ni.ExcitationSource | None = None,
        excitation_value: float | None = None,
        gage_factor: float | None = None,
        initial_bridge_voltage: float | None = None,
        nominal_gage_resistance: float | None = None,
        poisson_ratio: float | None = None,
        lead_wire_resistance: float | None = None,
        **kwargs: Unpack[Options],
    ) -> None:
        super().__init__(
            layout=layout,
            name=name,
            device=device,
            chan_type="Strain gauge",
            **kwargs,
        )

        self._configure_dropdown("Strain configuration", strain_configuration)
        self._configure_dropdown("Voltage excitation source", excitation_source)
        self._configure_input("Voltage excitation value", excitation_value)
        self._configure_input("Gage factor", gage_factor)
        self._configure_input("Initial bridge voltage", initial_bridge_voltage)
        self._configure_input("Nominal gage resistance", nominal_gage_resistance)
        self._configure_input("Poisson's ratio", poisson_ratio)
        self._configure_input("Lead wire resistance", lead_wire_resistance)
