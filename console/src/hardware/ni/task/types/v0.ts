// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, device, type task } from "@synnaxlabs/client";
import { z } from "zod";

export const VERSION = "0.0.0";
export const PREFIX = "ni";

const baseChanZ = z.object({ key: z.string(), port: z.number(), enabled: z.boolean() });
interface BaseChan extends z.infer<typeof baseChanZ> {}
const ZERO_BASE_CHAN: BaseChan = { key: "", port: 0, enabled: true };

const baseAIChanZ = baseChanZ.extend({ name: z.string(), channel: channel.keyZ });
interface BaseAIChan extends z.infer<typeof baseAIChanZ> {}
const ZERO_BASE_AI_CHAN: BaseAIChan = { ...ZERO_BASE_CHAN, name: "", channel: 0 };

const minMaxValZ = z.object({ minVal: z.number(), maxVal: z.number() });
interface MinMaxVal extends z.infer<typeof minMaxValZ> {}
const ZERO_MIN_MAX_VAL: MinMaxVal = { minVal: 0, maxVal: 1 };

const DEFAULT_TERMINAL_CONFIG = "Cfg_Default";
const RSE_TERMINAL_CONFIG = "RSE";
const NRSE_TERMINAL_CONFIG = "NRSE";
const DIFF_TERMINAL_CONFIG = "Diff";
const PSEUDO_DIFF_TERMINAL_CONFIG = "PseudoDiff";
const terminalConfigZ = z.enum([
  DEFAULT_TERMINAL_CONFIG,
  RSE_TERMINAL_CONFIG,
  NRSE_TERMINAL_CONFIG,
  DIFF_TERMINAL_CONFIG,
  PSEUDO_DIFF_TERMINAL_CONFIG,
]);

const terminalZ = z.object({ terminalConfig: terminalConfigZ });
interface Terminal extends z.infer<typeof terminalZ> {}
const ZERO_TERMINAL: Terminal = { terminalConfig: DEFAULT_TERMINAL_CONFIG };

const INTERNAL_EXCIT_SOURCE = "Internal";
const EXTERNAL_EXCIT_SOURCE = "External";
const NONE_EXCIT_SOURCE = "None";
const excitSourceZ = z.enum([
  INTERNAL_EXCIT_SOURCE,
  EXTERNAL_EXCIT_SOURCE,
  NONE_EXCIT_SOURCE,
]);

const currentExcitZ = z.object({
  currentExcitSource: excitSourceZ,
  currentExcitVal: z.number(),
});
interface CurrentExcit extends z.infer<typeof currentExcitZ> {}
const ZERO_CURRENT_EXCIT: CurrentExcit = {
  currentExcitSource: INTERNAL_EXCIT_SOURCE,
  currentExcitVal: 0,
};

const VOLTS = "Volts";
const AMPS = "Amps";
const DEG_F = "DegF";
const DEG_C = "DegC";
const DEG_R = "DegR";
const KELVINS = "Kelvins";
const STRAIN = "Strain";
const OHMS = "Ohms";
const HZ = "Hz";
const SECONDS = "Seconds";
const METERS = "Meters";
const INCHES = "Inches";
const DEGREES = "Degrees";
const RADIANS = "Radians";
const GRAVITY = "g";
const METERS_PER_SECOND_SQUARED = "MetersPerSecondSquared";
const NEWTONS = "Newtons";
const POUNDS = "Pounds";
const KG_FORCE = "KilogramForce";
const LBS_PER_SQUARE_INCH = "PoundsPerSquareInch";
const BAR = "Bar";
const PASCALS = "Pascals";
const VOLTS_PER_VOLT = "VoltsPerVolt";
const MVOLTS_PER_VOLT = "mVoltsPerVolt";
const NEWTON_METERS = "NewtonMeters";
const INCH_LBS = "InchPounds";
const IN_OZ = "InchOunces";
const FT_LBS = "FootPounds";

const unitsZ = z.enum([
  VOLTS,
  AMPS,
  DEG_F,
  DEG_C,
  DEG_R,
  KELVINS,
  STRAIN,
  OHMS,
  HZ,
  SECONDS,
  METERS,
  INCHES,
  DEGREES,
  RADIANS,
  GRAVITY,
  METERS_PER_SECOND_SQUARED,
  NEWTONS,
  POUNDS,
  KG_FORCE,
  LBS_PER_SQUARE_INCH,
  BAR,
  PASCALS,
  VOLTS_PER_VOLT,
  MVOLTS_PER_VOLT,
  NEWTON_METERS,
  INCH_LBS,
  IN_OZ,
  FT_LBS,
]);
export type Units = z.infer<typeof unitsZ>;

const LINEAR_SCALE_TYPE = "linear";
const linearScaleZ = z.object({
  type: z.literal(LINEAR_SCALE_TYPE),
  slope: z.number().refine((val) => val !== 0, { message: "Slope must be nonzero" }),
  yIntercept: z.number(),
  preScaledUnits: unitsZ,
  scaledUnits: unitsZ,
});
interface LinearScale extends z.infer<typeof linearScaleZ> {}
const ZERO_LINEAR_SCALE: LinearScale = {
  type: LINEAR_SCALE_TYPE,
  slope: 1,
  yIntercept: 0,
  preScaledUnits: VOLTS,
  scaledUnits: VOLTS,
};

const MAP_SCALE_TYPE = "map";
const mapScaleZ = z.object({
  type: z.literal(MAP_SCALE_TYPE),
  preScaledMin: z.number(),
  preScaledMax: z.number(),
  scaledMin: z.number(),
  scaledMax: z.number(),
  preScaledUnits: unitsZ,
});
interface MapScale extends z.infer<typeof mapScaleZ> {}
const ZERO_MAP_SCALE: MapScale = {
  type: MAP_SCALE_TYPE,
  preScaledMin: 0,
  preScaledMax: 0,
  scaledMin: 0,
  scaledMax: 0,
  preScaledUnits: VOLTS,
};

