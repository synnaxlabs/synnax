#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import json
from typing import Literal
from uuid import uuid4

from pydantic import BaseModel, Field, confloat, conint, field_validator, validator

from synnax import ValidationError
from synnax.hardware import device
from synnax.hardware.task import (
    BaseReadTaskConfig,
    BaseWriteTaskConfig,
    JSONConfigMixin,
    MetaTask,
    StarterStopperMixin,
    Task,
)
from synnax.telem import CrudeRate

UnitsVolts = Literal["Volts"]
UnitsAmps = Literal["Amps"]
UnitsDegF = Literal["DegF"]
UnitsDegC = Literal["DegC"]
UnitsDegR = Literal["DegR"]
UnitsKelvins = Literal["Kelvins"]
UnitsStrain = Literal["Strain"]
UnitsOhms = Literal["Ohms"]
UnitsHz = Literal["Hz"]
UnitsSeconds = Literal["Seconds"]
UnitsMeters = Literal["Meters"]
UnitsInches = Literal["Inches"]
UnitsDegAngle = Literal["Degrees"]
UnitsRadiansAngle = Literal["Radians"]
UnitsGravity = Literal["g"]
UnitsMetersPerSecondSquared = Literal["MetersPerSecondSquared"]
UnitsNewtons = Literal["Newtons"]
UnitsPounds = Literal["Pounds"]
UnitsKgForce = Literal["KilogramForce"]
UnitsLbsPerSquareInch = Literal["PoundsPerSquareInch"]
UnitsBar = Literal["Bar"]
UnitsPascals = Literal["Pascals"]
UnitsVoltsPerVolt = Literal["VoltsPerVolt"]
UnitsmVoltsPerVolt = Literal["mVoltsPerVolt"]
UnitsNewtonMeters = Literal["NewtonMeters"]
UnitsInchLbs = Literal["InchPounds"]
UnitsInOz = Literal["InchOunces"]
UnitsFtLbs = Literal["FootPounds"]
UnitsRPM = Literal["RPM"]
UnitsRadiansPerSecond = Literal["Radians/s"]
UnitsMetersPerSecond = Literal["MetersPerSecond"]
UnitsInchesPerSecond = Literal["InchesPerSecond"]

Units = (
    UnitsVolts
    | UnitsAmps
    | UnitsDegF
    | UnitsDegC
    | UnitsDegR
    | UnitsKelvins
    | UnitsStrain
    | UnitsOhms
    | UnitsHz
    | UnitsSeconds
    | UnitsMeters
    | UnitsInches
    | UnitsDegAngle
    | UnitsRadiansAngle
    | UnitsGravity
    | UnitsMetersPerSecondSquared
    | UnitsNewtons
    | UnitsPounds
    | UnitsKgForce
    | UnitsLbsPerSquareInch
    | UnitsBar
    | UnitsPascals
    | UnitsVoltsPerVolt
    | UnitsmVoltsPerVolt
    | UnitsNewtonMeters
    | UnitsInchLbs
    | UnitsInOz
    | UnitsFtLbs
    | UnitsRPM
    | UnitsRadiansPerSecond
    | UnitsMetersPerSecond
    | UnitsInchesPerSecond
)


class LinScale(BaseModel):
    """Custom linear scaling for analog input channels.

    For detailed information, see the NI-DAQmx documentation:
    <https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreatelinscale.html>
    """

    type: Literal["linear"] = "linear"
    slope: float
    y_intercept: float
    pre_scaled_units: Units
    scaled_units: Units


class MapScale(BaseModel):
    """Custom map scale for analog input channels.

    For detailed information, see the NI-DAQmx documentation:
    <https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreatemapscale.html>
    """

    type: Literal["map"] = "map"
    pre_scaled_min: float
    pre_scaled_max: float
    scaled_min: float
    scaled_max: float
    pre_scaled_units: Units


class TableScale(BaseModel):
    """Custom table scale for analog input channels.

    For detailed information, see the NI-DAQmx documentation:
    <https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreatetablescale.html>
    """

    type: Literal["table"] = "table"
    pre_scaled_vals: list[float]
    scaled_vals: list[float]
    pre_scaled_units: Units


class PolynomialScale(BaseModel):
    """Custom polynomial scale for analog input channels.

    For detailed information, see the NI-DAQmx documentation:
    <https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreatepoly3scale.html>
    """

    type: Literal["polynomial"] = "polynomial"
    forward_coeffs: list[float]
    reverse_coeffs: list[float]
    pre_scaled_units: Units
    scaled_units: Units


class NoScale(BaseModel):
    """Applies no scaling to the analog input channel. This is a default value that
    should rarely be used.
    """

    type: Literal["none"] = "none"


Scale = LinScale | MapScale | TableScale | NoScale
ScaleType = Literal["linear", "map", "table", "polynomial", "none"]
TerminalConfig = Literal["Cfg_Default", "RSE", "NRSE", "Diff", "PseudoDiff"]
ExcitationSource = Literal["Internal", "External", "None"]


class BaseChan(BaseModel):
    key: str
    enabled: bool = True

    def __init__(self, **data):
        if "key" not in data:
            data["key"] = str(uuid4())
        super().__init__(**data)


class BaseAIChan(BaseChan):
    device: str = ""
    port: int
    channel: int


class MinMaxVal(BaseModel):
    min_val: float = 0
    max_val: float = 1


class AIAccelChan(BaseAIChan, MinMaxVal):
    """
    Analog Input Accelerometer Channel

    Measures acceleration using IEPE (Integrated Electronics Piezo-Electric)
    accelerometers with built-in signal conditioning. Commonly used for vibration
    analysis, shock testing, and modal analysis.

    For detailed information, see the NI-DAQmx documentation:
    <https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaiaccelchan.html>

    Example:
        >>> # 100 mV/g accelerometer with internal current excitation
        >>> accel_chan = AIAccelChan(
        ...     port=0,
        ...     channel=0,
        ...     terminal_config="PseudoDiff",
        ...     sensitivity=100.0,  # 100 mV/g
        ...     sensitivity_units="mVoltsPerG",
        ...     units="g",
        ...     current_excit_source="Internal",
        ...     current_excit_val=0.004,  # 4 mA excitation
        ...     min_val=-50.0,
        ...     max_val=50.0
        ... )

    :param terminal_config: Input terminal configuration (Diff, PseudoDiff, RSE, NRSE)
    :param sensitivity: Sensor sensitivity value
    :param sensitivity_units: Sensitivity units (mVoltsPerG or VoltsPerG)
    :param units: Output units (g, MetersPerSecondSquared, InchesPerSecondSquared)
    :param current_excit_source: Excitation current source (Internal, External, or None)
    :param current_excit_val: Excitation current in amps (typically 0.002-0.004A)
    :param custom_scale: Optional custom scaling for the channel
    :param min_val: Minimum expected acceleration value
    :param max_val: Maximum expected acceleration value
    """

    type: Literal["ai_accel"] = "ai_accel"
    terminal_config: TerminalConfig = "Cfg_Default"
    sensitivity: float
    sensitivity_units: Literal["mVoltsPerG", "VoltsPerG"]
    units: Literal["g", "MetersPerSecondSquared", "InchesPerSecondSquared"]
    current_excit_source: ExcitationSource
    current_excit_val: float
    custom_scale: Scale = NoScale()


class AIAccel4WireDCVoltageChan(BaseAIChan, MinMaxVal):
    """
    Analog Input Accelerometer 4-Wire DC Voltage Channel

    Measures acceleration using DC-coupled accelerometers with 4-wire voltage
    excitation. The 4-wire configuration provides accurate excitation voltage
    at the sensor by compensating for lead resistance.

    For detailed information, see the NI-DAQmx documentation:
    <https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaiaccel4wiredcvoltagechan.html>

    Example:
        >>> # DC accelerometer with 4-wire voltage excitation
        >>> accel_4w_chan = AIAccel4WireDCVoltageChan(
        ...     port=0,
        ...     channel=1,
        ...     terminal_config="Diff",
        ...     sensitivity=50.0,  # 50 mV/g
        ...     sensitivity_units="mVoltsPerG",
        ...     units="g",
        ...     voltage_excit_source="Internal",
        ...     voltage_excit_val=5.0,  # 5V excitation
        ...     use_excit_for_scaling=True,
        ...     min_val=-10.0,
        ...     max_val=10.0
        ... )

    :param terminal_config: Input terminal configuration
    :param sensitivity: Sensor sensitivity value
    :param sensitivity_units: Sensitivity units (mVoltsPerG or VoltsPerG)
    :param units: Output units (g, MetersPerSecondSquared, InchesPerSecondSquared)
    :param voltage_excit_source: Voltage excitation source (Internal, External, or None)
    :param voltage_excit_val: Excitation voltage in volts
    :param use_excit_for_scaling: Whether to use excitation voltage for scaling
    :param custom_scale: Optional custom scaling
    :param min_val: Minimum expected acceleration
    :param max_val: Maximum expected acceleration
    """

    type: Literal["ai_accel_4_wire_dc_voltage"] = "ai_accel_4_wire_dc_voltage"
    terminal_config: TerminalConfig = "Cfg_Default"
    sensitivity: float
    sensitivity_units: Literal["mVoltsPerG", "VoltsPerG"]
    units: Literal["g", "MetersPerSecondSquared", "InchesPerSecondSquared"]
    voltage_excit_source: ExcitationSource
    voltage_excit_val: float
    use_excit_for_scaling: bool
    custom_scale: Scale = NoScale()


class AIAccelChargeChan(BaseAIChan, MinMaxVal):
    """
    Analog Input Accelerometer Charge Channel

    Measures acceleration using charge-mode piezoelectric accelerometers.
    Charge-mode sensors are ideal for high-temperature applications and
    provide excellent low-frequency response. Requires charge amplifier hardware.

    For detailed information, see the NI-DAQmx documentation:
    <https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaiaccelchargechan.html>

    Example:
        >>> # Charge-mode accelerometer for high temperature
        >>> accel_charge_chan = AIAccelChargeChan(
        ...     port=0,
        ...     channel=2,
        ...     units="g",
        ...     min_val=-100.0,
        ...     max_val=100.0
        ... )

    :param units: Output units (g, MetersPerSecondSquared, InchesPerSecondSquared)
    :param custom_scale: Optional custom scaling
    :param min_val: Minimum expected acceleration
    :param max_val: Maximum expected acceleration
    """

    type: Literal["ai_accel_charge"] = "ai_accel_charge"
    units: Literal["g", "MetersPerSecondSquared", "InchesPerSecondSquared"]
    custom_scale: Scale = NoScale()


class AIBridgeChan(BaseAIChan, MinMaxVal):
    """
    Analog Input Bridge Channel

    Measures voltage output from a Wheatstone bridge circuit without applying
    physical unit scaling. Outputs in volts per volt (V/V). Use this for custom
    bridge sensors or when you want raw bridge output before applying calibration.

    For detailed information, see the NI-DAQmx documentation:
    <https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaibridgechan.html>

    Example:
        >>> # Generic full-bridge sensor with 10V excitation
        >>> bridge_chan = AIBridgeChan(
        ...     port=0,
        ...     channel=0,
        ...     terminal_config="Diff",
        ...     units="VoltsPerVolt",
        ...     bridge_config="FullBridge",
        ...     voltage_excit_source="Internal",
        ...     voltage_excit_val=10.0,
        ...     nominal_bridge_resistance=350.0,  # 350 ohm bridge
        ...     min_val=-0.01,  # ±10 mV/V
        ...     max_val=0.01
        ... )

    :param terminal_config: Input terminal configuration
    :param units: Output units (mVoltsPerVolt or VoltsPerVolt)
    :param bridge_config: Bridge type (FullBridge, HalfBridge, QuarterBridge)
    :param voltage_excit_source: Excitation voltage source
    :param voltage_excit_val: Excitation voltage in volts
    :param nominal_bridge_resistance: Bridge resistance in ohms at null condition
    :param custom_scale: Optional custom scaling
    :param min_val: Minimum expected bridge output
    :param max_val: Maximum expected bridge output
    """

    type: Literal["ai_bridge"] = "ai_bridge"
    terminal_config: TerminalConfig = "Cfg_Default"
    units: Literal["mVoltsPerVolt", "VoltsPerVolt"]
    bridge_config: Literal["FullBridge", "HalfBridge", "QuarterBridge"]
    voltage_excit_source: ExcitationSource
    voltage_excit_val: float
    nominal_bridge_resistance: confloat(gt=0)
    custom_scale: Scale = NoScale()


class AIChargeChan(BaseAIChan, MinMaxVal):
    """
    Analog Input Charge Channel

    Measures electrical charge from piezoelectric sensors. Commonly used with
    piezoelectric force sensors, pressure sensors, and other charge-output
    transducers. Requires charge amplifier hardware.

    For detailed information, see the NI-DAQmx documentation:
    <https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaichargechan.html>

    Example:
        >>> # Piezoelectric sensor measuring in Coulombs
        >>> charge_chan = AIChargeChan(
        ...     port=0,
        ...     channel=0,
        ...     terminal_config="Diff",
        ...     units="C",  # Coulombs
        ...     min_val=-0.000001,  # -1 µC
        ...     max_val=0.000001    # +1 µC
        ... )

    :param terminal_config: Input terminal configuration
    :param units: Charge units (C for Coulombs, uC for microCoulombs)
    :param custom_scale: Optional custom scaling
    :param min_val: Minimum expected charge
    :param max_val: Maximum expected charge
    """

    type: Literal["ai_charge"] = "ai_charge"
    terminal_config: TerminalConfig = "Cfg_Default"
    units: Literal["C", "uC"]
    custom_scale: Scale = NoScale()


