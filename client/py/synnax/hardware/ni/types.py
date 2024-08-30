from pydantic import BaseModel, conint, confloat, constr, validator, Field
from typing import List, Literal, Union, Optional, Dict
from uuid import uuid4
from synnax.hardware.task import TaskPayload, Task, MetaTask
from synnax.telem import CrudeRate, TimeSpan
from contextlib import contextmanager
import json

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

Units = Union[
    UnitsVolts,
    UnitsAmps,
    UnitsDegF,
    UnitsDegC,
    UnitsDegR,
    UnitsKelvins,
    UnitsStrain,
    UnitsOhms,
    UnitsHz,
    UnitsSeconds,
    UnitsMeters,
    UnitsInches,
    UnitsDegAngle,
    UnitsRadiansAngle,
    UnitsGravity,
    UnitsMetersPerSecondSquared,
    UnitsNewtons,
    UnitsPounds,
    UnitsKgForce,
    UnitsLbsPerSquareInch,
    UnitsBar,
    UnitsPascals,
    UnitsVoltsPerVolt,
    UnitsmVoltsPerVolt,
    UnitsNewtonMeters,
    UnitsInchLbs,
    UnitsInOz,
    UnitsFtLbs,
]


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
    pre_scaled_vals: List[float]
    scaled_vals: List[float]
    pre_scaled_units: Units


class PolynomialScale(BaseModel):
    """Custom polynomial scale for analog input channels.

    For detailed information, see the NI-DAQmx documentation:
    <https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreatepoly3scale.html>
    """

    type: Literal["polynomial"] = "polynomial"
    forward_coeffs: List[float]
    reverse_coeffs: List[float]
    pre_scaled_units: Units
    scaled_units: Units


class NoScale(BaseModel):
    """Applies no scaling to the analog input channel. This is a default value that
    should rarely be used.
    """

    type: Literal["none"] = "none"


Scale = Union[LinScale, MapScale, TableScale, NoScale]
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
    port: int
    channel: int


class MinMaxVal(BaseModel):
    min_val: float = 0
    max_val: float = 1


class AIAccelChan(BaseAIChan, MinMaxVal):
    """
    Analog Input Accelerometer Channel

    For detailed information, see the NI-DAQmx documentation:
    <https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaiaccelchan.html>
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

    For detailed information, see the NI-DAQmx documentation:
    https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaiaccel4wiredcvoltagechan.html
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

    For detailed information, see the NI-DAQmx documentation:
    <https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaiaccelchargechan.html>
    """

    type: Literal["ai_accel_charge"] = "ai_accel_charge"
    units: Literal["g", "MetersPerSecondSquared", "InchesPerSecondSquared"]
    custom_scale: Scale = NoScale()


class AIBridgeChan(BaseAIChan, MinMaxVal):
    """
    Analog Input Bridge Channel

    For detailed information, see the NI-DAQmx documentation:
    <https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaibridgechan.html>
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

    For detailed information, see the NI-DAQmx documentation:
    <https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaichargechan.html>
    """

    type: Literal["ai_charge"] = "ai_charge"
    terminal_config: TerminalConfig = "Cfg_Default"
    units: Literal["C", "uC"]
    custom_scale: Scale = NoScale()


class AICurrentChan(BaseAIChan, MinMaxVal):
    """
    Analog Input Current Channel

    For detailed information, see the NI-DAQmx documentation:
    <https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaicurrentchan.html>
    """

    type: Literal["ai_current"] = "ai_current"
    terminal_config: TerminalConfig = "Cfg_Default"
    units: Literal["Amps"] = "Amps"
    shunt_resistor_loc: Literal["Default", "Internal", "External"]
    ext_shunt_resistor_val: confloat(gt=0)
    custom_scale: Scale = NoScale()