const TABLE_SCALE_TYPE = "table";
const tableScaleZ = z.object({
  type: z.literal(TABLE_SCALE_TYPE),
  preScaledVals: z.array(z.number()),
  scaledVals: z.array(z.number()),
  preScaledUnits: unitsZ,
});
interface TableScale extends z.infer<typeof tableScaleZ> {}
const ZERO_TABLE_SCALE: TableScale = {
  type: TABLE_SCALE_TYPE,
  preScaledVals: [],
  scaledVals: [],
  preScaledUnits: VOLTS,
};

const NO_SCALE_TYPE = "none";
const noScaleZ = z.object({ type: z.literal(NO_SCALE_TYPE) });
interface NoScale extends z.infer<typeof noScaleZ> {}
const NO_SCALE: NoScale = { type: NO_SCALE_TYPE };

const scaleZ = z.union([linearScaleZ, mapScaleZ, tableScaleZ, noScaleZ]);
export type Scale = z.infer<typeof scaleZ>;
export type ScaleType = Scale["type"];

const customScaleZ = z.object({ customScale: scaleZ });
interface CustomScale extends z.infer<typeof customScaleZ> {}
const ZERO_CUSTOM_SCALE: CustomScale = { customScale: NO_SCALE };

export const ZERO_SCALES: Record<ScaleType, Scale> = {
  [LINEAR_SCALE_TYPE]: ZERO_LINEAR_SCALE,
  [MAP_SCALE_TYPE]: ZERO_MAP_SCALE,
  [TABLE_SCALE_TYPE]: ZERO_TABLE_SCALE,
  [NO_SCALE_TYPE]: NO_SCALE,
};

export const SCALE_SCHEMAS: Record<ScaleType, z.ZodType<Scale>> = {
  [LINEAR_SCALE_TYPE]: linearScaleZ,
  [MAP_SCALE_TYPE]: mapScaleZ,
  [TABLE_SCALE_TYPE]: tableScaleZ,
  [NO_SCALE_TYPE]: noScaleZ,
};

const MVOLTS_PER_G = "mVoltsPerG";
const VOLTS_PER_G = "VoltsPerG";

const accelSensitivityUnitsZ = z.enum([MVOLTS_PER_G, VOLTS_PER_G]);
export type AccelSensitivityUnits = z.infer<typeof accelSensitivityUnitsZ>;

const INCHES_PER_SECOND_SQUARED = "InchesPerSecondSquared";

const accelUnitsZ = z.enum([
  GRAVITY,
  METERS_PER_SECOND_SQUARED,
  INCHES_PER_SECOND_SQUARED,
]);
export type AccelUnits = z.infer<typeof accelUnitsZ>;

const KILOGRAM_FORCE = "KilogramForce";
const forceUnitsZ = z.enum([NEWTONS, POUNDS, KILOGRAM_FORCE]);
export type ForceUnits = z.infer<typeof forceUnitsZ>;

const electricalUnitsZ = z.enum([MVOLTS_PER_VOLT, VOLTS_PER_VOLT]);
export type ElectricalUnits = z.infer<typeof electricalUnitsZ>;

const sensitivityZ = z.object({ sensitivity: z.number() });
interface Sensitivity extends z.infer<typeof sensitivityZ> {}
const ZERO_SENSITIVITY: Sensitivity = { sensitivity: 0 };

// https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaiaccelchan.html
export const AI_ACCEL_CHAN_TYPE = "ai_accel";
export const aiAccelChanZ = baseAIChanZ
  .merge(terminalZ)
  .merge(minMaxValZ)
  .merge(sensitivityZ)
  .merge(currentExcitZ)
  .merge(customScaleZ)
  .extend({
    type: z.literal(AI_ACCEL_CHAN_TYPE),
    units: accelUnitsZ,
    sensitivityUnits: accelSensitivityUnitsZ,
  });
interface AIAccelChan extends z.infer<typeof aiAccelChanZ> {}
export const ZERO_AI_ACCEL_CHAN: AIAccelChan = {
  ...ZERO_BASE_AI_CHAN,
  ...ZERO_TERMINAL,
  ...ZERO_MIN_MAX_VAL,
  ...ZERO_SENSITIVITY,
  ...ZERO_CURRENT_EXCIT,
  ...ZERO_CUSTOM_SCALE,
  type: AI_ACCEL_CHAN_TYPE,
  units: GRAVITY,
  sensitivityUnits: MVOLTS_PER_G,
};

const voltageExcitZ = z.object({
  voltageExcitSource: excitSourceZ,
  voltageExcitVal: z.number(),
});
interface VoltageExcit extends z.infer<typeof voltageExcitZ> {}
const ZERO_VOLTAGE_EXCIT: VoltageExcit = {
  voltageExcitSource: INTERNAL_EXCIT_SOURCE,
  voltageExcitVal: 0,
};

const FULL_BRIDGE_CONFIG = "FullBridge";
const HALF_BRIDGE_CONFIG = "HalfBridge";
const QUARTER_BRIDGE_CONFIG = "QuarterBridge";
const bridgeConfigZ = z.enum([
  FULL_BRIDGE_CONFIG,
  HALF_BRIDGE_CONFIG,
  QUARTER_BRIDGE_CONFIG,
]);

const bridgeZ = z.object({
  bridgeConfig: bridgeConfigZ,
  nominalBridgeResistance: z.number(),
});
interface Bridge extends z.infer<typeof bridgeZ> {}
const ZERO_BRIDGE: Bridge = {
  bridgeConfig: FULL_BRIDGE_CONFIG,
  nominalBridgeResistance: 0,
};