class AICurrentChan(BaseAIChan, MinMaxVal):
    """
    Analog Input Current Channel

    Measures current from 4-20mA current loop sensors or other current-output
    transducers. Uses a shunt resistor to convert current to voltage.
    Common in industrial process control applications.

    For detailed information, see the NI-DAQmx documentation:
    <https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaicurrentchan.html>

    Example:
        >>> # 4-20 mA current loop sensor
        >>> current_chan = AICurrentChan(
        ...     port=0,
        ...     channel=0,
        ...     terminal_config="Diff",
        ...     units="Amps",
        ...     shunt_resistor_loc="Internal",  # Use internal shunt
        ...     ext_shunt_resistor_val=249.0,  # External shunt value if used
        ...     min_val=0.004,  # 4 mA
        ...     max_val=0.020   # 20 mA
        ... )

    :param terminal_config: Input terminal configuration
    :param units: Output units (Amps)
    :param shunt_resistor_loc: Shunt resistor location (Default, Internal, External)
    :param ext_shunt_resistor_val: External shunt resistor value in ohms
    :param custom_scale: Optional custom scaling
    :param min_val: Minimum expected current
    :param max_val: Maximum expected current
    """

    type: Literal["ai_current"] = "ai_current"
    terminal_config: TerminalConfig = "Cfg_Default"
    units: Literal["Amps"] = "Amps"
    shunt_resistor_loc: Literal["Default", "Internal", "External"]
    ext_shunt_resistor_val: confloat(gt=0)
    custom_scale: Scale = NoScale()


class AICurrentRMSChan(BaseAIChan, MinMaxVal):
    """
    Analog Input Current RMS Channel

    Measures root-mean-square (RMS) value of AC current signals. Useful for
    measuring AC power consumption or monitoring AC motor current.
    Automatically calculates RMS over the specified measurement period.

    For detailed information, see the NI-DAQmx documentation:
    <https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaicurrentrmschan.html>

    Example:
        >>> # AC current measurement (RMS)
        >>> current_rms_chan = AICurrentRMSChan(
        ...     port=0,
        ...     channel=0,
        ...     terminal_config="Diff",
        ...     units="Amps",
        ...     shunt_resistor_loc="Internal",
        ...     ext_shunt_resistor_val=249.0,
        ...     min_val=0.0,
        ...     max_val=10.0  # 0-10A RMS
        ... )

    :param terminal_config: Input terminal configuration
    :param units: Output units (Amps)
    :param shunt_resistor_loc: Shunt resistor location (Default, Internal, External)
    :param ext_shunt_resistor_val: External shunt resistor value in ohms
    :param custom_scale: Optional custom scaling
    :param min_val: Minimum expected RMS current
    :param max_val: Maximum expected RMS current
    """

    type: Literal["ai_current_rms"] = "ai_current_rms"
    terminal_config: TerminalConfig = "Cfg_Default"
    units: Literal["Amps"] = "Amps"
    shunt_resistor_loc: Literal["Default", "Internal", "External"]
    ext_shunt_resistor_val: confloat(gt=0)
    custom_scale: Scale = NoScale()


class AIForceBridgePolynomialChan(BaseAIChan, MinMaxVal):
    """
    Analog Input Force Bridge Polynomial Channel

    Measures force using a Wheatstone bridge load cell with polynomial scaling.
    Polynomial coefficients convert bridge output voltage to force units.
    Use this when your load cell comes with polynomial calibration data.

    For detailed information, see the NI-DAQmx documentation:
    <https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaiforcebridgepolynomialchan.html>

    Example:
        >>> # Load cell with polynomial calibration
        >>> force_poly_chan = AIForceBridgePolynomialChan(
        ...     port=0,
        ...     channel=0,
        ...     terminal_config="Diff",
        ...     units="Newtons",
        ...     bridge_config="FullBridge",
        ...     voltage_excit_source="Internal",
        ...     voltage_excit_val=10.0,
        ...     nominal_bridge_resistance=350.0,
        ...     forward_coeffs=[0.0, 1000.0, 0.1, 0.001],  # Calibration coefficients
        ...     reverse_coeffs=[0.0, 0.001, -0.0001],
        ...     electrical_units="mVoltsPerVolt",
        ...     physical_units="Newtons",
        ...     min_val=-500.0,
        ...     max_val=500.0
        ... )

    :param terminal_config: Input terminal configuration
    :param units: Output force units (Newtons, Pounds, KilogramForce)
    :param bridge_config: Bridge type (FullBridge, HalfBridge, QuarterBridge)
    :param voltage_excit_source: Excitation voltage source
    :param voltage_excit_val: Excitation voltage in volts
    :param nominal_bridge_resistance: Bridge resistance in ohms
    :param forward_coeffs: Forward polynomial coefficients (electrical to physical)
    :param reverse_coeffs: Reverse polynomial coefficients (physical to electrical)
    :param electrical_units: Bridge output units (mVoltsPerVolt or VoltsPerVolt)
    :param physical_units: Force units matching the units parameter
    :param custom_scale: Optional custom scaling
    :param min_val: Minimum expected force
    :param max_val: Maximum expected force
    """

    type: Literal["ai_force_bridge_polynomial"] = "ai_force_bridge_polynomial"
    terminal_config: TerminalConfig = "Cfg_Default"
    units: Literal["Newtons", "Pounds", "KilogramForce"]
    bridge_config: Literal["FullBridge", "HalfBridge", "QuarterBridge"]
    voltage_excit_source: ExcitationSource
    voltage_excit_val: float
    nominal_bridge_resistance: float
    forward_coeffs: list[float]
    reverse_coeffs: list[float]
    electrical_units: Literal["mVoltsPerVolt", "VoltsPerVolt"]
    physical_units: Literal["Newtons", "Pounds", "KilogramForce"]
    custom_scale: Scale = NoScale()


class AIForceBridgeTableChan(BaseAIChan, MinMaxVal):
    """
    Analog Input Force Bridge Table Channel

    Measures force using a Wheatstone bridge load cell with table-based scaling.
    Maps electrical values to physical force values using lookup table interpolation.
    Ideal when you have discrete calibration points from load cell testing.

    For detailed information, see the NI-DAQmx documentation:
    <https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaiforcebridgetablechan.html>

    Example:
        >>> # Load cell with calibration table
        >>> force_table_chan = AIForceBridgeTableChan(
        ...     port=0,
        ...     channel=0,
        ...     units="Pounds",
        ...     bridge_config="FullBridge",
        ...     voltage_excit_source="Internal",
        ...     voltage_excit_val=10.0,
        ...     nominal_bridge_resistance=350.0,
        ...     electrical_units="mVoltsPerVolt",
        ...     electrical_vals=[-2.0, -1.0, 0.0, 1.0, 2.0],  # mV/V readings
        ...     physical_units="Pounds",
        ...     physical_vals=[-100.0, -50.0, 0.0, 50.0, 100.0],  # Force values
        ...     min_val=-100.0,
        ...     max_val=100.0
        ... )

    :param units: Output force units (Newtons, Pounds, KilogramForce)
    :param bridge_config: Bridge type (FullBridge, HalfBridge, QuarterBridge)
    :param voltage_excit_source: Excitation voltage source
    :param voltage_excit_val: Excitation voltage in volts
    :param nominal_bridge_resistance: Bridge resistance in ohms
    :param electrical_units: Bridge output units (mVoltsPerVolt or VoltsPerVolt)
    :param electrical_vals: List of electrical calibration point values
    :param physical_units: Force units matching the units parameter
    :param physical_vals: List of corresponding force values
    :param custom_scale: Optional custom scaling
    :param min_val: Minimum expected force
    :param max_val: Maximum expected force
    """

    type: Literal["ai_force_bridge_table"] = "ai_force_bridge_table"
    units: Literal["Newtons", "Pounds", "KilogramForce"]
    bridge_config: Literal["FullBridge", "HalfBridge", "QuarterBridge"]
    voltage_excit_source: ExcitationSource
    voltage_excit_val: float
    nominal_bridge_resistance: float
    electrical_units: Literal["mVoltsPerVolt", "VoltsPerVolt"]
    electrical_vals: list[float]
    physical_units: Literal["Newtons", "Pounds", "KilogramForce"]
    physical_vals: list[float]
    custom_scale: Scale = NoScale()


class AIForceBridgeTwoPointLinChan(BaseAIChan, MinMaxVal):
    """
    Analog Input Force Bridge Two Point Linear Channel

    Measures force using a Wheatstone bridge load cell with two-point linear scaling.
    Defines a linear calibration using two known force/voltage pairs.
    Simple and effective for load cells with linear characteristics.

    For detailed information, see the NI-DAQmx documentation:
    <https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaiforcebridgetwopointlinchan.html>

    Example:
        >>> # Load cell with two-point calibration
        >>> force_2pt_chan = AIForceBridgeTwoPointLinChan(
        ...     port=0,
        ...     channel=0,
        ...     terminal_config="Diff",
        ...     units="Newtons",
        ...     bridge_config="FullBridge",
        ...     voltage_excit_source="Internal",
        ...     voltage_excit_val=10.0,
        ...     nominal_bridge_resistance=350.0,
        ...     electrical_units="mVoltsPerVolt",
        ...     physical_units="Newtons",
        ...     first_electrical_val=0.0,   # No load = 0 mV/V
        ...     first_physical_val=0.0,     # 0 N
        ...     second_electrical_val=2.0,  # Full scale = 2 mV/V
        ...     second_physical_val=1000.0, # 1000 N
        ...     min_val=-100.0,
        ...     max_val=1100.0
        ... )

    :param terminal_config: Input terminal configuration
    :param units: Output force units (Newtons, Pounds, KilogramForce)
    :param bridge_config: Bridge type (FullBridge, HalfBridge, QuarterBridge)
    :param voltage_excit_source: Excitation voltage source
    :param voltage_excit_val: Excitation voltage in volts
    :param nominal_bridge_resistance: Bridge resistance in ohms
    :param electrical_units: Bridge output units (mVoltsPerVolt or VoltsPerVolt)
    :param physical_units: Force units matching the units parameter
    :param first_electrical_val: First calibration point electrical value
    :param first_physical_val: First calibration point force value
    :param second_electrical_val: Second calibration point electrical value
    :param second_physical_val: Second calibration point force value
    :param custom_scale: Optional custom scaling
    :param min_val: Minimum expected force
    :param max_val: Maximum expected force
    """

    type: Literal["ai_force_bridge_two_point_lin"] = "ai_force_bridge_two_point_lin"
    terminal_config: TerminalConfig = "Cfg_Default"
    units: Literal["Newtons", "Pounds", "KilogramForce"]
    bridge_config: Literal["FullBridge", "HalfBridge", "QuarterBridge"]
    voltage_excit_source: ExcitationSource
    voltage_excit_val: float
    nominal_bridge_resistance: float
    electrical_units: Literal["mVoltsPerVolt", "VoltsPerVolt"]
    physical_units: Literal["Newtons", "Pounds", "KilogramForce"]
    first_electrical_val: float
    first_physical_val: float
    second_electrical_val: float
    second_physical_val: float
    custom_scale: Scale = NoScale()


class AIForceIEPEChan(BaseAIChan, MinMaxVal):
    """
    Analog Input Force IEPE Channel

    Measures force using IEPE (Integrated Electronics Piezo-Electric) force sensors
    with built-in electronics. Provides current excitation and measures dynamic
    force events. Ideal for impact testing and dynamic force measurements.

    For detailed information, see the NI-DAQmx documentation:
    <https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaiforceiepechan.html>

    Example:
        >>> # IEPE force sensor for impact testing
        >>> force_iepe_chan = AIForceIEPEChan(
        ...     port=0,
        ...     channel=0,
        ...     terminal_config="PseudoDiff",
        ...     units="Newtons",
        ...     sensitivity=10.0,  # 10 mV/N
        ...     sensitivity_units="mVoltsPerVolt",
        ...     current_excit_source="Internal",
        ...     current_excit_val=0.004,  # 4 mA
        ...     min_val=-5000.0,
        ...     max_val=5000.0
        ... )

    :param terminal_config: Input terminal configuration
    :param units: Output force units (Newtons, Pounds, KilogramForce)
    :param sensitivity: Sensor sensitivity value
    :param sensitivity_units: Sensitivity units (mVoltsPerVolt or VoltsPerVolt)
    :param current_excit_source: Excitation current source (Internal, External, or None)
    :param current_excit_val: Excitation current in amps (typically 2-4 mA)
    :param custom_scale: Optional custom scaling
    :param min_val: Minimum expected force
    :param max_val: Maximum expected force
    """

    type: Literal["ai_force_iepe"] = "ai_force_iepe"
    terminal_config: TerminalConfig = "Cfg_Default"
    units: Literal["Newtons", "Pounds", "KilogramForce"]
    sensitivity: float
    sensitivity_units: Literal["mVoltsPerVolt", "VoltsPerVolt"]
    current_excit_source: ExcitationSource
    current_excit_val: float
    custom_scale: Scale = NoScale()