class AICurrentRMSChan(BaseAIChan, MinMaxVal):
    """
    Analog Current RMS Channel

    For detailed information, see the NI-DAQmx documentation:
    <https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaicurrentrmschan.html>
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

    For detailed information, see the NI-DAQmx documentation:
    <https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaiforcebridgepolynomialchan.html>
    """

    type: Literal["ai_force_bridge_polynomial"] = "ai_force_bridge_polynomial"
    terminal_config: TerminalConfig = "Cfg_Default"
    units: Literal["Newtons", "Pounds", "KilogramForce"]
    bridge_config: Literal["FullBridge", "HalfBridge", "QuarterBridge"]
    voltage_excit_source: ExcitationSource
    voltage_excit_val: float
    nominal_bridge_resistance: float
    forward_coeffs: List[float]
    reverse_coeffs: List[float]
    electrical_units: Literal["mVoltsPerVolt", "VoltsPerVolt"]
    physical_units: Literal["Newtons", "Pounds", "KilogramForce"]
    custom_scale: Scale = NoScale()


class AIForceBridgeTableChan(BaseAIChan, MinMaxVal):
    """
    Analog Input Force Bridge Table Channel

    For detailed information, see the NI-DAQmx documentation:
    <https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaiforcebridgetablechan.html>
    """

    type: Literal["ai_force_bridge_table"] = "ai_force_bridge_table"
    units: Literal["Newtons", "Pounds", "KilogramForce"]
    bridge_config: Literal["FullBridge", "HalfBridge", "QuarterBridge"]
    voltage_excit_source: ExcitationSource
    voltage_excit_val: float
    nominal_bridge_resistance: float
    electrical_units: Literal["mVoltsPerVolt", "VoltsPerVolt"]
    electrical_vals: List[float]
    physical_units: Literal["Newtons", "Pounds", "KilogramForce"]
    physical_vals: List[float]
    custom_scale: Scale = NoScale()


class AIForceBridgeTwoPointLinChan(BaseAIChan, MinMaxVal):
    """
    Analog Input Force Bridge Two Point Linear Channel

    For detailed information, see the NI-DAQmx documentation:
    <https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaiforcebridgetwopointlinchan.html>
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

    For detailed information, see the NI-DAQmx documentation:
    <https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaiforceiepechan.html>
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

    For detailed information, see the NI-DAQmx documentation:
    <https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaifreqvoltagechan.html>
    """

    type: Literal["ai_freq_voltage"] = "ai_freq_voltage"
    units: Literal["Hz"] = "Hz"
    threshold_level: float
    hysteresis: float
    custom_scale: Scale = NoScale()


class AIMicrophoneChan(BaseAIChan):
    """
    Analog Input Microphone Channel

    For detailed information, see the NI-DAQmx documentation:
    <https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaimicrophonechan.html>
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

    For detailed information, see the NI-DAQmx documentation:
    <https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaipressurebridgepolynomialchan.html>
    """

    type: Literal["ai_pressure_bridge_polynomial"] = "ai_pressure_bridge_polynomial"
    units: Literal["PoundsPerSquareInch", "Pascals", "Bar"]
    bridge_config: Literal["FullBridge", "HalfBridge", "QuarterBridge"]
    voltage_excit_source: ExcitationSource
    voltage_excit_val: float
    nominal_bridge_resistance: float
    forward_coeffs: List[float]
    reverse_coeffs: List[float]
    electrical_units: Literal["mVoltsPerVolt", "VoltsPerVolt"]
    physical_units: Literal["PoundsPerSquareInch", "Pascals", "Bar"]
    custom_scale: Scale = NoScale()


class AIPressureBridgeTableChan(BaseAIChan, MinMaxVal):
    """
    Analog Input Pressure Bridge Table Channel

    For detailed information, see the NI-DAQmx documentation:
    <https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaipressurebridgetablechan.html>
    """

    type: Literal["ai_pressure_bridge_table"] = "ai_pressure_bridge_table"
    units: Literal["PoundsPerSquareInch", "Pascals", "Bar"]
    bridge_config: Literal["FullBridge", "HalfBridge", "QuarterBridge"]
    voltage_excit_source: ExcitationSource
    voltage_excit_val: float
    nominal_bridge_resistance: float
    electrical_units: Literal["mVoltsPerVolt", "VoltsPerVolt"]
    electrical_vals: List[float]
    physical_units: Literal["PoundsPerSquareInch", "Pascals", "Bar"]
    physical_vals: List[float]
    custom_scale: Scale = NoScale()