// https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaibridgechan.html
export const AI_BRIDGE_CHAN_TYPE = "ai_bridge";
export const aiBridgeChanZ = baseAIChanZ
  .merge(minMaxValZ)
  .merge(bridgeZ)
  .merge(voltageExcitZ)
  .merge(customScaleZ)
  .extend({
    type: z.literal(AI_BRIDGE_CHAN_TYPE),
    units: electricalUnitsZ,
    nominalBridgeResistance: z
      .number()
      .refine((val) => val > 0, { message: "Value must be greater than 0" }),
  });
interface AIBridgeChan extends z.infer<typeof aiBridgeChanZ> {}
export const ZERO_AI_BRIDGE_CHAN: AIBridgeChan = {
  ...ZERO_BASE_AI_CHAN,
  ...ZERO_MIN_MAX_VAL,
  ...ZERO_BRIDGE,
  ...ZERO_VOLTAGE_EXCIT,
  ...ZERO_CUSTOM_SCALE,
  type: AI_BRIDGE_CHAN_TYPE,
  units: MVOLTS_PER_VOLT,
  nominalBridgeResistance: 1,
};

const DEFAULT_SHUNT_RESISTOR_LOC = "Default";
const INTERNAL_SHUNT_RESISTOR_LOC = "Internal";
const EXTERNAL_SHUNT_RESISTOR_LOC = "External";
const shuntResistorLocZ = z.enum([
  DEFAULT_SHUNT_RESISTOR_LOC,
  INTERNAL_SHUNT_RESISTOR_LOC,
  EXTERNAL_SHUNT_RESISTOR_LOC,
]);
export type ShuntResistorLoc = z.infer<typeof shuntResistorLocZ>;

const shuntResistorValZ = z.number().refine((val) => val > 0, {
  message: "Value must be greater than 0",
});

// https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaicurrentchan.html
export const AI_CURRENT_CHAN_TYPE = "ai_current";
export const aiCurrentChanZ = baseAIChanZ
  .merge(terminalZ)
  .merge(minMaxValZ)
  .merge(customScaleZ)
  .extend({
    type: z.literal(AI_CURRENT_CHAN_TYPE),
    units: z.literal(AMPS),
    shuntResistorLoc: shuntResistorLocZ,
    extShuntResistorVal: shuntResistorValZ,
  });
interface AICurrentChan extends z.infer<typeof aiCurrentChanZ> {}
export const ZERO_AI_CURRENT_CHAN: AICurrentChan = {
  ...ZERO_BASE_AI_CHAN,
  ...ZERO_TERMINAL,
  ...ZERO_MIN_MAX_VAL,
  ...ZERO_CUSTOM_SCALE,
  type: AI_CURRENT_CHAN_TYPE,
  units: AMPS,
  shuntResistorLoc: DEFAULT_SHUNT_RESISTOR_LOC,
  extShuntResistorVal: 1,
};

const tableZ = z.object({
  electricalVals: z.array(z.number()),
  electricalUnits: electricalUnitsZ,
  physicalVals: z.array(z.number()),
});
type Table = z.infer<typeof tableZ>;
const ZERO_TABLE: Table = {
  electricalVals: [],
  electricalUnits: MVOLTS_PER_VOLT,
  physicalVals: [],
};

// https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaiforcebridgetablechan.html
export const AI_FORCE_BRIDGE_TABLE_CHAN_TYPE = "ai_force_bridge_table";
export const aiForceBridgeTableChanZ = baseAIChanZ
  .merge(minMaxValZ)
  .merge(bridgeZ)
  .merge(voltageExcitZ)
  .merge(tableZ)
  .merge(customScaleZ)
  .extend({
    type: z.literal(AI_FORCE_BRIDGE_TABLE_CHAN_TYPE),
    units: forceUnitsZ,
    physicalUnits: forceUnitsZ,
  });
interface AIForceBridgeTableChan extends z.infer<typeof aiForceBridgeTableChanZ> {}
export const ZERO_AI_FORCE_BRIDGE_TABLE_CHAN: AIForceBridgeTableChan = {
  ...ZERO_BASE_AI_CHAN,
  ...ZERO_MIN_MAX_VAL,
  ...ZERO_BRIDGE,
  ...ZERO_VOLTAGE_EXCIT,
  ...ZERO_TABLE,
  ...ZERO_CUSTOM_SCALE,
  type: AI_FORCE_BRIDGE_TABLE_CHAN_TYPE,
  units: NEWTONS,
  physicalUnits: NEWTONS,
};

const twoPointLinZ = z.object({
  firstElectricalVal: z.number(),
  secondElectricalVal: z.number(),
  electricalUnits: electricalUnitsZ,
  firstPhysicalVal: z.number(),
  secondPhysicalVal: z.number(),
});
type TwoPointLin = z.infer<typeof twoPointLinZ>;
const ZERO_TWO_POINT_LIN: TwoPointLin = {
  firstElectricalVal: 0,
  secondElectricalVal: 1,
  electricalUnits: MVOLTS_PER_VOLT,
  firstPhysicalVal: 0,
  secondPhysicalVal: 1,
};

// https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaiforcebridgetwopointlinchan.html
export const AI_FORCE_BRIDGE_TWO_POINT_LIN_CHAN_TYPE = "ai_force_bridge_two_point_lin";
export const aiForceBridgeTwoPointLinChan = baseAIChanZ
  .merge(minMaxValZ)
  .merge(bridgeZ)
  .merge(voltageExcitZ)
  .merge(twoPointLinZ)
  .merge(customScaleZ)
  .extend({
    type: z.literal(AI_FORCE_BRIDGE_TWO_POINT_LIN_CHAN_TYPE),
    units: forceUnitsZ,
    physicalUnits: forceUnitsZ,
  });