class AIFreqVoltageChan(BaseAIChan, MinMaxVal):
    """
    Analog Input Frequency Voltage Channel

    Measures frequency of an analog voltage signal by detecting zero-crossings
    or threshold crossings. Useful for measuring AC line frequency, VFD output
    frequency, or other varying voltage signals.

    For detailed information, see the NI-DAQmx documentation:
    <https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaifreqvoltagechan.html>

    Example:
        >>> # Measure AC line frequency (50/60 Hz)
        >>> freq_voltage_chan = AIFreqVoltageChan(
        ...     port=0,
        ...     channel=0,
        ...     units="Hz",
        ...     threshold_level=0.0,  # Zero-crossing detection
        ...     hysteresis=0.1,       # 100mV hysteresis
        ...     min_val=45.0,
        ...     max_val=65.0
        ... )

    :param units: Output units (Hz for frequency)
    :param threshold_level: Voltage threshold for frequency detection
    :param hysteresis: Hysteresis voltage to prevent false triggering
    :param custom_scale: Optional custom scaling
    :param min_val: Minimum expected frequency
    :param max_val: Maximum expected frequency
    """

    type: Literal["ai_freq_voltage"] = "ai_freq_voltage"
    units: Literal["Hz"] = "Hz"
    threshold_level: float
    hysteresis: float
    custom_scale: Scale = NoScale()


class AIMicrophoneChan(BaseAIChan):
    """
    Analog Input Microphone Channel

    Measures sound pressure using IEPE microphones with built-in preamplifiers.
    Provides current excitation and converts acoustic pressure to electrical signal.
    Common in noise analysis, acoustic testing, and sound level measurements.

    For detailed information, see the NI-DAQmx documentation:
    <https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaimicrophonechan.html>

    Example:
        >>> # IEPE microphone for acoustic measurements
        >>> microphone_chan = AIMicrophoneChan(
        ...     port=0,
        ...     channel=0,
        ...     terminal_config="PseudoDiff",
        ...     mic_sensitivity=50.0,  # 50 mV/Pa
        ...     max_snd_press_level=140.0,  # 140 dB SPL max
        ...     current_excit_source="Internal",
        ...     current_excit_val=0.004,  # 4 mA
        ...     units="Pascals"
        ... )

    :param terminal_config: Input terminal configuration
    :param mic_sensitivity: Microphone sensitivity in mV/Pa
    :param max_snd_press_level: Maximum sound pressure level in dB SPL
    :param current_excit_source: Excitation current source (Internal, External, or None)
    :param current_excit_val: Excitation current in amps (typically 2-4 mA)
    :param units: Output units (Pascals)
    :param custom_scale: Optional custom scaling
    """

    type: Literal["ai_microphone"] = "ai_microphone"
    terminal_config: TerminalConfig = "Cfg_Default"
    mic_sensitivity: float
    max_snd_press_level: float
    current_excit_source: ExcitationSource
    current_excit_val: float
    units: Literal["Pascals"] = "Pascals"
    custom_scale: Scale = NoScale()


class AIPressureBridgePolynomialChan(BaseAIChan, MinMaxVal):
    """
    Analog Input Pressure Bridge Polynomial Channel

    Measures pressure using a Wheatstone bridge pressure sensor with polynomial
    scaling. Polynomial coefficients convert bridge output to pressure units.
    Use when your pressure transducer has polynomial calibration data.

    For detailed information, see the NI-DAQmx documentation:
    <https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaipressurebridgepolynomialchan.html>

    Example:
        >>> # Pressure sensor with polynomial calibration
        >>> pressure_poly_chan = AIPressureBridgePolynomialChan(
        ...     port=0,
        ...     channel=0,
        ...     units="PoundsPerSquareInch",
        ...     bridge_config="FullBridge",
        ...     voltage_excit_source="Internal",
        ...     voltage_excit_val=10.0,
        ...     nominal_bridge_resistance=350.0,
        ...     forward_coeffs=[0.0, 100.0, 0.01],  # Calibration coefficients
        ...     reverse_coeffs=[0.0, 0.01],
        ...     electrical_units="mVoltsPerVolt",
        ...     physical_units="PoundsPerSquareInch",
        ...     min_val=0.0,
        ...     max_val=1000.0
        ... )

    :param units: Output pressure units (PoundsPerSquareInch, Pascals, Bar)
    :param bridge_config: Bridge type (FullBridge, HalfBridge, QuarterBridge)
    :param voltage_excit_source: Excitation voltage source
    :param voltage_excit_val: Excitation voltage in volts
    :param nominal_bridge_resistance: Bridge resistance in ohms
    :param forward_coeffs: Forward polynomial coefficients
    :param reverse_coeffs: Reverse polynomial coefficients
    :param electrical_units: Bridge output units (mVoltsPerVolt or VoltsPerVolt)
    :param physical_units: Pressure units matching the units parameter
    :param custom_scale: Optional custom scaling
    :param min_val: Minimum expected pressure
    :param max_val: Maximum expected pressure
    """

    type: Literal["ai_pressure_bridge_polynomial"] = "ai_pressure_bridge_polynomial"
    units: Literal["PoundsPerSquareInch", "Pascals", "Bar"]
    bridge_config: Literal["FullBridge", "HalfBridge", "QuarterBridge"]
    voltage_excit_source: ExcitationSource
    voltage_excit_val: float
    nominal_bridge_resistance: float
    forward_coeffs: list[float]
    reverse_coeffs: list[float]
    electrical_units: Literal["mVoltsPerVolt", "VoltsPerVolt"]
    physical_units: Literal["PoundsPerSquareInch", "Pascals", "Bar"]
    custom_scale: Scale = NoScale()


class AIPressureBridgeTableChan(BaseAIChan, MinMaxVal):
    """
    Analog Input Pressure Bridge Table Channel

    Measures pressure using a Wheatstone bridge sensor with table-based scaling.
    Maps electrical values to pressure using lookup table interpolation.
    Ideal when you have discrete calibration points from pressure sensor testing.

    For detailed information, see the NI-DAQmx documentation:
    <https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaipressurebridgetablechan.html>

    Example:
        >>> # Pressure sensor with calibration table
        >>> pressure_table_chan = AIPressureBridgeTableChan(
        ...     port=0,
        ...     channel=0,
        ...     units="Bar",
        ...     bridge_config="FullBridge",
        ...     voltage_excit_source="Internal",
        ...     voltage_excit_val=10.0,
        ...     nominal_bridge_resistance=350.0,
        ...     electrical_units="mVoltsPerVolt",
        ...     electrical_vals=[0.0, 0.5, 1.0, 1.5, 2.0],
        ...     physical_units="Bar",
        ...     physical_vals=[0.0, 25.0, 50.0, 75.0, 100.0],
        ...     min_val=0.0,
        ...     max_val=100.0
        ... )

    :param units: Output pressure units (PoundsPerSquareInch, Pascals, Bar)
    :param bridge_config: Bridge type (FullBridge, HalfBridge, QuarterBridge)
    :param voltage_excit_source: Excitation voltage source
    :param voltage_excit_val: Excitation voltage in volts
    :param nominal_bridge_resistance: Bridge resistance in ohms
    :param electrical_units: Bridge output units (mVoltsPerVolt or VoltsPerVolt)
    :param electrical_vals: List of electrical calibration point values
    :param physical_units: Pressure units matching the units parameter
    :param physical_vals: List of corresponding pressure values
    :param custom_scale: Optional custom scaling
    :param min_val: Minimum expected pressure
    :param max_val: Maximum expected pressure
    """

    type: Literal["ai_pressure_bridge_table"] = "ai_pressure_bridge_table"
    units: Literal["PoundsPerSquareInch", "Pascals", "Bar"]
    bridge_config: Literal["FullBridge", "HalfBridge", "QuarterBridge"]
    voltage_excit_source: ExcitationSource
    voltage_excit_val: float
    nominal_bridge_resistance: float
    electrical_units: Literal["mVoltsPerVolt", "VoltsPerVolt"]
    electrical_vals: list[float]
    physical_units: Literal["PoundsPerSquareInch", "Pascals", "Bar"]
    physical_vals: list[float]
    custom_scale: Scale = NoScale()


class AIPressureBridgeTwoPointLinChan(BaseAIChan, MinMaxVal):
    """
    Analog Input Pressure Bridge Two Point Linear Channel

    Measures pressure using a Wheatstone bridge sensor with two-point linear scaling.
    Defines linear calibration using two known pressure/voltage pairs.
    Simple and effective for pressure transducers with linear response.

    For detailed information, see the NI-DAQmx documentation:
    <https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaipressurebridgetwopointlinchan.html>

    Example:
        >>> # Pressure sensor with two-point calibration
        >>> pressure_2pt_chan = AIPressureBridgeTwoPointLinChan(
        ...     port=0,
        ...     channel=0,
        ...     terminal_config="Diff",
        ...     units="Pascals",
        ...     bridge_config="FullBridge",
        ...     voltage_excit_source="Internal",
        ...     voltage_excit_val=10.0,
        ...     nominal_bridge_resistance=350.0,
        ...     electrical_units="mVoltsPerVolt",
        ...     physical_units="Pascals",
        ...     first_electrical_val=0.0,
        ...     first_physical_val=0.0,      # 0 Pa at 0 mV/V
        ...     second_electrical_val=2.0,
        ...     second_physical_val=100000.0, # 100 kPa at 2 mV/V
        ...     min_val=-10000.0,
        ...     max_val=110000.0
        ... )

    :param terminal_config: Input terminal configuration
    :param units: Output pressure units (PoundsPerSquareInch, Pascals, Bar)
    :param bridge_config: Bridge type (FullBridge, HalfBridge, QuarterBridge)
    :param voltage_excit_source: Excitation voltage source
    :param voltage_excit_val: Excitation voltage in volts
    :param nominal_bridge_resistance: Bridge resistance in ohms
    :param electrical_units: Bridge output units (mVoltsPerVolt or VoltsPerVolt)
    :param physical_units: Pressure units matching the units parameter
    :param first_electrical_val: First calibration point electrical value
    :param first_physical_val: First calibration point pressure value
    :param second_electrical_val: Second calibration point electrical value
    :param second_physical_val: Second calibration point pressure value
    :param custom_scale: Optional custom scaling
    :param min_val: Minimum expected pressure
    :param max_val: Maximum expected pressure
    """

    type: Literal["ai_pressure_bridge_two_point_lin"] = (
        "ai_pressure_bridge_two_point_lin"
    )
    terminal_config: TerminalConfig = "Cfg_Default"
    units: Literal["PoundsPerSquareInch", "Pascals", "Bar"]
    bridge_config: Literal["FullBridge", "HalfBridge", "QuarterBridge"]
    voltage_excit_source: ExcitationSource
    voltage_excit_val: float
    nominal_bridge_resistance: float
    electrical_units: Literal["mVoltsPerVolt", "VoltsPerVolt"]
    physical_units: Literal["PoundsPerSquareInch", "Pascals", "Bar"]
    first_electrical_val: float
    first_physical_val: float
    second_electrical_val: float
    second_physical_val: float
    custom_scale: Scale = NoScale()


class AIResistanceChan(BaseAIChan, MinMaxVal):
    """
    Analog Input Resistance Channel

    Measures electrical resistance using 2-wire, 3-wire, or 4-wire configurations.
    Applies excitation current and measures resulting voltage to calculate resistance.
    Common for custom resistive sensors or resistance monitoring applications.

    For detailed information, see the NI-DAQmx documentation:
    <https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateairesistancechan.html>

    Example:
        >>> # 4-wire resistance measurement
        >>> resistance_chan = AIResistanceChan(
        ...     port=0,
        ...     channel=0,
        ...     terminal_config="Diff",
        ...     units="Ohms",
        ...     resistance_config="4Wire",  # Most accurate
        ...     current_excit_source="Internal",
        ...     current_excit_val=0.001,  # 1 mA excitation
        ...     min_val=0.0,
        ...     max_val=1000.0
        ... )

    :param terminal_config: Input terminal configuration
    :param units: Output units (Ohms)
    :param resistance_config: Wire configuration (2Wire, 3Wire, or 4Wire)
    :param current_excit_source: Excitation current source (Internal, External, or None)
    :param current_excit_val: Excitation current in amps
    :param custom_scale: Optional custom scaling
    :param min_val: Minimum expected resistance
    :param max_val: Maximum expected resistance
    """

    type: Literal["ai_resistance"] = "ai_resistance"
    terminal_config: TerminalConfig = "Cfg_Default"
    units: Literal["Ohms"] = "Ohms"
    resistance_config: Literal["2Wire", "3Wire", "4Wire"]
    current_excit_source: ExcitationSource
    current_excit_val: float
    custom_scale: Scale = NoScale()


