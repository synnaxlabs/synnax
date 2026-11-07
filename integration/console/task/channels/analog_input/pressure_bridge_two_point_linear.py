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


class PressureBridgeTwoPointLinear(Analog):
    """
    Pressure Bridge Two-Point Linear channel type for NI analog read tasks.

    Supported kwargs (in addition to Analog base kwargs):
        pressure_units (str): "Pascals", "PoundsPerSquareInch", etc.
        bridge_configuration (str): "FullBridge", "HalfBridge", "QuarterBridge"
        resistance (float): Nominal bridge resistance
        excitation_source (str): "Internal", "External", "None"
        excitation_value (float): Voltage excitation value
        physical_units (str): "Pascals", "PoundsPerSquareInch", etc.
        electrical_units (str): "mVoltsPerVolt", "VoltsPerVolt"
        physical_value_one (float): Physical value one for scaling
        physical_value_two (float): Physical value two for scaling
        electrical_value_one (float): Electrical value one for scaling
        electrical_value_two (float): Electrical value two for scaling

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
        pressure_units: sy.ni.PressureUnits | None = None,
        bridge_configuration: sy.ni.BridgeConfig | None = None,
        resistance: float | None = None,
        excitation_source: sy.ni.ExcitationSource | None = None,
        excitation_value: float | None = None,
        physical_units: sy.ni.PressureUnits | None = None,
        electrical_units: sy.ni.ElectricalUnits | None = None,
        physical_value_one: float | None = None,
        physical_value_two: float | None = None,
        electrical_value_one: float | None = None,
        electrical_value_two: float | None = None,
        **kwargs: Unpack[Options],
    ) -> None:
        super().__init__(
            layout=layout,
            name=name,
            device=device,
            chan_type="Pressure bridge two-point linear",
            **kwargs,
        )

        self._configure_dropdown("Pressure units", pressure_units)
        self._configure_dropdown("Bridge configuration", bridge_configuration)
        self._configure_input("Nominal bridge resistance", resistance)
        self._configure_dropdown("Voltage excitation source", excitation_source)
        self._configure_input("Voltage excitation value", excitation_value)
        self._configure_dropdown("Physical units", physical_units)
        self._configure_dropdown("Electrical units", electrical_units)
        self._configure_input("Physical value one", physical_value_one)
        self._configure_input("Physical value two", physical_value_two)
        self._configure_input("Electrical value one", electrical_value_one)
        self._configure_input("Electrical value two", electrical_value_two)