interface AIForceBridgeTwoPointLinChan
  extends z.infer<typeof aiForceBridgeTwoPointLinChan> {}
export const ZERO_AI_FORCE_BRIDGE_TWO_POINT_LIN_CHAN: AIForceBridgeTwoPointLinChan = {
  ...ZERO_BASE_AI_CHAN,
  ...ZERO_MIN_MAX_VAL,
  ...ZERO_BRIDGE,
  ...ZERO_VOLTAGE_EXCIT,
  ...ZERO_TWO_POINT_LIN,
  ...ZERO_CUSTOM_SCALE,
  type: AI_FORCE_BRIDGE_TWO_POINT_LIN_CHAN_TYPE,
  units: NEWTONS,
  physicalUnits: NEWTONS,
};

const MVOLTS_PER_NEWTON = "mVoltsPerNewton";
const MVOLTS_PER_LB = "mVoltsPerPound";
const forceSensitivityUnitsZ = z.enum([MVOLTS_PER_NEWTON, MVOLTS_PER_LB]);

// https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaiforceiepechan.html
export const AI_FORCE_IEPE_CHAN_TYPE = "ai_force_iepe";
export const aiForceIEPEChanZ = baseAIChanZ
  .merge(terminalZ)
  .merge(minMaxValZ)
  .merge(sensitivityZ)
  .merge(currentExcitZ)
  .merge(customScaleZ)
  .extend({
    type: z.literal(AI_FORCE_IEPE_CHAN_TYPE),
    units: forceUnitsZ,
    sensitivityUnits: forceSensitivityUnitsZ,
  });
interface AIForceIEPEChan extends z.infer<typeof aiForceIEPEChanZ> {}
export const ZERO_AI_FORCE_IEPE_CHAN: AIForceIEPEChan = {
  ...ZERO_BASE_AI_CHAN,
  ...ZERO_TERMINAL,
  ...ZERO_MIN_MAX_VAL,
  ...ZERO_SENSITIVITY,
  ...ZERO_CURRENT_EXCIT,
  ...ZERO_CUSTOM_SCALE,
  type: AI_FORCE_IEPE_CHAN_TYPE,
  units: NEWTONS,
  sensitivityUnits: MVOLTS_PER_NEWTON,
};

// https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaimicrophonechan.html
export const AI_MICROPHONE_CHAN_TYPE = "ai_microphone";
export const aiMicrophoneChanZ = baseAIChanZ
  .merge(terminalZ)
  .merge(currentExcitZ)
  .merge(customScaleZ)
  .extend({
    type: z.literal(AI_MICROPHONE_CHAN_TYPE),
    units: z.literal(PASCALS),
    micSensitivity: z.number(),
    maxSndPressLevel: z.number(),
  });
interface AIMicrophoneChan extends z.infer<typeof aiMicrophoneChanZ> {}
export const ZERO_AI_MICROPHONE_CHAN: AIMicrophoneChan = {
  ...ZERO_BASE_AI_CHAN,
  ...ZERO_TERMINAL,
  ...ZERO_CURRENT_EXCIT,
  ...ZERO_CUSTOM_SCALE,
  type: AI_MICROPHONE_CHAN_TYPE,
  units: PASCALS,
  micSensitivity: 0,
  maxSndPressLevel: 0,
};

const pressureUnitsZ = z.enum([LBS_PER_SQUARE_INCH, PASCALS, BAR]);
export type PressureUnits = z.infer<typeof pressureUnitsZ>;

// https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaipressurebridgetablechan.html
export const AI_PRESSURE_BRIDGE_TABLE_CHAN_TYPE = "ai_pressure_bridge_table";
export const aiPressureBridgeTableChanZ = baseAIChanZ
  .merge(minMaxValZ)
  .merge(bridgeZ)
  .merge(voltageExcitZ)
  .merge(tableZ)
  .merge(customScaleZ)
  .extend({
    type: z.literal(AI_PRESSURE_BRIDGE_TABLE_CHAN_TYPE),
    units: pressureUnitsZ,
    physicalUnits: pressureUnitsZ,
  });
interface AIPressureBridgeTableChan
  extends z.infer<typeof aiPressureBridgeTableChanZ> {}
export const ZERO_AI_PRESSURE_BRIDGE_TABLE_CHAN: AIPressureBridgeTableChan = {
  ...ZERO_BASE_AI_CHAN,
  ...ZERO_MIN_MAX_VAL,
  ...ZERO_BRIDGE,
  ...ZERO_VOLTAGE_EXCIT,
  ...ZERO_TABLE,
  ...ZERO_CUSTOM_SCALE,
  type: AI_PRESSURE_BRIDGE_TABLE_CHAN_TYPE,
  units: LBS_PER_SQUARE_INCH,
  physicalUnits: LBS_PER_SQUARE_INCH,
};

// https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaipressurebridgetwopointlinchan.html
export const AI_PRESSURE_BRIDGE_TWO_POINT_LIN_CHAN_TYPE =
  "ai_pressure_bridge_two_point_lin";
export const aiPressureBridgeTwoPointLinChanZ = baseAIChanZ
  .merge(minMaxValZ)
  .merge(bridgeZ)
  .merge(voltageExcitZ)
  .merge(twoPointLinZ)
  .merge(customScaleZ)
  .extend({
    type: z.literal(AI_PRESSURE_BRIDGE_TWO_POINT_LIN_CHAN_TYPE),
    units: pressureUnitsZ,
    physicalUnits: pressureUnitsZ,
  });