class AIRosetteStrainGageChan(BaseAIChan, MinMaxVal):
    """
    Analog Input Rosette Strain Gage Channel

    Measures strain using a rosette strain gage configuration with multiple elements
    arranged at specific angles. Calculates principal strains, shear strains, and
    strain directions. Used for complex stress analysis where strain direction is unknown.

    For detailed information, see the NI-DAQmx documentation:
    <https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateairosettestraingagechan.html>

    Example:
        >>> # Rectangular rosette for principal strain analysis
        >>> rosette_chan = AIRosetteStrainGageChan(
        ...     port=0,
        ...     channel=0,
        ...     terminal_config="Diff",
        ...     rosette_type="RectangularRosette",
        ...     gage_orientation=0.0,  # degrees
        ...     rosette_meas_types=["PrincipleStrain1", "PrincipleStrain2", "MaxShearStrain"],
        ...     strain_config="QuarterBridgeI",
        ...     units="strain",
        ...     voltage_excit_source="Internal",
        ...     voltage_excit_val=2.5,
        ...     nominal_gage_resistance=350.0,
        ...     poisson_ratio=0.3,
        ...     lead_wire_resistance=0.0,
        ...     gage_factor=2.1,
        ...     min_val=-0.01,
        ...     max_val=0.01
        ... )

    :param terminal_config: Input terminal configuration
    :param rosette_type: Rosette configuration (RectangularRosette, DeltaRosette, TeeRosette)
    :param gage_orientation: Orientation of gage element 0 in degrees
    :param rosette_meas_types: List of measurement types to calculate
    :param strain_config: Bridge configuration for each gage element
    :param units: Output units (strain)
    :param voltage_excit_source: Excitation voltage source
    :param voltage_excit_val: Excitation voltage in volts
    :param nominal_gage_resistance: Gage resistance in ohms
    :param poisson_ratio: Poisson's ratio of test material
    :param lead_wire_resistance: Lead wire resistance in ohms
    :param gage_factor: Gage factor (typically 2.0-2.1)
    :param min_val: Minimum expected strain
    :param max_val: Maximum expected strain
    """

    type: Literal["ai_rosette_strain_gage"] = "ai_rosette_strain_gage"
    terminal_config: TerminalConfig = "Cfg_Default"
    rosette_type: Literal["RectangularRosette", "DeltaRosette", "TeeRosette"]
    gage_orientation: float
    rosette_meas_types: list[
        Literal[
            "PrincipleStrain1",
            "PrincipleStrain2",
            "PrincipleStrainAngle",
            "CartesianStrainX",
            "CartesianStrainY",
            "CartesianShearStrainXY",
            "MaxShearStrain",
            "MaxShearStrainAngle",
        ]
    ]
    strain_config: Literal[
        "FullBridgeI",
        "FullBridgeII",
        "FullBridgeIII",
        "HalfBridgeI",
        "HalfBridgeII",
        "QuarterBridgeI",
        "QuarterBridgeII",
    ]
    units: Literal["strain"] = "strain"
    voltage_excit_source: ExcitationSource
    voltage_excit_val: float
    nominal_gage_resistance: float
    poisson_ratio: float
    lead_wire_resistance: float
    gage_factor: float


class AIRTDChan(BaseAIChan, MinMaxVal):
    """
    Analog Input RTD (Resistance Temperature Detector) Channel

    Measures temperature using RTD sensors with configurable platinum curves
    and wire configurations. Supports 2-wire, 3-wire, and 4-wire RTD connections
    for varying accuracy requirements.

    For detailed information, see the NI-DAQmx documentation:
    <https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateairtdchan.html>

    Example:
        >>> # PT100 sensor in 3-wire configuration
        >>> rtd_chan = AIRTDChan(
        ...     port=0,
        ...     channel=1,
        ...     units="DegC",
        ...     rtd_type="Pt3851",
        ...     resistance_config="3Wire",
        ...     current_excit_source="Internal",
        ...     current_excit_val=0.001,  # 1mA excitation
        ...     r0=100.0,  # 100 ohm at 0°C
        ...     min_val=0.0,
        ...     max_val=100.0
        ... )

    :param units: Temperature units for measurement output
    :param rtd_type: RTD curve type (commonly Pt3851 for industrial RTDs)
    :param resistance_config: Wire configuration - 4Wire most accurate, 2Wire simplest
    :param current_excit_source: Excitation current source (Internal or External)
    :param current_excit_val: Excitation current in amps (typically 0.001A)
    :param r0: RTD resistance at 0°C in ohms (100.0 for PT100, 1000.0 for PT1000)
    :param min_val: Minimum expected temperature value
    :param max_val: Maximum expected temperature value
    """

    type: Literal["ai_rtd"] = "ai_rtd"
    units: Literal["DegC", "DegF", "Kelvins", "DegR"]
    rtd_type: Literal[
        "Pt3750", "Pt3851", "Pt3911", "Pt3916", "Pt3920", "Pt3928", "Pt3850"
    ]
    resistance_config: Literal["2Wire", "3Wire", "4Wire"]
    current_excit_source: ExcitationSource
    current_excit_val: float
    r0: float


class AIStrainGageChan(BaseAIChan, MinMaxVal):
    """
    Analog Input Strain Gauge Channel

    Measures mechanical strain using foil strain gages in various bridge configurations.
    Applies voltage excitation and measures bridge imbalance caused by strain.
    Fundamental measurement for stress analysis, structural testing, and load monitoring.

    For detailed information, see the NI-DAQmx documentation:
    <https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaistraingagechan.html>

    Example:
        >>> # Quarter-bridge strain gage configuration
        >>> strain_chan = AIStrainGageChan(
        ...     port=0,
        ...     channel=0,
        ...     terminal_config="Diff",
        ...     units="strain",
        ...     strain_config="quarter-bridge-I",
        ...     voltage_excit_source="Internal",
        ...     voltage_excit_val=2.5,
        ...     gage_factor=2.1,  # Typical for foil gages
        ...     initial_bridge_voltage=0.0,
        ...     nominal_gage_resistance=350.0,
        ...     poisson_ratio=0.3,  # For steel
        ...     lead_wire_resistance=0.0,
        ...     min_val=-0.005,  # -5000 microstrain
        ...     max_val=0.005    # +5000 microstrain
        ... )

    :param terminal_config: Input terminal configuration
    :param units: Output units (strain - dimensionless)
    :param strain_config: Bridge configuration type
    :param voltage_excit_source: Excitation voltage source
    :param voltage_excit_val: Excitation voltage in volts (typically 2.5-10V)
    :param gage_factor: Gage factor provided by manufacturer (typically 2.0-2.1)
    :param initial_bridge_voltage: Initial bridge output voltage for offset compensation
    :param nominal_gage_resistance: Gage resistance in ohms (commonly 120, 350, or 1000)
    :param poisson_ratio: Poisson's ratio of test material
    :param lead_wire_resistance: Lead wire resistance in ohms
    :param custom_scale: Optional custom scaling
    :param min_val: Minimum expected strain
    :param max_val: Maximum expected strain
    """

    type: Literal["ai_strain_gauge"] = "ai_strain_gauge"
    terminal_config: TerminalConfig = "Cfg_Default"
    units: Literal["strain"] = "strain"
    strain_config: Literal[
        "full-bridge-I",
        "full-bridge-II",
        "full-bridge-III",
        "half-bridge-I",
        "half-bridge-II",
        "quarter-bridge-I",
        "quarter-bridge-II",
    ]
    voltage_excit_source: ExcitationSource
    voltage_excit_val: float
    gage_factor: float
    initial_bridge_voltage: float
    nominal_gage_resistance: float
    poisson_ratio: float
    lead_wire_resistance: float
    custom_scale: Scale = NoScale()


class AITempBuiltInChan(BaseAIChan):
    """
    Analog Input Temperature Built-In Channel

    Reads temperature from built-in temperature sensors on the DAQ device itself.
    Useful for monitoring ambient temperature for cold-junction compensation or
    environmental monitoring. No external sensor required.

    For detailed information, see the NI-DAQmx documentation:
    <https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaitempbuiltinchan.html>

    Example:
        >>> # Read built-in temperature sensor
        >>> temp_builtin_chan = AITempBuiltInChan(
        ...     port=0,
        ...     channel=0,
        ...     units="DegC"
        ... )

    :param units: Output temperature units (DegC, DegF, Kelvins, DegR)
    """

    type: Literal["ai_temp_builtin"] = "ai_temp_builtin"
    units: Literal["DegC", "DegF", "Kelvins", "DegR"]


class AIThermocoupleChan(BaseAIChan, MinMaxVal):
    """
    Analog Input Thermocouple Channel

    Measures temperature using thermocouples with built-in cold-junction compensation.
    Supports all standard thermocouple types (J, K, T, E, R, S, B, N).
    Ideal for wide temperature range measurements in harsh environments.

    For detailed information, see the NI-DAQmx documentation:
    <https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaithermocouplechan.html>

    Example:
        >>> # K-type thermocouple with built-in CJC
        >>> thermocouple_chan = AIThermocoupleChan(
        ...     port=0,
        ...     channel=0,
        ...     units="DegC",
        ...     thermocouple_type="K",
        ...     cjc_source="BuiltIn",  # Use built-in cold-junction sensor
        ...     cjc_val=None,  # Not needed for BuiltIn
        ...     cjc_port=None,  # Not needed for BuiltIn
        ...     min_val=-200.0,
        ...     max_val=1200.0
        ... )

    :param units: Output temperature units (DegC, DegF, Kelvins, DegR)
    :param thermocouple_type: Thermocouple type (J, K, N, R, S, T, B, E)
    :param cjc_source: Cold-junction compensation source (BuiltIn, ConstVal, or Chan)
    :param cjc_val: Constant CJC temperature if cjc_source is ConstVal
    :param cjc_port: Port number for CJC channel if cjc_source is Chan
    :param min_val: Minimum expected temperature
    :param max_val: Maximum expected temperature
    """

    type: Literal["ai_thermocouple"] = "ai_thermocouple"
    units: Literal["DegC", "DegF", "Kelvins", "DegR"]
    thermocouple_type: Literal["J", "K", "N", "R", "S", "T", "B", "E"]
    cjc_source: Literal["BuiltIn", "ConstVal", "Chan"]
    cjc_val: float | None
    cjc_port: int | None


class AIThermistorChanIex(BaseAIChan, MinMaxVal):
    """
    Analog Input Thermistor IEX Channel

    Measures temperature using thermistors with current excitation.
    Uses Steinhart-Hart equation (coefficients a, b, c) to convert resistance to temperature.
    Excellent for narrow temperature range measurements with high resolution.

    For detailed information, see the NI-DAQmx documentation:
    <https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaithermistoriexchan.html>

    Example:
        >>> # NTC thermistor with current excitation
        >>> thermistor_iex_chan = AIThermistorChanIex(
        ...     port=0,
        ...     channel=0,
        ...     units="DegC",
        ...     resistance_config="4Wire",
        ...     current_excit_source="Internal",
        ...     current_excit_val=0.00015,  # 150 µA
        ...     a=0.001129241,  # Steinhart-Hart coefficients
        ...     b=0.000234126,
        ...     c=0.000000876,
        ...     min_val=-40.0,
        ...     max_val=125.0
        ... )

    :param units: Output temperature units (DegC, DegF, Kelvins, DegR)
    :param resistance_config: Wire configuration (2Wire, 3Wire, or 4Wire)
    :param current_excit_source: Excitation current source
    :param current_excit_val: Excitation current in amps (typically 10-250 µA)
    :param a: Steinhart-Hart coefficient A
    :param b: Steinhart-Hart coefficient B
    :param c: Steinhart-Hart coefficient C
    :param min_val: Minimum expected temperature
    :param max_val: Maximum expected temperature
    """

    type: Literal["ai_thermistor_iex"] = "ai_thermistor_iex"
    units: Literal["DegC", "DegF", "Kelvins", "DegR"]
    resistance_config: Literal["2Wire", "3Wire", "4Wire"]
    current_excit_source: ExcitationSource
    current_excit_val: float
    a: float
    b: float
    c: float


class AIThermistorChanVex(BaseAIChan, MinMaxVal):
    """
    Analog Input Thermistor VEX Channel

    Measures temperature using thermistors with voltage excitation in a voltage divider
    configuration. Uses Steinhart-Hart equation with series resistor (r1) for scaling.
    Common configuration for simple thermistor temperature measurements.

    For detailed information, see the NI-DAQmx documentation:
    <https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaithermistorvexchan.html>

    Example:
        >>> # NTC thermistor in voltage divider
        >>> thermistor_vex_chan = AIThermistorChanVex(
        ...     port=0,
        ...     channel=0,
        ...     units="DegC",
        ...     resistance_config="3Wire",
        ...     voltage_excit_source="Internal",
        ...     voltage_excit_val=2.5,
        ...     a=0.001129241,  # Steinhart-Hart coefficients
        ...     b=0.000234126,
        ...     c=0.000000876,
        ...     r1=10000.0,  # 10kΩ series resistor
        ...     min_val=-40.0,
        ...     max_val=125.0
        ... )

    :param units: Output temperature units (DegC, DegF, Kelvins, DegR)
    :param resistance_config: Wire configuration (2Wire, 3Wire, or 4Wire)
    :param voltage_excit_source: Excitation voltage source
    :param voltage_excit_val: Excitation voltage in volts
    :param a: Steinhart-Hart coefficient A
    :param b: Steinhart-Hart coefficient B
    :param c: Steinhart-Hart coefficient C
    :param r1: Series resistor value in ohms
    :param min_val: Minimum expected temperature
    :param max_val: Maximum expected temperature
    """

    type: Literal["ai_thermistor_vex"] = "ai_thermistor_vex"
    units: Literal["DegC", "DegF", "Kelvins", "DegR"]
    resistance_config: Literal["2Wire", "3Wire", "4Wire"]
    voltage_excit_source: ExcitationSource
    voltage_excit_val: float
    a: float
    b: float
    c: float
    r1: float