class AIPressureBridgeTwoPointLinChan(BaseAIChan, MinMaxVal):
    """
    Analog Input Pressure Bridge Two Point Linear Channel

    For detailed information, see the NI-DAQmx documentation:
    <https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaipressurebridgetwopointlinchan.html>
    """

    type: Literal[
        "ai_pressure_bridge_two_point_lin"
    ] = "ai_pressure_bridge_two_point_lin"
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

    For detailed information, see the NI-DAQmx documentation:
    <https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateairesistancechan.html>
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

    For detailed information, see the NI-DAQmx documentation:
    <https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateairosettestraingagechan.html>
    """

    type: Literal["ai_rosette_strain_gage"] = "ai_rosette_strain_gage"
    terminal_config: TerminalConfig = "Cfg_Default"
    rosette_type: Literal["RectangularRosette", "DeltaRosette", "TeeRosette"]
    gage_orientation: float
    rosette_meas_types: List[
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

    For detailed information, see the NI-DAQmx documentation:
    <https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaistraingagechan.html>
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

    For detailed information, see the NI-DAQmx documentation:
    <https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaitempbuiltinchan.html>
    """

    type: Literal["ai_temp_builtin"] = "ai_temp_builtin"
    units: Literal["DegC", "DegF", "Kelvins", "DegR"]


class AIThermocoupleChan(BaseAIChan, MinMaxVal):
    """
    Analog Input Thermocouple Channel

    For detailed information, see the NI-DAQmx documentation:
    <https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaithermocouplechan.html>
    """

    type: Literal["ai_thermocouple"] = "ai_thermocouple"
    units: Literal["DegC", "DegF", "Kelvins", "DegR"]
    thermocouple_type: Literal["J", "K", "N", "R", "S", "T", "B", "E"]
    cjc_source: Literal["BuiltIn", "ConstVal", "Chan"]
    cjc_val: Optional[float]
    cjc_port: Optional[int]


class AIThermistorChanIex(BaseAIChan, MinMaxVal):
    """
    Analog Input Thermistor IEX Channel

    For detailed information, see the NI-DAQmx documentation:
    <https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaithermistoriexchan.html>
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

    For detailed information, see the NI-DAQmx documentation:
    <https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaithermistorvexchan.html>
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

    For detailed information, see the NI-DAQmx documentation:
    <https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaitorquebridgepolynomialchan.html>
    """

    type: Literal["ai_torque_bridge_polynomial"] = "ai_torque_bridge_polynomial"
    units: Literal["NewtonMeters", "InchOunces", "FootPounds"]
    bridge_config: Literal["FullBridge", "HalfBridge", "QuarterBridge"]
    voltage_excit_source: ExcitationSource
    voltage_excit_val: float
    nominal_bridge_resistance: float
    forward_coeffs: List[float]
    reverse_coeffs: List[float]
    electrical_units: Literal["mVoltsPerVolt", "VoltsPerVolt"]
    physical_units: Literal["NewtonMeters", "InchOunces", "FootPounds"]
    custom_scale: Scale = NoScale()


class AITorqueBridgeTableChan(BaseAIChan, MinMaxVal):
    """
    Analog Input Torque Bridge Table Channel

    For detailed information, see the NI-DAQmx documentation:
    <https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaitorquebridgetablechan.html>
    """

    type: Literal["ai_torque_bridge_table"] = "ai_torque_bridge_table"
    units: Literal["NewtonMeters", "InchOunces", "FootPounds"]
    bridge_config: Literal["FullBridge", "HalfBridge", "QuarterBridge"]
    voltage_excit_source: ExcitationSource
    voltage_excit_val: float
    nominal_bridge_resistance: float
    electrical_units: Literal["mVoltsPerVolt", "VoltsPerVolt"]
    electrical_vals: List[float]
    physical_units: Literal["NewtonMeters", "InchOunces", "FootPounds"]
    physical_vals: List[float]
    custom_scale: Scale = NoScale()