interface AIPressureBridgeTwoPointLinChan
  extends z.infer<typeof aiPressureBridgeTwoPointLinChanZ> {}
export const ZERO_AI_PRESSURE_BRIDGE_TWO_POINT_LIN_CHAN: AIPressureBridgeTwoPointLinChan =
  {
    ...ZERO_BASE_AI_CHAN,
    ...ZERO_MIN_MAX_VAL,
    ...ZERO_BRIDGE,
    ...ZERO_VOLTAGE_EXCIT,
    ...ZERO_TWO_POINT_LIN,
    ...ZERO_CUSTOM_SCALE,
    type: AI_PRESSURE_BRIDGE_TWO_POINT_LIN_CHAN_TYPE,
    units: LBS_PER_SQUARE_INCH,
    physicalUnits: LBS_PER_SQUARE_INCH,
  };

const RESISTANCE_CONFIG_2_WIRE = "2Wire";
const RESISTANCE_CONFIG_3_WIRE = "3Wire";
const RESISTANCE_CONFIG_4_WIRE = "4Wire";
const resistanceConfigZ = z.enum([
  RESISTANCE_CONFIG_2_WIRE,
  RESISTANCE_CONFIG_3_WIRE,
  RESISTANCE_CONFIG_4_WIRE,
]);

const resistanceZ = z.object({ resistanceConfig: resistanceConfigZ });
type Resistance = z.infer<typeof resistanceZ>;
const ZERO_RESISTANCE: Resistance = { resistanceConfig: RESISTANCE_CONFIG_2_WIRE };

// https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateairesistancechan.html
export const AI_RESISTANCE_CHAN_TYPE = "ai_resistance";
export const aiResistanceChanZ = baseAIChanZ
  .merge(minMaxValZ)
  .merge(resistanceZ)
  .merge(currentExcitZ)
  .merge(customScaleZ)
  .extend({ type: z.literal(AI_RESISTANCE_CHAN_TYPE), units: z.literal(OHMS) });
interface AIResistanceChan extends z.infer<typeof aiResistanceChanZ> {}
export const ZERO_AI_RESISTANCE_CHAN: AIResistanceChan = {
  ...ZERO_BASE_AI_CHAN,
  ...ZERO_MIN_MAX_VAL,
  ...ZERO_RESISTANCE,
  ...ZERO_CURRENT_EXCIT,
  ...ZERO_CUSTOM_SCALE,
  type: AI_RESISTANCE_CHAN_TYPE,
  units: OHMS,
};

const temperatureUnitsZ = z.enum([DEG_C, DEG_F, KELVINS, DEG_R]);
export type TemperatureUnits = z.infer<typeof temperatureUnitsZ>;

const PT3750 = "Pt3750";
const PT3851 = "Pt3851";
const PT3911 = "Pt3911";
const PT3916 = "Pt3916";
const PT3920 = "Pt3920";
const PT3928 = "Pt3928";
const rtdTypeZ = z.enum([PT3750, PT3851, PT3911, PT3916, PT3920, PT3928]);

// https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateairtdchan.html
export const AI_RTD_CHAN_TYPE = "ai_rtd";
export const aiRTDChanZ = baseAIChanZ
  .merge(minMaxValZ)
  .merge(resistanceZ)
  .merge(currentExcitZ)
  .extend({
    type: z.literal(AI_RTD_CHAN_TYPE),
    units: temperatureUnitsZ,
    rtdType: rtdTypeZ,
    r0: z.number(),
  });
interface AIRTDChan extends z.infer<typeof aiRTDChanZ> {}
export const ZERO_AI_RTD_CHAN: AIRTDChan = {
  ...ZERO_BASE_AI_CHAN,
  ...ZERO_MIN_MAX_VAL,
  ...ZERO_RESISTANCE,
  ...ZERO_CURRENT_EXCIT,
  type: AI_RTD_CHAN_TYPE,
  units: DEG_C,
  rtdType: PT3750,
  r0: 0,
};

const FULL_BRIDGE_I = "full-bridge-I";
const FULL_BRIDGE_II = "full-bridge-II";
const FULL_BRIDGE_III = "full-bridge-III";
const HALF_BRIDGE_I = "half-bridge-I";
const HALF_BRIDGE_II = "half-bridge-II";
const QUARTER_BRIDGE_I = "quarter-bridge-I";
const QUARTER_BRIDGE_II = "quarter-bridge-II";
const strainConfigZ = z.enum([
  FULL_BRIDGE_I,
  FULL_BRIDGE_II,
  FULL_BRIDGE_III,
  HALF_BRIDGE_I,
  HALF_BRIDGE_II,
  QUARTER_BRIDGE_I,
  QUARTER_BRIDGE_II,
]);

// https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaistraingagechan.html
export const AI_STRAIN_GAGE_CHAN_TYPE = "ai_strain_gauge";
export const aiStrainGageChanZ = baseAIChanZ
  .merge(minMaxValZ)
  .merge(voltageExcitZ)
  .merge(customScaleZ)
  .extend({
    type: z.literal(AI_STRAIN_GAGE_CHAN_TYPE),
    units: z.literal(STRAIN),
    strainConfig: strainConfigZ,
    gageFactor: z.number(),
    initialBridgeVoltage: z.number(),
    nominalGageResistance: z.number(),
    poissonRatio: z.number(),
    leadWireResistance: z.number(),
  });