class AITorqueBridgePolynomialChan(BaseAIChan, MinMaxVal):
    """
    Analog Input Torque Bridge Polynomial Channel

    Measures torque using a Wheatstone bridge torque sensor with polynomial scaling.
    Polynomial coefficients convert bridge output to torque units. Common in
    rotating shaft torque measurements and dynamometer applications.

    For detailed information, see the NI-DAQmx documentation:
    <https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaitorquebridgepolynomialchan.html>

    Example:
        >>> # Torque sensor with polynomial calibration
        >>> torque_poly_chan = AITorqueBridgePolynomialChan(
        ...     port=0,
        ...     channel=0,
        ...     units="NewtonMeters",
        ...     bridge_config="FullBridge",
        ...     voltage_excit_source="Internal",
        ...     voltage_excit_val=10.0,
        ...     nominal_bridge_resistance=350.0,
        ...     forward_coeffs=[0.0, 500.0, 0.05],
        ...     reverse_coeffs=[0.0, 0.002],
        ...     electrical_units="mVoltsPerVolt",
        ...     physical_units="NewtonMeters",
        ...     min_val=-100.0,
        ...     max_val=100.0
        ... )

    :param units: Output torque units (NewtonMeters, InchOunces, FootPounds)
    :param bridge_config: Bridge type (FullBridge, HalfBridge, QuarterBridge)
    :param voltage_excit_source: Excitation voltage source
    :param voltage_excit_val: Excitation voltage in volts
    :param nominal_bridge_resistance: Bridge resistance in ohms
    :param forward_coeffs: Forward polynomial coefficients
    :param reverse_coeffs: Reverse polynomial coefficients
    :param electrical_units: Bridge output units (mVoltsPerVolt or VoltsPerVolt)
    :param physical_units: Torque units matching the units parameter
    :param custom_scale: Optional custom scaling
    :param min_val: Minimum expected torque
    :param max_val: Maximum expected torque
    """

    type: Literal["ai_torque_bridge_polynomial"] = "ai_torque_bridge_polynomial"
    units: Literal["NewtonMeters", "InchOunces", "FootPounds"]
    bridge_config: Literal["FullBridge", "HalfBridge", "QuarterBridge"]
    voltage_excit_source: ExcitationSource
    voltage_excit_val: float
    nominal_bridge_resistance: float
    forward_coeffs: list[float]
    reverse_coeffs: list[float]
    electrical_units: Literal["mVoltsPerVolt", "VoltsPerVolt"]
    physical_units: Literal["NewtonMeters", "InchOunces", "FootPounds"]
    custom_scale: Scale = NoScale()


class AITorqueBridgeTableChan(BaseAIChan, MinMaxVal):
    """
    Analog Input Torque Bridge Table Channel

    Measures torque using a Wheatstone bridge sensor with table-based scaling.
    Maps electrical values to torque using lookup table interpolation.
    Ideal when you have discrete calibration points from torque sensor testing.

    For detailed information, see the NI-DAQmx documentation:
    <https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaitorquebridgetablechan.html>

    Example:
        >>> # Torque sensor with calibration table
        >>> torque_table_chan = AITorqueBridgeTableChan(
        ...     port=0,
        ...     channel=0,
        ...     units="FootPounds",
        ...     bridge_config="FullBridge",
        ...     voltage_excit_source="Internal",
        ...     voltage_excit_val=10.0,
        ...     nominal_bridge_resistance=350.0,
        ...     electrical_units="mVoltsPerVolt",
        ...     electrical_vals=[-2.0, -1.0, 0.0, 1.0, 2.0],
        ...     physical_units="FootPounds",
        ...     physical_vals=[-50.0, -25.0, 0.0, 25.0, 50.0],
        ...     min_val=-50.0,
        ...     max_val=50.0
        ... )

    :param units: Output torque units (NewtonMeters, InchOunces, FootPounds)
    :param bridge_config: Bridge type (FullBridge, HalfBridge, QuarterBridge)
    :param voltage_excit_source: Excitation voltage source
    :param voltage_excit_val: Excitation voltage in volts
    :param nominal_bridge_resistance: Bridge resistance in ohms
    :param electrical_units: Bridge output units (mVoltsPerVolt or VoltsPerVolt)
    :param electrical_vals: List of electrical calibration point values
    :param physical_units: Torque units matching the units parameter
    :param physical_vals: List of corresponding torque values
    :param custom_scale: Optional custom scaling
    :param min_val: Minimum expected torque
    :param max_val: Maximum expected torque
    """

    type: Literal["ai_torque_bridge_table"] = "ai_torque_bridge_table"
    units: Literal["NewtonMeters", "InchOunces", "FootPounds"]
    bridge_config: Literal["FullBridge", "HalfBridge", "QuarterBridge"]
    voltage_excit_source: ExcitationSource
    voltage_excit_val: float
    nominal_bridge_resistance: float
    electrical_units: Literal["mVoltsPerVolt", "VoltsPerVolt"]
    electrical_vals: list[float]
    physical_units: Literal["NewtonMeters", "InchOunces", "FootPounds"]
    physical_vals: list[float]
    custom_scale: Scale = NoScale()


class AITorqueBridgeTwoPointLinChan(BaseAIChan, MinMaxVal):
    """
    Analog Input Torque Bridge Two Point Linear Channel

    Measures torque using a Wheatstone bridge sensor with two-point linear scaling.
    Defines linear calibration using two known torque/voltage pairs.
    Simple and effective for torque sensors with linear response characteristics.

    For detailed information, see the NI-DAQmx documentation:
    <https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaitorquebridgetwopointlinchan.html>

    Example:
        >>> # Torque sensor with two-point calibration
        >>> torque_2pt_chan = AITorqueBridgeTwoPointLinChan(
        ...     port=0,
        ...     channel=0,
        ...     units="NewtonMeters",
        ...     bridge_config="FullBridge",
        ...     voltage_excit_source="Internal",
        ...     voltage_excit_val=10.0,
        ...     nominal_bridge_resistance=350.0,
        ...     electrical_units="mVoltsPerVolt",
        ...     physical_units="NewtonMeters",
        ...     first_electrical_val=0.0,
        ...     first_physical_val=0.0,     # 0 Nm at 0 mV/V
        ...     second_electrical_val=2.0,
        ...     second_physical_val=200.0,  # 200 Nm at 2 mV/V
        ...     min_val=-20.0,
        ...     max_val=220.0
        ... )

    :param units: Output torque units (NewtonMeters, InchOunces, FootPounds)
    :param bridge_config: Bridge type (FullBridge, HalfBridge, QuarterBridge)
    :param voltage_excit_source: Excitation voltage source
    :param voltage_excit_val: Excitation voltage in volts
    :param nominal_bridge_resistance: Bridge resistance in ohms
    :param electrical_units: Bridge output units (mVoltsPerVolt or VoltsPerVolt)
    :param physical_units: Torque units matching the units parameter
    :param first_electrical_val: First calibration point electrical value
    :param first_physical_val: First calibration point torque value
    :param second_electrical_val: Second calibration point electrical value
    :param second_physical_val: Second calibration point torque value
    :param custom_scale: Optional custom scaling
    :param min_val: Minimum expected torque
    :param max_val: Maximum expected torque
    """

    type: Literal["ai_torque_bridge_two_point_lin"] = "ai_torque_bridge_two_point_lin"
    units: Literal["NewtonMeters", "InchOunces", "FootPounds"]
    bridge_config: Literal["FullBridge", "HalfBridge", "QuarterBridge"]
    voltage_excit_source: ExcitationSource
    voltage_excit_val: float
    nominal_bridge_resistance: float
    electrical_units: Literal["mVoltsPerVolt", "VoltsPerVolt"]
    physical_units: Literal["NewtonMeters", "InchOunces", "FootPounds"]
    first_electrical_val: float
    first_physical_val: float
    second_electrical_val: float
    second_physical_val: float
    custom_scale: Scale = NoScale()


class AIVelocityIEPEChan(BaseAIChan, MinMaxVal):
    """
    Analog Input Velocity IEPE Channel

    Measures velocity using IEPE (Integrated Electronics Piezo-Electric) velocity
    sensors with built-in electronics. Provides current excitation for seismic
    velocity transducers. Common in vibration monitoring and machinery diagnostics.

    For detailed information, see the NI-DAQmx documentation:
    <https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaivelocityiepechan.html>

    Example:
        >>> # IEPE velocity sensor for vibration monitoring
        >>> velocity_iepe_chan = AIVelocityIEPEChan(
        ...     port=0,
        ...     channel=0,
        ...     units="MetersPerSecond",
        ...     terminal_config="PseudoDiff",
        ...     sensitivity=100.0,  # 100 mV/(mm/s)
        ...     sensitivity_units="MillivoltsPerMillimeterPerSecond",
        ...     current_excit_source="Internal",
        ...     current_excit_val=0.004,  # 4 mA
        ...     min_val=-0.5,
        ...     max_val=0.5
        ... )

    :param units: Output velocity units (MetersPerSecond or InchesPerSecond)
    :param terminal_config: Input terminal configuration
    :param sensitivity: Sensor sensitivity value
    :param sensitivity_units: Sensitivity units (MillivoltsPerMillimeterPerSecond or MilliVoltsPerInchPerSecond)
    :param current_excit_source: Excitation current source (Internal, External, or None)
    :param current_excit_val: Excitation current in amps (typically 2-4 mA)
    :param custom_scale: Optional custom scaling
    :param min_val: Minimum expected velocity
    :param max_val: Maximum expected velocity
    """

    type: Literal["ai_velocity_iepe"] = "ai_velocity_iepe"
    units: Literal["MetersPerSecond", "InchesPerSecond"]
    terminal_config: TerminalConfig = "Cfg_Default"
    sensitivity: float
    sensitivity_units: Literal[
        "MillivoltsPerMillimeterPerSecond", "MilliVoltsPerInchPerSecond"
    ]
    current_excit_source: ExcitationSource
    current_excit_val: float
    custom_scale: Scale = NoScale()


class AIVoltageChan(BaseAIChan, MinMaxVal):
    """
    Analog Input Voltage Channel

    Measures DC or AC voltage directly. The most basic and commonly used analog
    input channel type. Suitable for general-purpose voltage measurements from
    sensors, test equipment, or any voltage source.

    For detailed information, see the NI-DAQmx documentation:
    <https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaivoltagechan.html>

    Example:
        >>> # Basic voltage measurement
        >>> voltage_chan = AIVoltageChan(
        ...     port=0,
        ...     channel=0,
        ...     terminal_config="Diff",  # Differential for noise rejection
        ...     units="Volts",
        ...     min_val=-10.0,
        ...     max_val=10.0
        ... )

    :param terminal_config: Input terminal configuration (Diff, RSE, NRSE, PseudoDiff)
    :param units: Output units (Volts)
    :param custom_scale: Optional custom scaling for engineering units
    :param min_val: Minimum expected voltage
    :param max_val: Maximum expected voltage
    """

    type: Literal["ai_voltage"] = "ai_voltage"
    terminal_config: TerminalConfig = "Cfg_Default"
    units: Literal["Volts"] = "Volts"
    custom_scale: Scale = NoScale()


class AIVoltageRMSChan(BaseAIChan, MinMaxVal):
    """
    Analog Input Voltage RMS Channel

    Measures root-mean-square (RMS) value of AC voltage signals. Automatically
    calculates RMS over the acquisition period. Useful for AC power measurements,
    audio levels, or any application requiring true RMS voltage.

    For detailed information, see the NI-DAQmx documentation:
    <https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaivoltagermschan.html>

    Example:
        >>> # AC voltage RMS measurement
        >>> voltage_rms_chan = AIVoltageRMSChan(
        ...     port=0,
        ...     channel=0,
        ...     terminal_config="Diff",
        ...     units="V",
        ...     min_val=0.0,
        ...     max_val=250.0  # 0-250V RMS
        ... )

    :param terminal_config: Input terminal configuration
    :param units: Output units (V or mV)
    :param custom_scale: Optional custom scaling
    :param min_val: Minimum expected RMS voltage
    :param max_val: Maximum expected RMS voltage
    """

    type: Literal["ai_voltage_rms"] = "ai_voltage_rms"
    terminal_config: TerminalConfig = "Cfg_Default"
    units: Literal["V", "mV"]
    custom_scale: Scale = NoScale()


