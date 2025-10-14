#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from .analog_read import AnalogRead
from .analog_write import AnalogWrite
from .channels.accelerometer import Accelerometer
from .channels.analog import Analog
from .channels.bridge import Bridge
from .channels.current import Current
from .channels.force_bridge_table import ForceBridgeTable
from .channels.force_bridge_two_point_linear import ForceBridgeTwoPointLinear
from .channels.force_iepe import ForceIEPE
from .channels.microphone import Microphone
from .channels.pressure_bridge_table import PressureBridgeTable
from .channels.pressure_bridge_two_point_linear import PressureBridgeTwoPointLinear
from .channels.resistance import Resistance
from .channels.rtd import RTD
from .channels.strain_gauge import StrainGauge
from .channels.temperature_built_in_sensor import TemperatureBuiltInSensor
from .channels.thermocouple import Thermocouple
from .channels.torque_bridge_table import TorqueBridgeTable
from .channels.torque_bridge_two_point_linear import TorqueBridgeTwoPointLinear
from .channels.velocity_iepe import VelocityIEPE
from .channels.voltage import Voltage
from .ni import NITask

__all__ = [
    "NITask",
    "AnalogRead",
    "AnalogWrite",
    "Analog",
    "Accelerometer",
    "Bridge",
    "Current",
    "ForceBridgeTable",
    "ForceBridgeTwoPointLinear",
    "ForceIEPE",
    "Microphone",
    "PressureBridgeTable",
    "PressureBridgeTwoPointLinear",
    "Resistance",
    "RTD",
    "StrainGauge",
    "TemperatureBuiltInSensor",
    "Thermocouple",
    "TorqueBridgeTable",
    "TorqueBridgeTwoPointLinear",
    "VelocityIEPE",
    "Voltage",
]