interface AIStrainGageChan extends z.infer<typeof aiStrainGageChanZ> {}
export const ZERO_AI_STRAIN_GAGE_CHAN: AIStrainGageChan = {
  ...ZERO_BASE_AI_CHAN,
  ...ZERO_MIN_MAX_VAL,
  ...ZERO_VOLTAGE_EXCIT,
  ...ZERO_CUSTOM_SCALE,
  type: AI_STRAIN_GAGE_CHAN_TYPE,
  units: STRAIN,
  strainConfig: FULL_BRIDGE_I,
  gageFactor: 0,
  initialBridgeVoltage: 0,
  nominalGageResistance: 0,
  poissonRatio: 0,
  leadWireResistance: 0,
};

// https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaitempbuiltinsensorchan.html
export const AI_TEMP_BUILT_IN_CHAN_TYPE = "ai_temp_builtin";
export const aiTempBuiltInChanZ = baseAIChanZ.extend({
  type: z.literal(AI_TEMP_BUILT_IN_CHAN_TYPE),
  units: temperatureUnitsZ,
});
interface AITempBuiltInChan extends z.infer<typeof aiTempBuiltInChanZ> {}
export const ZERO_AI_TEMP_BUILT_IN_CHAN: AITempBuiltInChan = {
  ...ZERO_BASE_AI_CHAN,
  type: AI_TEMP_BUILT_IN_CHAN_TYPE,
  units: DEG_C,
};

const J_TYPE_TC = "J";
const K_TYPE_TC = "K";
const N_TYPE_TC = "N";
const R_TYPE_TC = "R";
const S_TYPE_TC = "S";
const T_TYPE_TC = "T";
const B_TYPE_TC = "B";
const E_TYPE_TC = "E";
const thermocoupleTypeZ = z.enum([
  J_TYPE_TC,
  K_TYPE_TC,
  N_TYPE_TC,
  R_TYPE_TC,
  S_TYPE_TC,
  T_TYPE_TC,
  B_TYPE_TC,
  E_TYPE_TC,
]);

const BUILT_IN = "BuiltIn";
export const CONST_VAL = "ConstVal";
export const CHAN = "Chan";
const cjcSourceZ = z.enum([BUILT_IN, CONST_VAL, CHAN]);

// https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaithrmcplchan.html
export const AI_THRMCPL_CHAN_TYPE = "ai_thermocouple";
export const aiThrmcplChanZ = baseAIChanZ.merge(minMaxValZ).extend({
  type: z.literal(AI_THRMCPL_CHAN_TYPE),
  units: temperatureUnitsZ,
  thermocoupleType: thermocoupleTypeZ,
  cjcSource: cjcSourceZ,
  cjcVal: z.number(),
  cjcPort: z.number(),
});
interface AIThrmcplChan extends z.infer<typeof aiThrmcplChanZ> {}
export const ZERO_AI_THRMCPL_CHAN: AIThrmcplChan = {
  ...ZERO_BASE_AI_CHAN,
  ...ZERO_MIN_MAX_VAL,
  type: AI_THRMCPL_CHAN_TYPE,
  units: DEG_C,
  thermocoupleType: J_TYPE_TC,
  cjcSource: BUILT_IN,
  cjcVal: 0,
  cjcPort: 0,
};

const torqueUnitsZ = z.enum([NEWTON_METERS, IN_OZ, INCH_LBS, FT_LBS]);
export type TorqueUnits = z.infer<typeof torqueUnitsZ>;

// https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaitorquebridgetablechan.html
export const AI_TORQUE_BRIDGE_TABLE_CHAN_TYPE = "ai_torque_bridge_table";
export const aiTorqueBridgeTableChanZ = baseAIChanZ
  .merge(minMaxValZ)
  .merge(bridgeZ)
  .merge(voltageExcitZ)
  .merge(tableZ)
  .merge(customScaleZ)
  .extend({
    type: z.literal(AI_TORQUE_BRIDGE_TABLE_CHAN_TYPE),
    units: torqueUnitsZ,
    physicalUnits: torqueUnitsZ,
  });
interface AITorqueBridgeTableChan extends z.infer<typeof aiTorqueBridgeTableChanZ> {}
export const ZERO_AI_TORQUE_BRIDGE_TABLE_CHAN: AITorqueBridgeTableChan = {
  ...ZERO_BASE_AI_CHAN,
  ...ZERO_MIN_MAX_VAL,
  ...ZERO_BRIDGE,
  ...ZERO_VOLTAGE_EXCIT,
  ...ZERO_TABLE,
  ...ZERO_CUSTOM_SCALE,
  type: AI_TORQUE_BRIDGE_TABLE_CHAN_TYPE,
  units: NEWTON_METERS,
  physicalUnits: NEWTON_METERS,
};

// https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaitorquebridgetwopointlinchan.html
export const AI_TORQUE_BRIDGE_TWO_POINT_LIN_CHAN_TYPE =
  "ai_torque_bridge_two_point_lin";
export const aiTorqueBridgeTwoPointLinChanZ = baseAIChanZ
  .merge(minMaxValZ)
  .merge(bridgeZ)
  .merge(voltageExcitZ)
  .merge(twoPointLinZ)
  .merge(customScaleZ)
  .extend({
    type: z.literal(AI_TORQUE_BRIDGE_TWO_POINT_LIN_CHAN_TYPE),
    units: torqueUnitsZ,
    physicalUnits: torqueUnitsZ,
  });
interface AITorqueBridgeTwoPointLinChan
  extends z.infer<typeof aiTorqueBridgeTwoPointLinChanZ> {}
export const ZERO_AI_TORQUE_BRIDGE_TWO_POINT_LIN_CHAN: AITorqueBridgeTwoPointLinChan = {
  ...ZERO_BASE_AI_CHAN,
  ...ZERO_MIN_MAX_VAL,
  ...ZERO_BRIDGE,
  ...ZERO_VOLTAGE_EXCIT,
  ...ZERO_TWO_POINT_LIN,
  ...ZERO_CUSTOM_SCALE,
  type: AI_TORQUE_BRIDGE_TWO_POINT_LIN_CHAN_TYPE,
  units: NEWTON_METERS,
  physicalUnits: NEWTON_METERS,
};