class AIVoltageChanWithExcit(BaseAIChan, MinMaxVal):
    """
    Analog Input Voltage Channel with Excitation

    Measures voltage from sensors that require external voltage excitation.
    Provides excitation voltage and optionally uses it for ratiometric scaling.
    Common for potentiometric sensors, LVDTs, and bridge-based transducers.

    For detailed information, see the NI-DAQmx documentation:
    <https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaivoltagechanwithexcit.html>

    Example:
        >>> # Potentiometric position sensor with excitation
        >>> voltage_excit_chan = AIVoltageChanWithExcit(
        ...     port=0,
        ...     channel=0,
        ...     terminal_config="Diff",
        ...     units="V",
        ...     bridge_config="none",  # Not a bridge, just voltage output
        ...     voltage_excit_source="Internal",
        ...     voltage_excit_val=10.0,  # 10V excitation
        ...     use_excit_for_scaling=True,  # Ratiometric measurement
        ...     min_val=0.0,
        ...     max_val=10.0
        ... )

    :param terminal_config: Input terminal configuration
    :param units: Output units (V or mV)
    :param bridge_config: Bridge configuration (full, half, quarter, or none)
    :param voltage_excit_source: Excitation voltage source (Internal, External, or None)
    :param voltage_excit_val: Excitation voltage in volts
    :param use_excit_for_scaling: Use excitation voltage for ratiometric scaling
    :param custom_scale: Optional custom scaling
    :param min_val: Minimum expected voltage
    :param max_val: Maximum expected voltage
    """

    type: Literal["ai_voltage_with_excit"] = "ai_voltage_with_excit"
    terminal_config: TerminalConfig = "Cfg_Default"
    units: Literal["V", "mV"]
    bridge_config: Literal["full", "half", "quarter", "none"]
    voltage_excit_source: ExcitationSource
    voltage_excit_val: float
    use_excit_for_scaling: bool
    custom_scale: Scale = NoScale()


AIChan = (
    AIVoltageChan
    | AIThermocoupleChan
    | AIRTDChan
    | AIPressureBridgeTwoPointLinChan
    | AIAccelChan
    | AIBridgeChan
    | AICurrentChan
    | AIForceBridgeTableChan
    | AIForceBridgeTwoPointLinChan
    | AIForceIEPEChan
    | AIMicrophoneChan
    | AIPressureBridgeTableChan
    | AIResistanceChan
    | AIStrainGageChan
    | AITempBuiltInChan
    | AITorqueBridgeTableChan
    | AITorqueBridgeTwoPointLinChan
    | AIVelocityIEPEChan
)


class BaseAOChan(BaseChan):
    device: str = ""
    port: int
    cmd_channel: int
    state_channel: int


class AOVoltageChan(BaseAOChan, MinMaxVal):
    """
    Analog Output Voltage Channel

    Generates DC or AC voltage output signals. The most common analog output
    channel type for controlling voltage-input devices, generating test signals,
    or providing analog control signals to external equipment.

    For detailed information, see the NI-DAQmx documentation:
    <https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaovoltagechan.html>

    Example:
        >>> # Basic voltage output channel
        >>> ao_voltage_chan = AOVoltageChan(
        ...     port=0,
        ...     cmd_channel=100,   # Synnax channel for command values
        ...     state_channel=101, # Synnax channel for state feedback
        ...     units="Volts",
        ...     min_val=-10.0,
        ...     max_val=10.0
        ... )

    :param cmd_channel: Synnax channel key for command/setpoint values
    :param state_channel: Synnax channel key for actual output state feedback
    :param units: Output units (Volts)
    :param custom_scale: Optional custom scaling
    :param min_val: Minimum output voltage
    :param max_val: Maximum output voltage
    """

    type: Literal["ao_voltage"] = "ao_voltage"
    units: Literal["Volts"] = "Volts"
    custom_scale: Scale = NoScale()


class AOCurrentChan(BaseAOChan, MinMaxVal):
    """
    Analog Output Current Channel

    Generates current output signals, typically 4-20mA for industrial control
    applications. Ideal for long-distance signal transmission where voltage
    drop is a concern, or for interfacing with current-loop devices.

    For detailed information, see the NI-DAQmx documentation:
    <https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaocurrentchan.html>

    Example:
        >>> # 4-20mA current loop output
        >>> ao_current_chan = AOCurrentChan(
        ...     port=0,
        ...     cmd_channel=102,   # Synnax channel for command values
        ...     state_channel=103, # Synnax channel for state feedback
        ...     units="Amps",
        ...     min_val=0.004,  # 4 mA
        ...     max_val=0.020   # 20 mA
        ... )

    :param cmd_channel: Synnax channel key for command/setpoint values
    :param state_channel: Synnax channel key for actual output state feedback
    :param units: Output units (Amps)
    :param custom_scale: Optional custom scaling
    :param min_val: Minimum output current
    :param max_val: Maximum output current
    """

    type: Literal["ao_current"] = "ao_current"
    units: Literal["Amps"] = "Amps"
    custom_scale: Scale = NoScale()


class AOFuncGenChan(BaseAOChan):
    """
    Analog Output Function Generator Channel

    Generates standard waveforms (sine, square, triangle, sawtooth) using
    onboard function generator hardware. Provides clean, hardware-timed
    waveforms without needing to stream data from the host computer.

    For detailed information, see the NI-DAQmx documentation:
    <https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaofuncgenchan.html>

    Example:
        >>> # 1 kHz sine wave generator
        >>> ao_funcgen_chan = AOFuncGenChan(
        ...     port=0,
        ...     cmd_channel=104,   # Synnax channel for command values
        ...     state_channel=105, # Synnax channel for state feedback
        ...     wave_type="Sine",
        ...     frequency=1000.0,  # 1 kHz
        ...     amplitude=5.0,     # 5V peak
        ...     offset=0.0         # No DC offset
        ... )

    :param cmd_channel: Synnax channel key for command/setpoint values
    :param state_channel: Synnax channel key for actual output state feedback
    :param wave_type: Waveform type (Sine, Triangle, Square, or Sawtooth)
    :param frequency: Output frequency in Hz
    :param amplitude: Peak amplitude in volts
    :param offset: DC offset in volts (default 0.0)
    """

    type: Literal["ao_func_gen"] = "ao_func_gen"
    wave_type: Literal["Sine", "Triangle", "Square", "Sawtooth"]
    frequency: float
    amplitude: float
    offset: float = 0.0


AOChan = AOVoltageChan | AOCurrentChan | AOFuncGenChan


class DIChan(BaseModel):
    """
    Digital Input Channel

    Reads digital state (high/low, 1/0) from a single digital line on a port.
    Commonly used for reading switch states, TTL signals, or other discrete inputs.

    For detailed information, see the NI-DAQmx documentation:
    <https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreatedichan.html>

    Example:
        >>> # Read from port 0, line 3
        >>> di_chan = DIChan(
        ...     channel=100,  # Synnax channel key
        ...     port=0,
        ...     line=3
        ... )

    :param channel: Synnax channel key to write digital input data to
    :param port: Digital port number on the device
    :param line: Line number within the port (0-7 for most devices)
    """

    channel: int
    type: Literal["digital_input"] = "digital_input"
    port: int
    line: int


class BaseCIChan(BaseChan):
    device: str = ""
    port: int


class CIFrequencyChan(BaseCIChan, MinMaxVal):
    """
    Counter Input Frequency Measurement Channel

    Measures the frequency of a digital signal using counter hardware. Supports
    multiple measurement methods optimized for different frequency ranges.

    For detailed information, see the NI-DAQmx documentation:
    <https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreatecifreqchan.html>

    Example:
        >>> # Measure 0-1000 Hz signal on counter 0
        >>> freq_chan = CIFrequencyChan(
        ...     port=0,
        ...     units="Hz",
        ...     edge="Rising",
        ...     meas_method="LowFreq1Ctr",  # Best for < 1 kHz
        ...     meas_time=0.001,  # 1ms measurement window
        ...     divisor=4,
        ...     terminal="/Dev1/PFI0",  # Input terminal
        ...     min_val=0.0,
        ...     max_val=1000.0
        ... )

    :param units: Output units (Hz for frequency, Seconds for period, Ticks for raw)
    :param edge: Which signal edge to count (Rising or Falling)
    :param meas_method: Measurement algorithm - LowFreq1Ctr for < 100 kHz, HighFreq2Ctr for > 100 kHz
    :param meas_time: Measurement averaging time in seconds
    :param divisor: Frequency divisor for HighFreq2Ctr method
    :param terminal: Physical terminal to measure (e.g., "/Dev1/PFI0")
    :param custom_scale: Optional custom scaling for output values
    :param min_val: Minimum expected frequency value
    :param max_val: Maximum expected frequency value
    """

    type: Literal["ci_frequency"] = "ci_frequency"
    units: Literal["Hz", "Seconds", "Ticks"] = "Hz"
    edge: Literal["Rising", "Falling"] = "Rising"
    meas_method: Literal["LowFreq1Ctr", "HighFreq2Ctr", "DynAvg"] = "LowFreq1Ctr"
    meas_time: float = 0.001
    divisor: int = 4
    terminal: str = ""
    custom_scale: Scale = NoScale()


class CIEdgeCountChan(BaseCIChan):
    """
    Counter Input Edge Count Channel

    Counts digital edges on an input signal. Useful for totalizing events,
    measuring encoder positions, or counting pulses. Can count up, down, or
    be externally controlled.

    For detailed information, see the NI-DAQmx documentation:
    <https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreatecicountedgeschan.html>

    Example:
        >>> # Count rising edges on PFI0
        >>> edge_count_chan = CIEdgeCountChan(
        ...     port=0,
        ...     active_edge="Rising",
        ...     count_direction="CountUp",
        ...     initial_count=0,
        ...     terminal="/Dev1/PFI0"
        ... )

    :param active_edge: Which edge to count (Rising or Falling)
    :param count_direction: Direction of counting (CountUp, CountDown, or ExtControlled)
    :param initial_count: Starting count value (default 0)
    :param terminal: Input terminal to count edges on (e.g., "/Dev1/PFI0")
    """

    type: Literal["ci_edge_count"] = "ci_edge_count"
    active_edge: Literal["Rising", "Falling"] = "Rising"
    count_direction: Literal["CountUp", "CountDown", "ExtControlled"] = "CountUp"
    initial_count: int = 0
    terminal: str = ""


class CIPeriodChan(BaseCIChan, MinMaxVal):
    """
    Counter Input Period Measurement Channel

    Measures the time duration between consecutive edges of a digital signal.
    This is the inverse of frequency measurement, useful for low-frequency signals
    or when period (not frequency) is the desired measurement.

    For detailed information, see the NI-DAQmx documentation:
    <https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateciperiodchan.html>

    Example:
        >>> # Measure period of 1-100 Hz signal
        >>> period_chan = CIPeriodChan(
        ...     port=0,
        ...     units="Seconds",
        ...     starting_edge="Rising",
        ...     meas_method="LowFreq1Ctr",
        ...     meas_time=0.001,
        ...     divisor=4,
        ...     terminal="/Dev1/PFI0",
        ...     min_val=0.01,   # 100 Hz = 0.01s period
        ...     max_val=1.0     # 1 Hz = 1s period
        ... )

    :param units: Output units (Seconds or Ticks)
    :param starting_edge: Edge that starts the period measurement
    :param meas_method: Measurement algorithm based on expected frequency range
    :param meas_time: Measurement averaging time in seconds
    :param divisor: Frequency divisor for HighFreq2Ctr method
    :param terminal: Physical input terminal (e.g., "/Dev1/PFI0")
    :param custom_scale: Optional custom scaling
    :param min_val: Minimum expected period (1/max_frequency)
    :param max_val: Maximum expected period (1/min_frequency)
    """

    type: Literal["ci_period"] = "ci_period"
    units: Literal["Seconds", "Ticks"] = "Seconds"
    starting_edge: Literal["Rising", "Falling"] = "Rising"
    meas_method: Literal["LowFreq1Ctr", "HighFreq2Ctr", "DynAvg"] = "LowFreq1Ctr"
    meas_time: float = 0.001
    divisor: int = 4
    terminal: str = ""
    custom_scale: Scale = NoScale()


class CIPulseWidthChan(BaseCIChan, MinMaxVal):
    """
    Counter Input Pulse Width Measurement Channel

    Measures the time duration of a pulse (high or low state) on a digital signal.
    Starting edge determines whether to measure high-time (Rising start) or
    low-time (Falling start).

    For detailed information, see the NI-DAQmx documentation:
    <https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateciplsewidthchan.html>

    Example:
        >>> # Measure pulse width (high-time) of PWM signal
        >>> pulse_width_chan = CIPulseWidthChan(
        ...     port=0,
        ...     units="Seconds",
        ...     starting_edge="Rising",  # Measure from rising to falling
        ...     terminal="/Dev1/PFI0",
        ...     min_val=0.000001,  # 1 µs minimum
        ...     max_val=0.001      # 1 ms maximum
        ... )

    :param units: Output units (Seconds or Ticks)
    :param starting_edge: Rising = measure high-time, Falling = measure low-time
    :param terminal: Physical input terminal (e.g., "/Dev1/PFI0")
    :param custom_scale: Optional custom scaling
    :param min_val: Minimum expected pulse width
    :param max_val: Maximum expected pulse width
    """

    type: Literal["ci_pulse_width"] = "ci_pulse_width"
    units: Literal["Seconds", "Ticks"] = "Seconds"
    starting_edge: Literal["Rising", "Falling"] = "Rising"
    terminal: str = ""
    custom_scale: Scale = NoScale()