class AITorqueBridgeTwoPointLinChan(BaseAIChan, MinMaxVal):
    """
    Analog Input Torque Bridge Two Point Linear Channel

    For detailed information, see the NI-DAQmx documentation:
    <https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaitorquebridgetwopointlinchan
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

    For detailed information, see the NI-DAQmx documentation:
    <https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaivelocityiepechan.html>
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
    """Analog Input Voltage Channel

    For detailed information, see the NI-DAQmx documentation:
    <https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaivoltagechan.html>
    """

    type: Literal["ai_voltage"] = "ai_voltage"
    terminal_config: TerminalConfig = "Cfg_Default"
    units: Literal["Volts"] = "Volts"
    custom_scale: Scale = NoScale()


class AIVoltageRMSChan(BaseAIChan, MinMaxVal):
    """
    Analog Input Voltage RMS Channel

    For detailed information, see the NI-DAQmx documentation:
    <https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaivoltagermschan.html>
    """

    type: Literal["ai_voltage_rms"] = "ai_voltage_rms"
    terminal_config: TerminalConfig = "Cfg_Default"
    units: Literal["V", "mV"]
    custom_scale: Scale = NoScale()


class AIVoltageChanWithExcit(BaseAIChan, MinMaxVal):
    """
    Analog Input Voltage Channel with Excitation

    For detailed information, see the NI-DAQmx documentation:
    <https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaivoltagechanwithexcit.html>
    """

    type: Literal["ai_voltage_with_excit"] = "ai_voltage_with_excit"
    terminal_config: TerminalConfig = "Cfg_Default"
    units: Literal["V", "mV"]
    bridge_config: Literal["full", "half", "quarter", "none"]
    voltage_excit_source: ExcitationSource
    voltage_excit_val: float
    use_excit_for_scaling: bool
    custom_scale: Scale = NoScale()


AIChan = Union[
    AIVoltageChan,
    AIThermocoupleChan,
    AIRTDChan,
    AIPressureBridgeTwoPointLinChan,
    AIAccelChan,
    AIBridgeChan,
    AICurrentChan,
    AIForceBridgeTableChan,
    AIForceBridgeTwoPointLinChan,
    AIForceIEPEChan,
    AIMicrophoneChan,
    AIPressureBridgeTableChan,
    AIResistanceChan,
    AIStrainGageChan,
    AITempBuiltInChan,
    AITorqueBridgeTableChan,
    AITorqueBridgeTwoPointLinChan,
    AIVelocityIEPEChan,
]


class DOChan(BaseChan):
    """
    Digital Output Channel

    For detailed information, see the NI-DAQmx documentation:
    <https://www.ni.com/documentation/en/ni-daqmx/latest/daqmxcfunc/daqmxcreatedochan.html>
    """

    type: Literal["digital_output"] = "digital_output"
    cmd_channel: int
    state_channel: int
    port: int
    line: int


class DIChan(BaseModel):
    """
    Digital Input Channel

    For detailed information, see the NI-DAQmx documentation:
    <https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreatedichan.html>
    """

    channel: int
    type: Literal["digital_input"] = "digital_input"
    port: int
    line: int


class AnalogReadTaskConfig(BaseModel):
    device: str
    sample_rate: conint(ge=0, le=50000)
    stream_rate: conint(ge=0, le=50000)
    channels: List[AIChan]
    data_saving: bool

    @validator("stream_rate")
    def validate_stream_rate(cls, v, values):
        if "sample_rate" in values and v > values["sample_rate"]:
            raise ValueError(
                "Stream rate must be less than or equal to the sample rate"
            )
        return v

    @validator("channels")
    def validate_channel_ports(cls, v, values):
        ports = {c.port for c in v}
        if len(ports) < len(v):
            used_ports = [c.port for c in v]
            duplicate_ports = [port for port in ports if used_ports.count(port) > 1]
            raise ValueError(f"Port {duplicate_ports[0]} has already been used")
        return v


class DigitalWriteConfig(BaseModel):
    device: str
    channels: List[DOChan]
    state_rate: conint(ge=0, le=50000)
    data_saving: bool


class DigitalReadConfig(BaseModel):
    device: str
    sample_rate: conint(ge=0, le=50000)
    stream_rate: conint(ge=0, le=50000)
    data_saving: bool
    channels: List[DIChan]


class TaskStateDetails(BaseModel):
    running: bool
    message: str


class AnalogReadStateDetails(TaskStateDetails):
    errors: Optional[List[Dict[str, str]]]


class DigitalWriteTask(MetaTask):
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
        channels: List[DOChan] = None,
    ):
        if internal is not None:
            self._internal = internal
            self.config = DigitalWriteConfig.parse_obj(json.loads(internal.config))
            return
        self._internal = Task(name=name, type=self.TYPE)
        self.config = DigitalWriteConfig(
            device=device,
            state_rate=state_rate,
            data_saving=data_saving,
            channels=channels,
        )

    @property
    def name(self) -> str:
        return self._internal.name

    @property
    def key(self) -> int:
        return self._internal.key

    def to_payload(self) -> TaskPayload:
        pld = self._internal.to_payload()
        pld.config = json.dumps(self.config.dict())
        return pld

    def set_internal(self, task: Task):
        self._internal = task

    @contextmanager
    def start(self, timeout: float | TimeSpan = 0):
        self._internal.execute_command("start", timeout)
        try:
            yield
        finally:
            self.stop()

    def stop(self, timeout: float | TimeSpan = 0):
        self._internal.execute_command("stop", timeout)


class DigitalReadTask(MetaTask):
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
        channels: List[DIChan] = None,
    ) -> None:
        if internal is not None:
            self._internal = internal
            self.config = DigitalReadConfig.parse_obj(json.loads(internal.config))
            return
        self._internal = Task(name=name, type=self.TYPE)
        self.config = DigitalReadConfig(
            device=device,
            sample_rate=sample_rate,
            stream_rate=stream_rate,
            data_saving=data_saving,
            channels=channels,
        )

    @property
    def name(self) -> str:
        return self._internal.name

    @property
    def key(self) -> int:
        return self._internal.key

    def to_payload(self) -> TaskPayload:
        pld = self._internal.to_payload()
        pld.config = json.dumps(self.config.dict())
        return pld

    def set_internal(self, task: Task):
        self._internal = task

    @contextmanager
    def start(self, timeout: float | TimeSpan = 0):
        self._internal.execute_command_sync("start", timeout)
        try:
            yield
        finally:
            self.stop()

    def stop(self, timeout: float | TimeSpan = 0):
        self._internal.execute_command_sync("stop", timeout)


class AnalogReadTask(MetaTask):
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
        channels: List[AIChan] = None,
    ) -> None:
        if internal is not None:
            self._internal = internal
            self.config = AnalogReadTaskConfig.parse_obj(json.loads(internal.config))
            return
        self._internal = Task(name=name, type=self.TYPE)
        self.config = AnalogReadTaskConfig(
            device=device,
            sample_rate=sample_rate,
            stream_rate=stream_rate,
            data_saving=data_saving,
            channels=channels,
        )

    @property
    def name(self) -> str:
        return self._internal.name

    @property
    def key(self) -> int:
        return self._internal.key

    def to_payload(self) -> TaskPayload:
        pld = self._internal.to_payload()
        pld.config = json.dumps(self.config.dict())
        return pld

    def set_internal(self, task: Task):
        self._internal = task

    @contextmanager
    def start(self, timeout: float | TimeSpan = 0):
        self._internal.execute_command_sync("start", timeout)
        try:
            yield
        finally:
            self.stop()

    def stop(self, timeout: float | TimeSpan = 0):
        self._internal.execute_command_sync("stop", timeout)