const METERS_PER_SECOND = "MetersPerSecond";
const INCHES_PER_SECOND = "InchesPerSecond";
const velocityUnitsZ = z.enum([METERS_PER_SECOND, INCHES_PER_SECOND]);
export type VelocityUnits = z.infer<typeof velocityUnitsZ>;

const MVOLTS_PER_MM_PER_SECOND = "MillivoltsPerMillimeterPerSecond";
const MVOLTS_PER_INCH_PER_SECOND = "MilliVoltsPerInchPerSecond";
const velocitySensitivityUnitsZ = z.enum([
  MVOLTS_PER_MM_PER_SECOND,
  MVOLTS_PER_INCH_PER_SECOND,
]);
export type VelocitySensitivityUnits = z.infer<typeof velocitySensitivityUnitsZ>;

// https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaivelocityiepechan.html
export const AI_VELOCITY_IEPE_CHAN_TYPE = "ai_velocity_iepe";
export const aiVelocityIEPEChanZ = baseAIChanZ
  .merge(terminalZ)
  .merge(minMaxValZ)
  .merge(sensitivityZ)
  .merge(currentExcitZ)
  .merge(customScaleZ)
  .extend({
    type: z.literal(AI_VELOCITY_IEPE_CHAN_TYPE),
    units: velocityUnitsZ,
    sensitivityUnits: velocitySensitivityUnitsZ,
  });
interface AIVelocityIEPEChan extends z.infer<typeof aiVelocityIEPEChanZ> {}
export const ZERO_AI_VELOCITY_IEPE_CHAN: AIVelocityIEPEChan = {
  ...ZERO_BASE_AI_CHAN,
  ...ZERO_TERMINAL,
  ...ZERO_MIN_MAX_VAL,
  ...ZERO_SENSITIVITY,
  ...ZERO_CURRENT_EXCIT,
  ...ZERO_CUSTOM_SCALE,
  type: AI_VELOCITY_IEPE_CHAN_TYPE,
  units: METERS_PER_SECOND,
  sensitivityUnits: MVOLTS_PER_MM_PER_SECOND,
};

// https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaivoltagechan.html
export const AI_VOLTAGE_CHAN_TYPE = "ai_voltage";
export const aiVoltageChanZ = baseAIChanZ
  .merge(terminalZ)
  .merge(minMaxValZ)
  .merge(customScaleZ)
  .extend({ type: z.literal(AI_VOLTAGE_CHAN_TYPE), units: z.literal(VOLTS) });
interface AIVoltageChan extends z.infer<typeof aiVoltageChanZ> {}
export const ZERO_AI_VOLTAGE_CHAN: AIVoltageChan = {
  ...ZERO_BASE_AI_CHAN,
  ...ZERO_TERMINAL,
  ...ZERO_MIN_MAX_VAL,
  ...ZERO_CUSTOM_SCALE,
  type: AI_VOLTAGE_CHAN_TYPE,
  units: VOLTS,
};

const aiChanZ = z.union([
  aiAccelChanZ,
  aiBridgeChanZ,
  aiCurrentChanZ,
  aiForceBridgeTableChanZ,
  aiForceBridgeTwoPointLinChan,
  aiForceIEPEChanZ,
  aiMicrophoneChanZ,
  aiPressureBridgeTableChanZ,
  aiPressureBridgeTwoPointLinChanZ,
  aiResistanceChanZ,
  aiRTDChanZ,
  aiStrainGageChanZ,
  aiTempBuiltInChanZ,
  aiThrmcplChanZ,
  aiTorqueBridgeTableChanZ,
  aiTorqueBridgeTwoPointLinChanZ,
  aiVelocityIEPEChanZ,
  aiVoltageChanZ,
]);

type AIChan = z.infer<typeof aiChanZ>;
export type AIChanType = AIChan["type"];

export const AI_CHAN_TYPE_NAMES: Record<AIChanType, string> = {
  [AI_ACCEL_CHAN_TYPE]: "Accelerometer",
  [AI_BRIDGE_CHAN_TYPE]: "Bridge",
  [AI_CURRENT_CHAN_TYPE]: "Current",
  [AI_FORCE_BRIDGE_TABLE_CHAN_TYPE]: "Force Bridge Table",
  [AI_FORCE_BRIDGE_TWO_POINT_LIN_CHAN_TYPE]: "Force Bridge Two-Point Linear",
  [AI_FORCE_IEPE_CHAN_TYPE]: "Force IEPE",
  [AI_MICROPHONE_CHAN_TYPE]: "Microphone",
  [AI_PRESSURE_BRIDGE_TABLE_CHAN_TYPE]: "Pressure Bridge Table",
  [AI_PRESSURE_BRIDGE_TWO_POINT_LIN_CHAN_TYPE]: "Pressure Bridge Two-Point Linear",
  [AI_RESISTANCE_CHAN_TYPE]: "Resistance",
  [AI_RTD_CHAN_TYPE]: "RTD",
  [AI_STRAIN_GAGE_CHAN_TYPE]: "Strain Gage",
  [AI_TEMP_BUILT_IN_CHAN_TYPE]: "Temperature Built-In Sensor",
  [AI_THRMCPL_CHAN_TYPE]: "Thermocouple",
  [AI_TORQUE_BRIDGE_TABLE_CHAN_TYPE]: "Torque Bridge Table",
  [AI_TORQUE_BRIDGE_TWO_POINT_LIN_CHAN_TYPE]: "Torque Bridge Two-Point Linear",
  [AI_VELOCITY_IEPE_CHAN_TYPE]: "Velocity IEPE",
  [AI_VOLTAGE_CHAN_TYPE]: "Voltage",
};