class CISemiPeriodChan(BaseCIChan, MinMaxVal):
    """
    Counter Input Semi Period Measurement Channel

    Measures the time between alternating edges (rising-to-falling and falling-to-rising).
    This provides both high-time and low-time measurements in a single channel, useful
    for PWM duty cycle analysis or asymmetric waveform characterization.

    For detailed information, see the NI-DAQmx documentation:
    <https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreatecisemiperiodchan.html>

    Example:
        >>> # Measure both high and low semi-periods
        >>> semi_period_chan = CISemiPeriodChan(
        ...     port=0,
        ...     units="Seconds",
        ...     terminal="/Dev1/PFI0",
        ...     min_val=0.00001,  # 10 µs
        ...     max_val=0.01      # 10 ms
        ... )

    :param units: Output units (Seconds or Ticks)
    :param terminal: Physical input terminal (e.g., "/Dev1/PFI0")
    :param custom_scale: Optional custom scaling
    :param min_val: Minimum expected semi-period duration
    :param max_val: Maximum expected semi-period duration
    """

    type: Literal["ci_semi_period"] = "ci_semi_period"
    units: Literal["Seconds", "Ticks"] = "Seconds"
    terminal: str = ""
    custom_scale: Scale = NoScale()


class CITwoEdgeSepChan(BaseCIChan, MinMaxVal):
    """
    Counter Input Two Edge Separation Measurement Channel

    Measures the time between two edges that can be on different terminals.
    Useful for time-of-flight measurements, propagation delay measurements,
    or any application requiring precise timing between two events.

    For detailed information, see the NI-DAQmx documentation:
    <https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreatecitwoe
    dgeseparationchan.html>

    Example:
        >>> # Measure time between trigger and response signals
        >>> two_edge_sep_chan = CITwoEdgeSepChan(
        ...     port=0,
        ...     units="Seconds",
        ...     first_edge="Rising",
        ...     second_edge="Rising",
        ...     first_terminal="/Dev1/PFI0",   # Trigger signal
        ...     second_terminal="/Dev1/PFI1",  # Response signal
        ...     min_val=0.0,
        ...     max_val=0.01  # 10ms max separation
        ... )

    :param units: Output units (Seconds or Ticks)
    :param first_edge: Edge type on first terminal to start measurement
    :param second_edge: Edge type on second terminal to stop measurement
    :param first_terminal: First input terminal (e.g., "/Dev1/PFI0")
    :param second_terminal: Second input terminal (e.g., "/Dev1/PFI1")
    :param custom_scale: Optional custom scaling
    :param min_val: Minimum expected time separation
    :param max_val: Maximum expected time separation
    """

    type: Literal["ci_two_edge_sep"] = "ci_two_edge_sep"
    units: Literal["Seconds", "Ticks"] = "Seconds"
    first_edge: Literal["Rising", "Falling"] = "Rising"
    second_edge: Literal["Rising", "Falling"] = "Falling"
    first_terminal: str = ""
    second_terminal: str = ""
    custom_scale: Scale = NoScale()


class CILinearVelocityChan(BaseCIChan, MinMaxVal):
    """
    Counter Input Linear Velocity Measurement Channel

    Measures linear velocity from a quadrature encoder. Commonly used with
    linear encoders on actuators, conveyor belts, or CNC machines. Supports
    multiple decoding methods for different resolution requirements.

    For detailed information, see the NI-DAQmx documentation:
    <https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateci
    linvelocitychan.html>

    Example:
        >>> # Linear encoder with 1mm resolution, X4 decoding
        >>> lin_velocity_chan = CILinearVelocityChan(
        ...     port=0,
        ...     units="MetersPerSecond",
        ...     decoding_type="X4",  # 4x resolution
        ...     dist_per_pulse=0.001,  # 1mm per encoder pulse
        ...     terminalA="/Dev1/PFI0",  # A phase
        ...     terminalB="/Dev1/PFI1",  # B phase
        ...     min_val=0.0,
        ...     max_val=1.0  # 0-1 m/s range
        ... )

    :param units: Velocity units (MetersPerSecond or InchesPerSecond)
    :param decoding_type: X1=1x, X2=2x, X4=4x resolution, TwoPulse=two pulse encoder
    :param dist_per_pulse: Linear distance traveled per encoder pulse (in selected units)
    :param terminalA: Encoder A phase input terminal
    :param terminalB: Encoder B phase input terminal
    :param custom_scale: Optional custom scaling
    :param min_val: Minimum expected velocity
    :param max_val: Maximum expected velocity
    """

    type: Literal["ci_velocity_linear"] = "ci_velocity_linear"
    units: Literal["MetersPerSecond", "InchesPerSecond"] = "MetersPerSecond"
    decoding_type: Literal["X1", "X2", "X4", "TwoPulse"] = "X4"
    dist_per_pulse: float
    terminalA: str = ""
    terminalB: str = ""
    custom_scale: Scale = NoScale()


class CIAngularVelocityChan(BaseCIChan, MinMaxVal):
    """
    Counter Input Angular Velocity Measurement Channel

    Measures rotational velocity from a quadrature encoder. Commonly used with
    motors, turntables, or rotating machinery. Provides real-time RPM or angular
    velocity measurements.

    For detailed information, see the NI-DAQmx documentation:
    <https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreatecia
    ngvelocitychan.html>

    Example:
        >>> # Motor encoder with 1024 PPR (pulses per revolution)
        >>> ang_velocity_chan = CIAngularVelocityChan(
        ...     port=0,
        ...     units="RPM",
        ...     decoding_type="X4",  # 4x resolution = 4096 counts/rev
        ...     pulses_per_rev=1024,
        ...     terminalA="/Dev1/PFI0",  # A phase
        ...     terminalB="/Dev1/PFI1",  # B phase
        ...     min_val=0.0,
        ...     max_val=5000.0  # 0-5000 RPM
        ... )

    :param units: Velocity units (RPM, Radians/s, or Degrees)
    :param decoding_type: X1=1x, X2=2x, X4=4x resolution, TwoPulse=two pulse encoder
    :param pulses_per_rev: Encoder pulses per revolution (PPR)
    :param terminalA: Encoder A phase input terminal
    :param terminalB: Encoder B phase input terminal
    :param custom_scale: Optional custom scaling
    :param min_val: Minimum expected angular velocity
    :param max_val: Maximum expected angular velocity
    """

    type: Literal["ci_velocity_angular"] = "ci_velocity_angular"
    units: Literal["RPM", "Radians/s", "Degrees"] = "RPM"
    decoding_type: Literal["X1", "X2", "X4", "TwoPulse"] = "X4"
    pulses_per_rev: int
    terminalA: str = ""
    terminalB: str = ""
    custom_scale: Scale = NoScale()


class CILinearPositionChan(BaseCIChan):
    """
    Counter Input Linear Position Measurement Channel

    Tracks absolute or incremental linear position from a quadrature encoder.
    Supports Z-index for position reset/homing. Commonly used in CNC machines,
    linear stages, and precision positioning systems.

    For detailed information, see the NI-DAQmx documentation:
    <https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreatecil
    inencoderchan.html>

    Example:
        >>> # Linear stage with 10µm resolution and Z-index homing
        >>> lin_position_chan = CILinearPositionChan(
        ...     port=0,
        ...     units="Meters",
        ...     decoding_type="X4",
        ...     dist_per_pulse=0.00001,  # 10 µm per pulse
        ...     initial_pos=0.0,
        ...     z_index_enable=True,  # Enable homing with Z pulse
        ...     z_index_val=0.0,  # Reset to 0 at Z pulse
        ...     z_index_phase="AHighBLow",
        ...     terminalA="/Dev1/PFI0",
        ...     terminalB="/Dev1/PFI1",
        ...     terminalZ="/Dev1/PFI2"  # Z-index for homing
        ... )

    :param units: Position units (Meters or Inches)
    :param decoding_type: X1=1x, X2=2x, X4=4x resolution
    :param dist_per_pulse: Linear distance per encoder pulse
    :param initial_pos: Starting position value
    :param z_index_enable: Enable Z-index pulse for position reset
    :param z_index_val: Position value to set when Z-index occurs
    :param z_index_phase: AB phase state when Z-index is valid
    :param terminalA: Encoder A phase input
    :param terminalB: Encoder B phase input
    :param terminalZ: Z-index input (optional, for homing)
    :param custom_scale: Optional custom scaling
    """

    type: Literal["ci_position_linear"] = "ci_position_linear"
    units: Literal["Meters", "Inches"] = "Meters"
    decoding_type: Literal["X1", "X2", "X4", "TwoPulse"] = "X4"
    dist_per_pulse: float
    initial_pos: float = 0.0
    z_index_enable: bool = False
    z_index_val: float = 0.0
    z_index_phase: Literal["AHighBHigh", "AHighBLow", "ALowBHigh", "ALowBLow"] = (
        "AHighBHigh"
    )
    terminalA: str = ""
    terminalB: str = ""
    terminalZ: str = ""
    custom_scale: Scale = NoScale()


class CIAngularPositionChan(BaseCIChan):
    """
    Counter Input Angular Position Measurement Channel

    Tracks angular position from a rotary encoder with optional Z-index homing.
    Useful for motor control, turntables, and rotational positioning systems.
    Position can be tracked in degrees or radians.

    For detailed information, see the NI-DAQmx documentation:
    <https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreatecia
    ngencoderchan.html>

    Example:
        >>> # Rotary encoder with 2048 PPR and homing
        >>> ang_position_chan = CIAngularPositionChan(
        ...     port=0,
        ...     units="Degrees",
        ...     decoding_type="X4",  # 8192 counts/rev
        ...     pulses_per_rev=2048,
        ...     initial_angle=0.0,
        ...     z_index_enable=True,
        ...     z_index_val=0.0,  # Reset to 0° at Z pulse
        ...     z_index_phase="AHighBHigh",
        ...     terminalA="/Dev1/PFI0",
        ...     terminalB="/Dev1/PFI1",
        ...     terminalZ="/Dev1/PFI2"
        ... )

    :param units: Angular units (Degrees or Radians)
    :param decoding_type: X1=1x, X2=2x, X4=4x resolution
    :param pulses_per_rev: Encoder pulses per revolution (PPR)
    :param initial_angle: Starting angle value
    :param z_index_enable: Enable Z-index pulse for angle reset
    :param z_index_val: Angle value to set when Z-index occurs
    :param z_index_phase: AB phase state when Z-index is valid
    :param terminalA: Encoder A phase input
    :param terminalB: Encoder B phase input
    :param terminalZ: Z-index input (optional, for homing)
    :param custom_scale: Optional custom scaling
    """

    type: Literal["ci_position_angular"] = "ci_position_angular"
    units: Literal["Degrees", "Radians"] = "Degrees"
    decoding_type: Literal["X1", "X2", "X4", "TwoPulse"] = "X4"
    pulses_per_rev: int
    initial_angle: float = 0.0
    z_index_enable: bool = False
    z_index_val: float = 0.0
    z_index_phase: Literal["AHighBHigh", "AHighBLow", "ALowBHigh", "ALowBLow"] = (
        "AHighBHigh"
    )
    terminalA: str = ""
    terminalB: str = ""
    terminalZ: str = ""
    custom_scale: Scale = NoScale()


class CIDutyCycleChan(BaseCIChan, MinMaxVal):
    """
    Counter Input Duty Cycle Measurement Channel

    Measures the duty cycle (percentage of time signal is high) of a PWM or
    pulse signal. Useful for analyzing PWM control signals or validating
    signal generation.

    For detailed information, see the NI-DAQmx documentation:
    <https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreatecidutycyclechan.html>

    Example:
        >>> # Measure PWM duty cycle (0-100%)
        >>> duty_cycle_chan = CIDutyCycleChan(
        ...     port=0,
        ...     activeEdge="Rising",
        ...     terminal="/Dev1/PFI0",
        ...     min_val=0.0,   # 0% duty cycle
        ...     max_val=1.0    # 100% duty cycle (expressed as 0.0-1.0)
        ... )

    :param activeEdge: Edge that starts the measurement cycle
    :param terminal: Physical input terminal (e.g., "/Dev1/PFI0")
    :param custom_scale: Optional custom scaling
    :param min_val: Minimum expected duty cycle (0.0 = 0%)
    :param max_val: Maximum expected duty cycle (1.0 = 100%)
    """

    type: Literal["ci_duty_cycle"] = "ci_duty_cycle"
    activeEdge: Literal["Rising", "Falling"] = "Rising"
    terminal: str = ""
    custom_scale: Scale = NoScale()


CIChan = (
    CIFrequencyChan
    | CIEdgeCountChan
    | CIPeriodChan
    | CIPulseWidthChan
    | CISemiPeriodChan
    | CITwoEdgeSepChan
    | CILinearVelocityChan
    | CIAngularVelocityChan
    | CILinearPositionChan
    | CIAngularPositionChan
    | CIDutyCycleChan
)


class DOChan(BaseChan):
    """
    Digital Output Channel

    Controls digital output lines for switching, relay control, TTL signals,
    or any on/off control application. Each channel controls a single digital
    line on a port.

    For detailed information, see the NI-DAQmx documentation:
    <https://www.ni.com/documentation/en/ni-daqmx/latest/daqmxcfunc/daqmxcreatedochan.html>

    Example:
        >>> # Digital output for relay control
        >>> do_chan = DOChan(
        ...     cmd_channel=106,   # Synnax channel for command values
        ...     state_channel=107, # Synnax channel for state feedback
        ...     port=0,
        ...     line=5
        ... )

    :param cmd_channel: Synnax channel key to read digital output commands from
    :param state_channel: Synnax channel key to write digital output state to
    :param port: Digital port number on the device
    :param line: Line number within the port (0-7 for most devices)
    """

    type: Literal["digital_output"] = "digital_output"
    cmd_channel: int
    state_channel: int
    port: int
    line: int


class AnalogReadTaskConfig(BaseReadTaskConfig):
    """Configuration for NI Analog Read Task.

    Inherits common read task fields (sample_rate, stream_rate, data_saving,
    auto_start) from BaseReadTaskConfig and adds NI-specific channel configuration.
    """

    device: str = ""
    "The key of the Synnax NI device to read from (optional, can be set per channel)."
    channels: list[AIChan]

    @field_validator("channels")
    def validate_channel_ports(cls, v, values):
        ports = {c.port for c in v}
        if len(ports) < len(v):
            used_ports = [c.port for c in v]
            duplicate_ports = [port for port in ports if used_ports.count(port) > 1]
            raise ValueError(f"Port {duplicate_ports[0]} has already been used")
        return v


class AnalogWriteConfig(BaseWriteTaskConfig):
    """Configuration for NI Analog Write Task.

    Inherits common write task fields (device, auto_start) from
    BaseWriteTaskConfig and adds NI-specific data saving, state rate,
    and channel configuration with NI hardware state rate limits (50 kHz max).
    """

    data_saving: bool = True
    "Whether to persist state feedback data to disk (True) or only stream it (False)."
    state_rate: conint(ge=0, le=50000)
    "The rate at which to write task channel states to the Synnax cluster (Hz)."
    channels: list[AOChan]


class CounterReadConfig(BaseReadTaskConfig):
    """Configuration for NI Counter Read Task.

    Inherits common read task fields (sample_rate, stream_rate, data_saving,
    auto_start) from BaseReadTaskConfig and adds NI-specific channel configuration.
    """

    device: str = ""
    "The key of the Synnax NI device to read from (optional, can be set per channel)."
    channels: list[CIChan]

    @field_validator("channels")
    def validate_channel_ports(cls, v):
        ports = {c.port for c in v}
        if len(ports) < len(v):
            used_ports = [c.port for c in v]
            duplicate_ports = [port for port in ports if used_ports.count(port) > 1]
            raise ValueError(f"Port {duplicate_ports[0]} has already been used")
        return v


class DigitalReadConfig(BaseReadTaskConfig):
    """Configuration for NI Digital Read Task.

    Inherits common read task fields (sample_rate, stream_rate, data_saving,
    auto_start) from BaseReadTaskConfig and adds NI-specific channel configuration.
    """

    device: str = Field(min_length=1)
    "The key of the Synnax NI device to read from."
    channels: list[DIChan]


class DigitalWriteConfig(BaseWriteTaskConfig):
    """Configuration for NI Digital Write Task.

    Inherits common write task fields (device, auto_start) from
    BaseWriteTaskConfig and adds NI-specific data saving, state rate,
    and channel configuration with NI hardware state rate limits (50 kHz max).
    """

    data_saving: bool = True
    "Whether to persist state feedback data to disk (True) or only stream it (False)."
    state_rate: conint(ge=0, le=50000)
    "The rate at which to write task channel states to the Synnax cluster (Hz)."
    channels: list[DOChan]


class TaskStateDetails(BaseModel):
    running: bool
    message: str


class AnalogReadStateDetails(TaskStateDetails):
    errors: list[dict[str, str]] | None


class AnalogReadTask(StarterStopperMixin, JSONConfigMixin, MetaTask):
    """A task for reading analog data from NI devices and writing them to a Synnax
    cluster. This task is a programmatic representation of the analog read task
    configurable within the Synnax console. For detailed information on
    configuring/operating a analog read task,
    see https://docs.synnaxlabs.com/reference/driver/ni/analog-read-task

    :param device: The key of the Synnax OPC UA device to read from.
    :param name: A human-readable name for the task.
    :param sample_rate: The rate at which to sample data from the OPC UA device.
    :param stream_rate: The rate at which acquired data will be streamed to the Synnax
        cluster. For example, a sample rate of 100Hz and a stream rate of 25Hz will
        result in groups of 4 samples being streamed to the cluster every 40ms.
    :param channels: A list of physical channel configurations to acquire data from.
        These can be any channel subtype of AIChan
    :param data_saving: Whether to save data permanently within Synnax, or just stream
        it for real-time consumption.
    """

    TYPE = "ni_analog_read"
    config: AnalogReadTaskConfig
    _internal: Task

    def __init__(
        self,
        internal: Task | None = None,
        *,
        device: str = "",
        name: str = "",
        sample_rate: CrudeRate = 0,
        stream_rate: CrudeRate = 0,
        data_saving: bool = False,
        auto_start: bool = False,
        channels: list[AIChan] = None,
    ) -> None:
        if internal is not None:
            self._internal = internal
            self.config = AnalogReadTaskConfig.model_validate_json(internal.config)
            return
        self._internal = Task(name=name, type=self.TYPE)
        self.config = AnalogReadTaskConfig(
            device=device,
            sample_rate=sample_rate,
            stream_rate=stream_rate,
            data_saving=data_saving,
            auto_start=auto_start,
            channels=channels,
        )
        # Set the device provided to the task for any channels that don't have a device
        # field assigned.
        for i, channel in enumerate(self.config.channels):
            if len(channel.device) == 0:
                if len(device) == 0:
                    raise ValidationError(
                        f"""
                        No device provided for {channel} {i + 1} in task and no device
                        provided directly to the task. Please provide a device for the
                        channel or set the device for the task.
                    """
                    )
                channel.device = device


class AnalogWriteTask(StarterStopperMixin, JSONConfigMixin, MetaTask):
    """A task for writing analog output data to NI devices. This task is a programmatic
    representation of the analog write task configurable within the Synnax console.
    For detailed information on configuring/operating an analog write task, see
    https://docs.synnaxlabs.com/reference/driver/ni/analog-write-task

    :param device: The key of the Synnax NI device to write to.
    :param name: A human-readable name for the task.
    :param state_rate: The rate at which to write task channel states to the Synnax
        cluster.
    :param channels: A list of analog output channel configurations (AOChan subtypes:
        AOVoltageChan, AOCurrentChan, AOFuncGenChan).
    :param data_saving: Whether to save data permanently within Synnax, or just stream
        it for real-time consumption.
    :param auto_start: Whether to start the task automatically when it is created.
    """

    TYPE = "ni_analog_write"
    config: AnalogWriteConfig
    _internal: Task

    def __init__(
        self,
        internal: Task | None = None,
        *,
        device: str = "",
        name: str = "",
        state_rate: CrudeRate = 0,
        data_saving: bool = False,
        auto_start: bool = False,
        channels: list[AOChan] = None,
    ):
        if internal is not None:
            self._internal = internal
            self.config = AnalogWriteConfig.model_validate_json(internal.config)
            return
        self._internal = Task(name=name, type=self.TYPE)
        self.config = AnalogWriteConfig(
            device=device,
            state_rate=state_rate,
            data_saving=data_saving,
            auto_start=auto_start,
            channels=channels,
        )


class CounterReadTask(StarterStopperMixin, JSONConfigMixin, MetaTask):
    """A task for reading counter data from NI devices and writing them to a Synnax
    cluster. This task is a programmatic representation of the counter read task
    configurable within the Synnax console. For detailed information on
    configuring/operating a counter read task,
    see https://docs.synnaxlabs.com/reference/driver/ni/counter-read-task

    :param device: The key of the Synnax NI device to read from.
    :param name: A human-readable name for the task.
    :param sample_rate: The rate at which to sample data from the NI device.
    :param stream_rate: The rate at which acquired data will be streamed to the Synnax
        cluster. For example, a sample rate of 100Hz and a stream rate of 25Hz will
        result in groups of 4 samples being streamed to the cluster every 40ms.
    :param channels: A list of counter input channel configurations (CIChan subtypes).
    :param data_saving: Whether to save data permanently within Synnax, or just stream
        it for real-time consumption.
    :param auto_start: Whether to start the task automatically when it is created.
    """

    TYPE = "ni_counter_read"
    config: CounterReadConfig
    _internal: Task

    def __init__(
        self,
        internal: Task | None = None,
        *,
        device: str = "",
        name: str = "",
        sample_rate: CrudeRate = 0,
        stream_rate: CrudeRate = 0,
        data_saving: bool = False,
        auto_start: bool = False,
        channels: list[CIChan] = None,
    ) -> None:
        if internal is not None:
            self._internal = internal
            self.config = CounterReadConfig.model_validate_json(internal.config)
            return
        self._internal = Task(name=name, type=self.TYPE)
        self.config = CounterReadConfig(
            device=device,
            sample_rate=sample_rate,
            stream_rate=stream_rate,
            data_saving=data_saving,
            auto_start=auto_start,
            channels=channels,
        )
        # Set the device provided to the task for any channels that don't have a device
        # field assigned.
        for i, channel in enumerate(self.config.channels):
            if len(channel.device) == 0:
                if len(device) == 0:
                    raise ValidationError(
                        f"""
                        No device provided for {channel} {i + 1} in task and no device
                        provided directly to the task. Please provide a device for the
                        channel or set the device for the task.
                    """
                    )
                channel.device = device


class DigitalReadTask(StarterStopperMixin, JSONConfigMixin, MetaTask):
    """A task for reading digital data from NI devices and writing them to a Synnax
    cluster. This task is a programmatic representation of the digital read task
    configurable within the Synnax console. For detailed information on
    configuring/operating a digital read task,
    see https://docs.synnaxlabs.com/reference/driver/ni/digital-read-task

    :param device: The key of the Synnax OPC UA device to read from.
    :param name: A human-readable name for the task.
    :param sample_rate: The rate at which to sample data from the OPC UA device.
    :param stream_rate: The rate at which acquired data will be streamed to the Synnax
        cluster. For example, a sample rate of 100Hz and a stream rate of 25Hz will
        result in groups of 4 samples being streamed to the cluster every 40ms.
    :param channels: A list of physical channel configurations to acquire data from.
        These can be any channel subtype of DIChan.
    :param data_saving: Whether to save data permanently within Synnax, or just stream
        it for real-time consumption.
    """

    TYPE = "ni_digital_read"
    config: DigitalReadConfig
    _internal: Task

    def __init__(
        self,
        internal: Task | None = None,
        *,
        device: str = "",
        name: str = "",
        sample_rate: CrudeRate = 0,
        stream_rate: CrudeRate = 0,
        data_saving: bool = False,
        auto_start: bool = False,
        channels: list[DIChan] = None,
    ) -> None:
        if internal is not None:
            self._internal = internal
            self.config = DigitalReadConfig.model_validate_json(internal.config)
            return
        self._internal = Task(name=name, type=self.TYPE)
        self.config = DigitalReadConfig(
            device=device,
            sample_rate=sample_rate,
            stream_rate=stream_rate,
            data_saving=data_saving,
            auto_start=auto_start,
            channels=channels,
        )


class DigitalWriteTask(StarterStopperMixin, JSONConfigMixin, MetaTask):
    """A task for reading digital data from NI devices and writing them to a Synnax
    cluster. This task is a programmatic representation of the digital write task
    configurable within the Synnax console. For detailed information on
    configuring/operating a digital write task, see https://docs.synnaxlabs.com/reference/driver/ni/digital-write-task

    :param device: The key of the Synnax OPC UA device to read from.
    :param name: A human-readable name for the task.
    :param state_rate: The rate at which to write task channel states to the Synnax
        cluster.
    :param channels: A list of physical channel configurations to acquire data from.
        These can be any channel subtype of AIChan
    :param data_saving: Whether to save data permanently within Synnax, or just stream
        it for real-time consumption.
    """

    TYPE = "ni_digital_write"
    config: DigitalWriteConfig
    _internal: Task

    def __init__(
        self,
        internal: Task | None = None,
        *,
        device: str = "",
        name: str = "",
        state_rate: CrudeRate = 0,
        data_saving: bool = False,
        auto_start: bool = False,
        channels: list[DOChan] = None,
    ):
        if internal is not None:
            self._internal = internal
            self.config = DigitalWriteConfig.model_validate_json(internal.config)
            return
        self._internal = Task(name=name, type=self.TYPE)
        self.config = DigitalWriteConfig(
            device=device,
            state_rate=state_rate,
            data_saving=data_saving,
            auto_start=auto_start,
            channels=channels,
        )


# Device identifiers - must match Console expectations
MAKE = "NI"


class Device(device.Device):
    """
    National Instruments device configuration.

    This class extends the base Device class to provide NI-specific configuration.
    The `properties` field is automatically populated with the identifier for the device.

    Example:
        >>> from synnax.hardware import ni
        >>> device = ni.Device(
        ...     name="My NI Module",
        ...     model="NI 9205",
        ...     location="cDAQ1/dev_mod1",
        ...     rack=rack.key,
        ...     identifier="dev_mod1"
        ... )
        >>> client.hardware.devices.create(device)

    :param identifier: Channel name prefix for all channels on this device
    """

    def __init__(self, identifier: str, **kwargs):
        """
        Initialize an NI device.

        Args:
            identifier: Channel name prefix for all channels on this device
            **kwargs: Additional device properties (name, model, location, rack, etc.)
        """
        # Auto-generate key if not provided
        if "key" not in kwargs:
            kwargs["key"] = str(uuid4())

        # Set make to NI
        kwargs["make"] = MAKE

        # Set properties with identifier
        kwargs["properties"] = json.dumps({"identifier": identifier})

        super().__init__(**kwargs)