// Digital Output Channels

const DO_CHAN_TYPE = "digital_output";
const doChanZ = baseChanZ.extend({
  type: z.literal(DO_CHAN_TYPE),
  cmdChannel: channel.keyZ,
  stateChannel: channel.keyZ,
  line: z.number(),
});
export interface DOChan extends z.infer<typeof doChanZ> {}
export const ZERO_DO_CHAN: DOChan = {
  ...ZERO_BASE_CHAN,
  type: DO_CHAN_TYPE,
  cmdChannel: 0,
  stateChannel: 0,
  line: 0,
};

// Digital Input Channels

const DI_CHAN_TYPE = "digital_input";
const diChanZ = baseChanZ.extend({
  type: z.literal(DI_CHAN_TYPE),
  line: z.number(),
  channel: channel.keyZ,
});
export interface DIChan extends z.infer<typeof diChanZ> {}
export const ZERO_DI_CHAN: DIChan = {
  ...ZERO_BASE_CHAN,
  type: DI_CHAN_TYPE,
  line: 0,
  channel: 0,
};

// Tasks

const deviceKeyZ = device.deviceKeyZ.min(1, "Must specify a device");

const baseConfigZ = z.object({ device: deviceKeyZ, dataSaving: z.boolean() });
interface BaseConfig extends z.infer<typeof baseConfigZ> {}
const ZERO_BASE_CONFIG: BaseConfig = { device: "", dataSaving: true };

type BaseDetails = { running: boolean };

// Analog Read Task

export const analogReadConfigZ = baseConfigZ.extend({
  version: z.literal(VERSION).optional().default(VERSION),
  sampleRate: z.number().min(0).max(50000),
  streamRate: z.number().min(0).max(50000),
  channels: z.array(aiChanZ),
});
export interface AnalogReadConfig extends z.infer<typeof analogReadConfigZ> {}
export const ZERO_ANALOG_READ_CONFIG: AnalogReadConfig = {
  ...ZERO_BASE_CONFIG,
  version: VERSION,
  sampleRate: 10,
  streamRate: 5,
  channels: [],
};

type BaseAnalogReadDetails = BaseDetails & { message: string };
type ErrorAnalogReadDetails = BaseAnalogReadDetails & {
  errors: { message: string; path: string }[];
};
export type AnalogReadDetails = BaseAnalogReadDetails | ErrorAnalogReadDetails;

export const ANALOG_READ_TYPE = `${PREFIX}_analog_read`;
export type AnalogReadType = typeof ANALOG_READ_TYPE;

interface AnalogReadPayload
  extends task.Payload<AnalogReadConfig, AnalogReadDetails, AnalogReadType> {}
export const ZERO_ANALOG_READ_PAYLOAD: AnalogReadPayload = {
  key: "",
  name: "NI Analog Read Task",
  config: ZERO_ANALOG_READ_CONFIG,
  type: ANALOG_READ_TYPE,
};

// Digital Write Task

export const digitalWriteConfigZ = baseConfigZ.extend({
  channels: z.array(doChanZ),
  stateRate: z.number().min(0).max(50000),
});
export interface DigitalWriteConfig extends z.infer<typeof digitalWriteConfigZ> {}
const ZERO_DIGITAL_WRITE_CONFIG: DigitalWriteConfig = {
  ...ZERO_BASE_CONFIG,
  stateRate: 10,
  channels: [],
};

export interface DigitalWriteDetails extends BaseDetails {}

export const DIGITAL_WRITE_TYPE = `${PREFIX}_digital_write`;
export type DigitalWriteType = typeof DIGITAL_WRITE_TYPE;

export interface DigitalWrite
  extends task.Task<DigitalWriteConfig, DigitalWriteDetails, DigitalWriteType> {}
export interface DigitalWritePayload
  extends task.Payload<DigitalWriteConfig, DigitalWriteDetails, DigitalWriteType> {}
export const ZERO_DIGITAL_WRITE_PAYLOAD: DigitalWritePayload = {
  key: "",
  name: "NI Digital Write Task",
  config: ZERO_DIGITAL_WRITE_CONFIG,
  type: DIGITAL_WRITE_TYPE,
};

// Digital Read Task

export const digitalReadConfigZ = baseConfigZ.extend({
  sampleRate: z.number().min(0).max(50000),
  streamRate: z.number().min(0).max(50000),
  channels: z.array(diChanZ),
});
export interface DigitalReadConfig extends z.infer<typeof digitalReadConfigZ> {}
const ZERO_DIGITAL_READ_CONFIG: DigitalReadConfig = {
  ...ZERO_BASE_CONFIG,
  channels: [],
  sampleRate: 50,
  streamRate: 25,
};

export interface DigitalReadDetails extends BaseDetails {}

export const DIGITAL_READ_TYPE = `${PREFIX}_digital_read`;
export type DigitalReadType = typeof DIGITAL_READ_TYPE;

export interface DigitalRead
  extends task.Task<DigitalReadConfig, DigitalReadDetails, DigitalReadType> {}
export interface DigitalReadPayload
  extends task.Payload<DigitalReadConfig, DigitalReadDetails, DigitalReadType> {}
export const ZERO_DIGITAL_READ_PAYLOAD: DigitalReadPayload = {
  key: "",
  name: "NI Digital Read Task",
  config: ZERO_DIGITAL_READ_CONFIG,
  type: DIGITAL_READ_TYPE,
};

// Scan Task

export type ScanConfig = { enabled: boolean };
