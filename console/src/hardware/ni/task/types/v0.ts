// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type task } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/pluto";
import { z } from "zod";

import { Common } from "@/hardware/common";

export const PREFIX = "ni";

const portZ = z.number().int().nonnegative();
const lineZ = z.number().int().nonnegative();

const analogChannelExtensionShape = { port: portZ };
interface AnalogChannelExtension
  extends z.infer<z.ZodObject<typeof analogChannelExtensionShape>> {}
const ZERO_ANALOG_CHANNEL_EXTENSION: AnalogChannelExtension = { port: 0 };

const digitalChannelExtensionShape = {
  port: portZ,
  line: lineZ,
};
interface DigitalChannelExtension
  extends z.infer<z.ZodObject<typeof digitalChannelExtensionShape>> {}
const ZERO_DIGITAL_CHANNEL_EXTENSION: DigitalChannelExtension = { port: 0, line: 0 };

const baseAIChanZ = Common.Task.readChannelZ.extend(analogChannelExtensionShape);
interface BaseAIChan extends z.infer<typeof baseAIChanZ> {}
const ZERO_BASE_AI_CHAN: BaseAIChan = {
  ...Common.Task.ZERO_READ_CHANNEL,
  ...ZERO_ANALOG_CHANNEL_EXTENSION,
};

const minMaxValShape = { minVal: z.number(), maxVal: z.number() };
interface MinMaxVal extends z.infer<z.ZodObject<typeof minMaxValShape>> {}
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

const terminalShape = { terminalConfig: terminalConfigZ };
interface Terminal extends z.infer<z.ZodObject<typeof terminalShape>> {}
const ZERO_TERMINAL: Terminal = { terminalConfig: DEFAULT_TERMINAL_CONFIG };

const INTERNAL_EXCIT_SOURCE = "Internal";
const EXTERNAL_EXCIT_SOURCE = "External";
const NONE_EXCIT_SOURCE = "None";
const excitSourceZ = z.enum([
  INTERNAL_EXCIT_SOURCE,
  EXTERNAL_EXCIT_SOURCE,
  NONE_EXCIT_SOURCE,
]);

const currentExcitShape = {
  currentExcitSource: excitSourceZ,
  currentExcitVal: z.number(),
};
interface CurrentExcit extends z.infer<z.ZodObject<typeof currentExcitShape>> {}
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
const FROM_CUSTOM_SCALE = "FromCustomScale";
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

export const LINEAR_SCALE_TYPE = "linear";
const linearScaleZ = z.object({
  type: z.literal(LINEAR_SCALE_TYPE),
  slope: z.number(),
  yIntercept: z.number(),
  preScaledUnits: unitsZ,
  scaledUnits: z.string(),
});
interface LinearScale extends z.infer<typeof linearScaleZ> {}
const ZERO_LINEAR_SCALE: LinearScale = {
  type: LINEAR_SCALE_TYPE,
  slope: 1,
  yIntercept: 0,
  preScaledUnits: VOLTS,
  scaledUnits: VOLTS,
};

export const MAP_SCALE_TYPE = "map";
const mapScaleZ = z
  .object({
    type: z.literal(MAP_SCALE_TYPE),
    preScaledMin: z.number(),
    preScaledMax: z.number(),
    scaledMin: z.number(),
    scaledMax: z.number(),
    preScaledUnits: unitsZ,
    scaledUnits: z.string(),
  })
  .refine(({ preScaledMin, preScaledMax }) => preScaledMin < preScaledMax, {
    message: "Pre-scaled min must be less than pre-scaled max",
    path: ["preScaledMin"],
  })
  .refine(({ scaledMin, scaledMax }) => scaledMin < scaledMax, {
    message: "Scaled min must be less than scaled max",
    path: ["scaledMin"],
  });
interface MapScale extends z.infer<typeof mapScaleZ> {}
const ZERO_MAP_SCALE: MapScale = {
  type: MAP_SCALE_TYPE,
  preScaledMin: 0,
  preScaledMax: 1,
  scaledMin: 0,
  scaledMax: 1,
  preScaledUnits: VOLTS,
  scaledUnits: VOLTS,
};

export const TABLE_SCALE_TYPE = "table";
const tableScaleZ = z
  .object({
    type: z.literal(TABLE_SCALE_TYPE),
    preScaledVals: z.number().array(),
    scaledVals: z.number().array(),
    preScaledUnits: unitsZ,
    scaledUnits: z.string(),
  })
  .check(({ value, issues }) => {
    const { preScaledVals, scaledVals } = value;
    if (preScaledVals.length !== scaledVals.length) {
      const code = "custom";
      const message = "Pre-scaled and scaled values must have the same length";
      issues.push({ path: ["preScaledVals"], code, message, input: value });
      issues.push({ path: ["scaledVals"], code, message, input: value });
    }
  })
  .check(({ value, issues }) => {
    const { preScaledVals } = value;
    if (preScaledVals.length === 0) return;
    let lastVal = preScaledVals[0];
    for (let i = 1; i < preScaledVals.length; i++) {
      if (preScaledVals[i] <= lastVal)
        issues.push({
          code: "custom",
          message: "Pre-scaled values must be monotonically increasing",
          path: ["preScaledVals"],
          input: value,
        });
      lastVal = preScaledVals[i];
    }
  });
interface TableScale extends z.infer<typeof tableScaleZ> {}
const ZERO_TABLE_SCALE: TableScale = {
  type: TABLE_SCALE_TYPE,
  preScaledVals: [],
  scaledVals: [],
  preScaledUnits: VOLTS,
  scaledUnits: VOLTS,
};

export const NO_SCALE_TYPE = "none";
const noScaleZ = z.object({ type: z.literal(NO_SCALE_TYPE) });
interface NoScale extends z.infer<typeof noScaleZ> {}
const NO_SCALE: NoScale = { type: NO_SCALE_TYPE };

const scaleZ = z.union([linearScaleZ, mapScaleZ, tableScaleZ, noScaleZ]);
export type Scale = z.infer<typeof scaleZ>;
export type ScaleType = Scale["type"];

const customScaleShape = { customScale: scaleZ };
interface CustomScale extends z.infer<z.ZodObject<typeof customScaleShape>> {}
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

const sensitivityShape = { sensitivity: z.number() };
interface Sensitivity extends z.infer<z.ZodObject<typeof sensitivityShape>> {}
const ZERO_SENSITIVITY: Sensitivity = { sensitivity: 0 };

// https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaiaccelchan.html
export const AI_ACCEL_CHAN_TYPE = "ai_accel";
export const aiAccelChanZ = baseAIChanZ.extend({
  ...terminalShape,
  ...minMaxValShape,
  ...sensitivityShape,
  ...currentExcitShape,
  ...customScaleShape,
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

const voltageExcitShape = {
  voltageExcitSource: excitSourceZ,
  voltageExcitVal: z.number(),
};
interface VoltageExcit extends z.infer<z.ZodObject<typeof voltageExcitShape>> {}
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

const bridgeShape = {
  bridgeConfig: bridgeConfigZ,
  nominalBridgeResistance: z.number(),
};
interface Bridge extends z.infer<z.ZodObject<typeof bridgeShape>> {}
const ZERO_BRIDGE: Bridge = {
  bridgeConfig: FULL_BRIDGE_CONFIG,
  nominalBridgeResistance: 0,
};

// https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaibridgechan.html
export const AI_BRIDGE_CHAN_TYPE = "ai_bridge";
export const aiBridgeChanZ = baseAIChanZ.extend({
  ...minMaxValShape,
  ...bridgeShape,
  ...voltageExcitShape,
  ...customScaleShape,
  type: z.literal(AI_BRIDGE_CHAN_TYPE),
  units: electricalUnitsZ,
  nominalBridgeResistance: z
    .number()
    .refine((val) => val > 0, { message: "Value must be greater than 0" }),
});
export interface AIBridgeChan extends z.infer<typeof aiBridgeChanZ> {}
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

const shuntResistorValZ = z
  .number()
  .positive({ message: "Value must be greater than 0" });

// https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaicurrentchan.html
export const AI_CURRENT_CHAN_TYPE = "ai_current";
export const aiCurrentChanZ = baseAIChanZ.extend({
  ...terminalShape,
  ...minMaxValShape,
  ...customScaleShape,
  type: z.literal(AI_CURRENT_CHAN_TYPE),
  units: z.literal(AMPS),
  shuntResistorLoc: shuntResistorLocZ,
  extShuntResistorVal: shuntResistorValZ,
});
export interface AICurrentChan extends z.infer<typeof aiCurrentChanZ> {}
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

const tableShape = {
  electricalVals: z.array(z.number()),
  electricalUnits: electricalUnitsZ,
  physicalVals: z.array(z.number()),
};
interface Table extends z.infer<z.ZodObject<typeof tableShape>> {}
const ZERO_TABLE: Table = {
  electricalVals: [],
  electricalUnits: MVOLTS_PER_VOLT,
  physicalVals: [],
};

// https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaiforcebridgetablechan.html
export const AI_FORCE_BRIDGE_TABLE_CHAN_TYPE = "ai_force_bridge_table";
export const aiForceBridgeTableChanZ = baseAIChanZ.extend({
  ...minMaxValShape,
  ...bridgeShape,
  ...voltageExcitShape,
  ...tableShape,
  ...customScaleShape,
  type: z.literal(AI_FORCE_BRIDGE_TABLE_CHAN_TYPE),
  units: forceUnitsZ,
  physicalUnits: forceUnitsZ,
});
export interface AIForceBridgeTableChan
  extends z.infer<typeof aiForceBridgeTableChanZ> {}
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

const twoPointLinShape = {
  firstElectricalVal: z.number(),
  secondElectricalVal: z.number(),
  electricalUnits: electricalUnitsZ,
  firstPhysicalVal: z.number(),
  secondPhysicalVal: z.number(),
};
interface TwoPointLin extends z.infer<z.ZodObject<typeof twoPointLinShape>> {}
const ZERO_TWO_POINT_LIN: TwoPointLin = {
  firstElectricalVal: 0,
  secondElectricalVal: 1,
  electricalUnits: MVOLTS_PER_VOLT,
  firstPhysicalVal: 0,
  secondPhysicalVal: 1,
};

// https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaiforcebridgetwopointlinchan.html
export const AI_FORCE_BRIDGE_TWO_POINT_LIN_CHAN_TYPE = "ai_force_bridge_two_point_lin";
export const aiForceBridgeTwoPointLinChanZ = baseAIChanZ.extend({
  ...minMaxValShape,
  ...bridgeShape,
  ...voltageExcitShape,
  ...twoPointLinShape,
  ...customScaleShape,
  type: z.literal(AI_FORCE_BRIDGE_TWO_POINT_LIN_CHAN_TYPE),
  units: forceUnitsZ,
  physicalUnits: forceUnitsZ,
});
export interface AIForceBridgeTwoPointLinChan
  extends z.infer<typeof aiForceBridgeTwoPointLinChanZ> {}
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
export const aiForceIEPEChanZ = baseAIChanZ.extend({
  ...terminalShape,
  ...minMaxValShape,
  ...sensitivityShape,
  ...currentExcitShape,
  ...customScaleShape,
  type: z.literal(AI_FORCE_IEPE_CHAN_TYPE),
  units: forceUnitsZ,
  sensitivityUnits: forceSensitivityUnitsZ,
});
export interface AIForceIEPEChan extends z.infer<typeof aiForceIEPEChanZ> {}
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
export const aiMicrophoneChanZ = baseAIChanZ.extend({
  ...terminalShape,
  ...currentExcitShape,
  ...customScaleShape,
  type: z.literal(AI_MICROPHONE_CHAN_TYPE),
  units: z.literal(PASCALS),
  micSensitivity: z.number(),
  maxSndPressLevel: z.number(),
});
export interface AIMicrophoneChan extends z.infer<typeof aiMicrophoneChanZ> {}
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
export const aiPressureBridgeTableChanZ = baseAIChanZ.extend({
  ...minMaxValShape,
  ...bridgeShape,
  ...voltageExcitShape,
  ...tableShape,
  ...customScaleShape,
  type: z.literal(AI_PRESSURE_BRIDGE_TABLE_CHAN_TYPE),
  units: pressureUnitsZ,
  physicalUnits: pressureUnitsZ,
});
export interface AIPressureBridgeTableChan
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
export const aiPressureBridgeTwoPointLinChanZ = baseAIChanZ.extend({
  ...minMaxValShape,
  ...bridgeShape,
  ...voltageExcitShape,
  ...twoPointLinShape,
  ...customScaleShape,
  type: z.literal(AI_PRESSURE_BRIDGE_TWO_POINT_LIN_CHAN_TYPE),
  units: pressureUnitsZ,
  physicalUnits: pressureUnitsZ,
});
export interface AIPressureBridgeTwoPointLinChan
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

const resistanceShape = { resistanceConfig: resistanceConfigZ };
interface Resistance extends z.infer<z.ZodObject<typeof resistanceShape>> {}
const ZERO_RESISTANCE: Resistance = { resistanceConfig: RESISTANCE_CONFIG_2_WIRE };

// https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateairesistancechan.html
export const AI_RESISTANCE_CHAN_TYPE = "ai_resistance";
export const aiResistanceChanZ = baseAIChanZ.extend({
  ...minMaxValShape,
  ...resistanceShape,
  ...currentExcitShape,
  ...customScaleShape,
  type: z.literal(AI_RESISTANCE_CHAN_TYPE),
  units: z.literal(OHMS),
});
export interface AIResistanceChan extends z.infer<typeof aiResistanceChanZ> {}
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
export const aiRTDChanZ = baseAIChanZ.extend({
  ...minMaxValShape,
  ...resistanceShape,
  ...currentExcitShape,
  type: z.literal(AI_RTD_CHAN_TYPE),
  units: temperatureUnitsZ,
  rtdType: rtdTypeZ,
  r0: z.number(),
});
export interface AIRTDChan extends z.infer<typeof aiRTDChanZ> {}
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

const FULL_BRIDGE_I = "FullBridgeI";
const FULL_BRIDGE_II = "FullBridgeII";
const FULL_BRIDGE_III = "FullBridgeIII";
const HALF_BRIDGE_I = "HalfBridgeI";
const HALF_BRIDGE_II = "HalfBridgeII";
const QUARTER_BRIDGE_I = "QuarterBridgeI";
const QUARTER_BRIDGE_II = "QuarterBridgeII";
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
export const aiStrainGageChanZ = baseAIChanZ.extend({
  ...minMaxValShape,
  ...voltageExcitShape,
  ...customScaleShape,
  type: z.literal(AI_STRAIN_GAGE_CHAN_TYPE),
  units: z.literal(STRAIN),
  strainConfig: strainConfigZ,
  gageFactor: z.number(),
  initialBridgeVoltage: z.number(),
  nominalGageResistance: z.number(),
  poissonRatio: z.number(),
  leadWireResistance: z.number(),
});
export interface AIStrainGageChan extends z.infer<typeof aiStrainGageChanZ> {}
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
export interface AITempBuiltInChan extends z.infer<typeof aiTempBuiltInChanZ> {}
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
const CONST_VAL = "ConstVal";
const CHAN = "Chan";

// https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaithrmcplchan.html
export const AI_THRMCPL_CHAN_TYPE = "ai_thermocouple";
const baseAIThrmcplChanZ = baseAIChanZ.extend(minMaxValShape).extend({
  type: z.literal(AI_THRMCPL_CHAN_TYPE),
  units: temperatureUnitsZ,
  thermocoupleType: thermocoupleTypeZ,
});

const aiThrmcplChanWithBuiltInCJCSourceZ = baseAIThrmcplChanZ.extend({
  cjcSource: z.literal(BUILT_IN),
});
const aiThrmcplChanWithConstCJCSourceZ = baseAIThrmcplChanZ.extend({
  cjcSource: z.literal(CONST_VAL),
  cjcVal: z.number(),
});
const aiThrmcplChanWithChanCJCSourceZ = baseAIThrmcplChanZ.extend({
  cjcSource: z.literal(CHAN),
  cjcPort: z.number().int().nonnegative(),
});

export const aiThrmcplChanZ = z.union([
  aiThrmcplChanWithBuiltInCJCSourceZ,
  aiThrmcplChanWithConstCJCSourceZ,
  aiThrmcplChanWithChanCJCSourceZ,
]);
export type AIThrmcplChan = z.infer<typeof aiThrmcplChanZ>;
export const ZERO_AI_THRMCPL_CHAN: AIThrmcplChan = {
  ...ZERO_BASE_AI_CHAN,
  ...ZERO_MIN_MAX_VAL,
  type: AI_THRMCPL_CHAN_TYPE,
  units: DEG_C,
  thermocoupleType: J_TYPE_TC,
  cjcSource: BUILT_IN,
};

const torqueUnitsZ = z.enum([NEWTON_METERS, IN_OZ, INCH_LBS, FT_LBS]);
export type TorqueUnits = z.infer<typeof torqueUnitsZ>;

// https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaitorquebridgetablechan.html
export const AI_TORQUE_BRIDGE_TABLE_CHAN_TYPE = "ai_torque_bridge_table";
export const aiTorqueBridgeTableChanZ = baseAIChanZ.extend({
  ...minMaxValShape,
  ...bridgeShape,
  ...voltageExcitShape,
  ...tableShape,
  ...customScaleShape,
  type: z.literal(AI_TORQUE_BRIDGE_TABLE_CHAN_TYPE),
  units: torqueUnitsZ,
  physicalUnits: torqueUnitsZ,
});
export interface AITorqueBridgeTableChan
  extends z.infer<typeof aiTorqueBridgeTableChanZ> {}
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
export const aiTorqueBridgeTwoPointLinChanZ = baseAIChanZ.extend({
  ...minMaxValShape,
  ...bridgeShape,
  ...voltageExcitShape,
  ...twoPointLinShape,
  ...customScaleShape,
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
export const aiVelocityIEPEChanZ = baseAIChanZ.extend({
  ...terminalShape,
  ...minMaxValShape,
  ...sensitivityShape,
  ...currentExcitShape,
  ...customScaleShape,
  type: z.literal(AI_VELOCITY_IEPE_CHAN_TYPE),
  units: velocityUnitsZ,
  sensitivityUnits: velocitySensitivityUnitsZ,
});
export interface AIVelocityIEPEChan extends z.infer<typeof aiVelocityIEPEChanZ> {}
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
export const aiVoltageChanZ = baseAIChanZ.extend({
  ...terminalShape,
  ...minMaxValShape,
  ...customScaleShape,
  type: z.literal(AI_VOLTAGE_CHAN_TYPE),
  units: z.literal(VOLTS),
});
export interface AIVoltageChan extends z.infer<typeof aiVoltageChanZ> {}
export const ZERO_AI_VOLTAGE_CHAN: AIVoltageChan = {
  ...ZERO_BASE_AI_CHAN,
  ...ZERO_TERMINAL,
  ...ZERO_MIN_MAX_VAL,
  ...ZERO_CUSTOM_SCALE,
  type: AI_VOLTAGE_CHAN_TYPE,
  units: VOLTS,
};

const aiChannelZ = z.union([
  aiAccelChanZ,
  aiBridgeChanZ,
  aiCurrentChanZ,
  aiForceBridgeTableChanZ,
  aiForceBridgeTwoPointLinChanZ,
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

type AIChannel = z.infer<typeof aiChannelZ>;
export type AIChannelType = AIChannel["type"];

export const AI_CHANNEL_TYPE_NAMES: Record<AIChannelType, string> = {
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
  [AI_STRAIN_GAGE_CHAN_TYPE]: "Strain Gauge",
  [AI_TEMP_BUILT_IN_CHAN_TYPE]: "Temperature Built-In Sensor",
  [AI_THRMCPL_CHAN_TYPE]: "Thermocouple",
  [AI_TORQUE_BRIDGE_TABLE_CHAN_TYPE]: "Torque Bridge Table",
  [AI_TORQUE_BRIDGE_TWO_POINT_LIN_CHAN_TYPE]: "Torque Bridge Two-Point Linear",
  [AI_VELOCITY_IEPE_CHAN_TYPE]: "Velocity IEPE",
  [AI_VOLTAGE_CHAN_TYPE]: "Voltage",
};

export const AI_CHANNEL_TYPE_ICONS: Record<AIChannelType, Icon.FC> = {
  [AI_ACCEL_CHAN_TYPE]: Icon.Units.Acceleration,
  [AI_BRIDGE_CHAN_TYPE]: Icon.Bridge,
  [AI_CURRENT_CHAN_TYPE]: Icon.Units.Current,
  [AI_FORCE_BRIDGE_TABLE_CHAN_TYPE]: Icon.Units.Force,
  [AI_FORCE_BRIDGE_TWO_POINT_LIN_CHAN_TYPE]: Icon.Units.Force,
  [AI_FORCE_IEPE_CHAN_TYPE]: Icon.Units.Force,
  [AI_MICROPHONE_CHAN_TYPE]: Icon.Sound,
  [AI_PRESSURE_BRIDGE_TABLE_CHAN_TYPE]: Icon.Units.Pressure,
  [AI_PRESSURE_BRIDGE_TWO_POINT_LIN_CHAN_TYPE]: Icon.Units.Pressure,
  [AI_RESISTANCE_CHAN_TYPE]: Icon.Units.Resistance,
  [AI_RTD_CHAN_TYPE]: Icon.Units.Temperature,
  [AI_TEMP_BUILT_IN_CHAN_TYPE]: Icon.Units.Temperature,
  [AI_THRMCPL_CHAN_TYPE]: Icon.Units.Temperature,
  [AI_STRAIN_GAGE_CHAN_TYPE]: Icon.Units.Strain,
  [AI_TORQUE_BRIDGE_TABLE_CHAN_TYPE]: Icon.Units.Torque,
  [AI_TORQUE_BRIDGE_TWO_POINT_LIN_CHAN_TYPE]: Icon.Units.Torque,
  [AI_VELOCITY_IEPE_CHAN_TYPE]: Icon.Units.Velocity,
  [AI_VOLTAGE_CHAN_TYPE]: Icon.Units.Voltage,
};

// ==================== Counter Input Channels ====================

const counterChannelExtensionShape = { port: portZ };
interface CounterChannelExtension
  extends z.infer<z.ZodObject<typeof counterChannelExtensionShape>> {}
const ZERO_COUNTER_CHANNEL_EXTENSION: CounterChannelExtension = { port: 0 };

const baseCIChanZ = Common.Task.readChannelZ.extend(counterChannelExtensionShape);
interface BaseCIChan extends z.infer<typeof baseCIChanZ> {}
const ZERO_BASE_CI_CHAN: BaseCIChan = {
  ...Common.Task.ZERO_READ_CHANNEL,
  ...ZERO_COUNTER_CHANNEL_EXTENSION,
};

// Counter Input edge detection
const RISING_EDGE = "Rising";
const FALLING_EDGE = "Falling";
const ciEdgeZ = z.enum([RISING_EDGE, FALLING_EDGE]);
export type CIEdge = z.infer<typeof ciEdgeZ>;

// Counter Input measurement methods
const LOW_FREQ_1_CTR = "LowFreq1Ctr";
const HIGH_FREQ_2_CTR = "HighFreq2Ctr";
const LARGE_RNG_2_CTR = "LargeRng2Ctr";
const DYNAMIC_AVG = "DynamicAvg";
const ciMeasMethodZ = z.enum([
  LOW_FREQ_1_CTR,
  HIGH_FREQ_2_CTR,
  LARGE_RNG_2_CTR,
  DYNAMIC_AVG,
]);
export type CIMeasMethod = z.infer<typeof ciMeasMethodZ>;

// Counter Input frequency units
const TICKS = "Ticks";
const ciFreqUnitsZ = z.enum([HZ, TICKS]);
export type CIFreqUnits = z.infer<typeof ciFreqUnitsZ>;

// https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreatecifreqchan.html
export const CI_FREQUENCY_CHAN_TYPE = "ci_frequency";
export const ciFrequencyChanZ = baseCIChanZ.extend({
  ...minMaxValShape,
  ...customScaleShape,
  type: z.literal(CI_FREQUENCY_CHAN_TYPE),
  units: ciFreqUnitsZ,
  edge: ciEdgeZ,
  measMethod: ciMeasMethodZ,
  terminal: z.string(),
});
export interface CIFrequencyChan extends z.infer<typeof ciFrequencyChanZ> {}
export const ZERO_CI_FREQUENCY_CHAN: CIFrequencyChan = {
  ...ZERO_BASE_CI_CHAN,
  ...ZERO_MIN_MAX_VAL,
  ...ZERO_CUSTOM_SCALE,
  type: CI_FREQUENCY_CHAN_TYPE,
  minVal: 2,
  maxVal: 100,
  units: HZ,
  edge: RISING_EDGE,
  measMethod: DYNAMIC_AVG,
  terminal: "",
};

// Counter Input count direction
const COUNT_UP = "CountUp";
const COUNT_DOWN = "CountDown";
const EXTERNALLY_CONTROLLED = "ExternallyControlled";
const ciCountDirectionZ = z.enum([COUNT_UP, COUNT_DOWN, EXTERNALLY_CONTROLLED]);
export type CICountDirection = z.infer<typeof ciCountDirectionZ>;

// https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreatecicountedgeschan.html
export const CI_EDGE_COUNT_CHAN_TYPE = "ci_edge_count";
export const ciEdgeCountChanZ = baseCIChanZ.extend({
  type: z.literal(CI_EDGE_COUNT_CHAN_TYPE),
  activeEdge: ciEdgeZ,
  countDirection: ciCountDirectionZ,
  initialCount: z.number(),
  terminal: z.string(),
});
export interface CIEdgeCountChan extends z.infer<typeof ciEdgeCountChanZ> {}
export const ZERO_CI_EDGE_COUNT_CHAN: CIEdgeCountChan = {
  ...ZERO_BASE_CI_CHAN,
  type: CI_EDGE_COUNT_CHAN_TYPE,
  activeEdge: RISING_EDGE,
  countDirection: COUNT_UP,
  initialCount: 0,
  terminal: "",
};

// Counter Input period units
const ciPeriodUnitsZ = z.enum([SECONDS, TICKS, FROM_CUSTOM_SCALE]);
export type CIPeriodUnits = z.infer<typeof ciPeriodUnitsZ>;

// https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateciperiodchan.html
export const CI_PERIOD_CHAN_TYPE = "ci_period";
export const ciPeriodChanZ = baseCIChanZ.extend({
  ...minMaxValShape,
  ...customScaleShape,
  type: z.literal(CI_PERIOD_CHAN_TYPE),
  units: ciPeriodUnitsZ,
  startingEdge: ciEdgeZ,
  measMethod: ciMeasMethodZ,
  terminal: z.string(),
});
export interface CIPeriodChan extends z.infer<typeof ciPeriodChanZ> {}
export const ZERO_CI_PERIOD_CHAN: CIPeriodChan = {
  ...ZERO_BASE_CI_CHAN,
  ...ZERO_MIN_MAX_VAL,
  ...ZERO_CUSTOM_SCALE,
  type: CI_PERIOD_CHAN_TYPE,
  minVal: 0.000001,
  maxVal: 0.1,
  units: SECONDS,
  startingEdge: RISING_EDGE,
  measMethod: DYNAMIC_AVG,
  terminal: "",
};

// Counter Input pulse width units (same as period)
const ciPulseWidthUnitsZ = z.enum([SECONDS, TICKS, FROM_CUSTOM_SCALE]);
export type CIPulseWidthUnits = z.infer<typeof ciPulseWidthUnitsZ>;

// https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreatecipulsewidthchan.html
export const CI_PULSE_WIDTH_CHAN_TYPE = "ci_pulse_width";
export const ciPulseWidthChanZ = baseCIChanZ.extend({
  ...minMaxValShape,
  ...customScaleShape,
  type: z.literal(CI_PULSE_WIDTH_CHAN_TYPE),
  units: ciPulseWidthUnitsZ,
  startingEdge: ciEdgeZ,
  terminal: z.string(),
});
export interface CIPulseWidthChan extends z.infer<typeof ciPulseWidthChanZ> {}
export const ZERO_CI_PULSE_WIDTH_CHAN: CIPulseWidthChan = {
  ...ZERO_BASE_CI_CHAN,
  ...ZERO_MIN_MAX_VAL,
  ...ZERO_CUSTOM_SCALE,
  type: CI_PULSE_WIDTH_CHAN_TYPE,
  minVal: 0.000001,
  maxVal: 0.1,
  units: SECONDS,
  startingEdge: RISING_EDGE,
  terminal: "",
};

// Counter Input semi period units (same as period)
const ciSemiPeriodUnitsZ = z.enum([SECONDS, TICKS, FROM_CUSTOM_SCALE]);
export type CISemiPeriodUnits = z.infer<typeof ciSemiPeriodUnitsZ>;

// https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreatecisemiperiodchan.html
export const CI_SEMI_PERIOD_CHAN_TYPE = "ci_semi_period";
export const ciSemiPeriodChanZ = baseCIChanZ.extend({
  ...minMaxValShape,
  ...customScaleShape,
  type: z.literal(CI_SEMI_PERIOD_CHAN_TYPE),
  units: ciSemiPeriodUnitsZ,
});
export interface CISemiPeriodChan extends z.infer<typeof ciSemiPeriodChanZ> {}
export const ZERO_CI_SEMI_PERIOD_CHAN: CISemiPeriodChan = {
  ...ZERO_BASE_CI_CHAN,
  ...ZERO_MIN_MAX_VAL,
  ...ZERO_CUSTOM_SCALE,
  type: CI_SEMI_PERIOD_CHAN_TYPE,
  minVal: 0.000001,
  maxVal: 0.1,
  units: SECONDS,
};

// Counter Input two edge separation units
const ciTwoEdgeSepUnitsZ = z.enum([SECONDS, TICKS]);
export type CITwoEdgeSepUnits = z.infer<typeof ciTwoEdgeSepUnitsZ>;

// https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreatecitwoedgesepchan.html
export const CI_TWO_EDGE_SEP_CHAN_TYPE = "ci_two_edge_sep";
export const ciTwoEdgeSepChanZ = baseCIChanZ.extend({
  ...minMaxValShape,
  ...customScaleShape,
  type: z.literal(CI_TWO_EDGE_SEP_CHAN_TYPE),
  units: ciTwoEdgeSepUnitsZ,
  firstEdge: ciEdgeZ,
  secondEdge: ciEdgeZ,
});
export interface CITwoEdgeSepChan extends z.infer<typeof ciTwoEdgeSepChanZ> {}
export const ZERO_CI_TWO_EDGE_SEP_CHAN: CITwoEdgeSepChan = {
  ...ZERO_BASE_CI_CHAN,
  ...ZERO_MIN_MAX_VAL,
  ...ZERO_CUSTOM_SCALE,
  type: CI_TWO_EDGE_SEP_CHAN_TYPE,
  minVal: 0.000001,
  maxVal: 1,
  units: SECONDS,
  firstEdge: RISING_EDGE,
  secondEdge: FALLING_EDGE,
};

const ciChannelZ = z.union([
  ciFrequencyChanZ,
  ciEdgeCountChanZ,
  ciPeriodChanZ,
  ciPulseWidthChanZ,
  ciSemiPeriodChanZ,
  ciTwoEdgeSepChanZ,
]);

type CIChannel = z.infer<typeof ciChannelZ>;
export type CIChannelType = CIChannel["type"];

export const CI_CHANNEL_TYPE_NAMES: Record<CIChannelType, string> = {
  [CI_FREQUENCY_CHAN_TYPE]: "Frequency",
  [CI_EDGE_COUNT_CHAN_TYPE]: "Edge Count",
  [CI_PERIOD_CHAN_TYPE]: "Period",
  [CI_PULSE_WIDTH_CHAN_TYPE]: "Pulse Width",
  [CI_SEMI_PERIOD_CHAN_TYPE]: "Semi Period",
  [CI_TWO_EDGE_SEP_CHAN_TYPE]: "Two Edge Separation",
};

export const CI_CHANNEL_TYPE_ICONS: Record<CIChannelType, Icon.FC> = {
  [CI_FREQUENCY_CHAN_TYPE]: Icon.Wave.Square,
  [CI_EDGE_COUNT_CHAN_TYPE]: Icon.Value,
  [CI_PERIOD_CHAN_TYPE]: Icon.Time,
  [CI_PULSE_WIDTH_CHAN_TYPE]: Icon.AutoFitWidth,
  [CI_SEMI_PERIOD_CHAN_TYPE]: Icon.Range,
  [CI_TWO_EDGE_SEP_CHAN_TYPE]: Icon.AutoFitWidth,
};

// Counter Output Channels
const baseCOChanZ = Common.Task.writeChannelZ.extend(counterChannelExtensionShape);
interface BaseCOChan extends z.infer<typeof baseCOChanZ> {}
const ZERO_BASE_CO_CHAN: BaseCOChan = {
  ...Common.Task.ZERO_WRITE_CHANNEL,
  ...ZERO_COUNTER_CHANNEL_EXTENSION,
};

// Counter Output idle state
const IDLE_HIGH = "High";
const IDLE_LOW = "Low";
const coIdleStateZ = z.enum([IDLE_HIGH, IDLE_LOW]);
export type COIdleState = z.infer<typeof coIdleStateZ>;

// https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreatecopulsechantime.html
export const CO_PULSE_OUTPUT_CHAN_TYPE = "co_pulse_output";
export const coPulseOutputChanZ = baseCOChanZ.extend({
  ...minMaxValShape,
  type: z.literal(CO_PULSE_OUTPUT_CHAN_TYPE),
  units: z.literal(SECONDS),
  idleState: coIdleStateZ,
  initialDelay: z.number(),
  highTime: z.number(),
  lowTime: z.number(),
});
export interface COPulseOutputChan extends z.infer<typeof coPulseOutputChanZ> {}
export const ZERO_CO_PULSE_OUTPUT_CHAN: COPulseOutputChan = {
  ...ZERO_BASE_CO_CHAN,
  ...ZERO_MIN_MAX_VAL,
  type: CO_PULSE_OUTPUT_CHAN_TYPE,
  units: SECONDS,
  idleState: IDLE_LOW,
  initialDelay: 0,
  highTime: 0.01,
  lowTime: 0.01,
};

const coChannelZ = z.union([coPulseOutputChanZ]);
export type COChannel = z.infer<typeof coChannelZ>;
export type COChannelType = COChannel["type"];

export const CO_CHANNEL_SCHEMAS: Record<COChannelType, z.ZodType<COChannel>> = {
  [CO_PULSE_OUTPUT_CHAN_TYPE]: coPulseOutputChanZ,
};

export const CO_CHANNEL_TYPES = [CO_PULSE_OUTPUT_CHAN_TYPE] as const;

export const CO_CHANNEL_TYPE_NAMES: Record<COChannelType, string> = {
  [CO_PULSE_OUTPUT_CHAN_TYPE]: "Pulse Output",
};

export const CO_CHANNEL_TYPE_ICONS: Record<COChannelType, Icon.FC> = {
  [CO_PULSE_OUTPUT_CHAN_TYPE]: Icon.Wave.Square,
};

export const ZERO_CO_CHANNELS: Record<COChannelType, COChannel> = {
  [CO_PULSE_OUTPUT_CHAN_TYPE]: ZERO_CO_PULSE_OUTPUT_CHAN,
};
export const ZERO_CO_CHANNEL = ZERO_CO_CHANNELS[CO_PULSE_OUTPUT_CHAN_TYPE];

const baseAOChanZ = Common.Task.writeChannelZ.extend(analogChannelExtensionShape);
interface BaseAOChan extends z.infer<typeof baseAOChanZ> {}
const ZERO_BASE_AO_CHAN: BaseAOChan = {
  ...Common.Task.ZERO_WRITE_CHANNEL,
  ...ZERO_ANALOG_CHANNEL_EXTENSION,
};

export const AO_CURRENT_CHAN_TYPE = "ao_current";
const aoCurrentChanZ = baseAOChanZ.extend({
  ...minMaxValShape,
  ...customScaleShape,
  type: z.literal(AO_CURRENT_CHAN_TYPE),
  units: z.literal(AMPS),
});
interface AOCurrentChan extends z.infer<typeof aoCurrentChanZ> {}
const ZERO_AO_CURRENT_CHAN: AOCurrentChan = {
  ...ZERO_BASE_AO_CHAN,
  ...ZERO_MIN_MAX_VAL,
  ...ZERO_CUSTOM_SCALE,
  type: AO_CURRENT_CHAN_TYPE,
  units: AMPS,
};

export const AO_FUNC_GEN_CHAN_TYPE = "ao_func_gen";
export const SINE_WAVE_TYPE = "Sine";
export const TRIANGLE_WAVE_TYPE = "Triangle";
export const SQUARE_WAVE_TYPE = "Square";
export const SAWTOOTH_WAVE_TYPE = "Sawtooth";
export const WAVE_TYPES = [
  SINE_WAVE_TYPE,
  TRIANGLE_WAVE_TYPE,
  SQUARE_WAVE_TYPE,
  SAWTOOTH_WAVE_TYPE,
] as const;
export type WaveType = (typeof WAVE_TYPES)[number];

const aoFuncGenChanZ = baseAOChanZ.extend({
  type: z.literal(AO_FUNC_GEN_CHAN_TYPE),
  // note that waveType is called type in DAQmx, but this conflicts with our convention
  waveType: z.enum(WAVE_TYPES),
  frequency: z.number(),
  amplitude: z.number(),
  offset: z.number(),
});
interface AOFuncGenChan extends z.infer<typeof aoFuncGenChanZ> {}
const ZERO_AO_FUNC_GEN_CHAN: AOFuncGenChan = {
  ...ZERO_BASE_AO_CHAN,
  type: AO_FUNC_GEN_CHAN_TYPE,
  waveType: SINE_WAVE_TYPE,
  frequency: 0,
  amplitude: 0,
  offset: 0,
};

export const AO_VOLTAGE_CHAN_TYPE = "ao_voltage";
const aoVoltageChanZ = baseAOChanZ.extend({
  ...minMaxValShape,
  ...customScaleShape,
  type: z.literal(AO_VOLTAGE_CHAN_TYPE),
  units: z.literal(VOLTS),
});
interface AOVoltageChan extends z.infer<typeof aoVoltageChanZ> {}
const ZERO_AO_VOLTAGE_CHAN: AOVoltageChan = {
  ...ZERO_BASE_AO_CHAN,
  ...ZERO_MIN_MAX_VAL,
  ...ZERO_CUSTOM_SCALE,
  type: AO_VOLTAGE_CHAN_TYPE,
  units: VOLTS,
};

const aoChannelZ = z.union([aoCurrentChanZ, aoFuncGenChanZ, aoVoltageChanZ]);
export type AOChannel = z.infer<typeof aoChannelZ>;
export type AOChannelType = AOChannel["type"];

export const AO_CHANNEL_SCHEMAS: Record<AOChannelType, z.ZodType<AOChannel>> = {
  [AO_CURRENT_CHAN_TYPE]: aoCurrentChanZ,
  [AO_FUNC_GEN_CHAN_TYPE]: aoFuncGenChanZ,
  [AO_VOLTAGE_CHAN_TYPE]: aoVoltageChanZ,
};

export const AO_CHANNEL_TYPES = [
  AO_CURRENT_CHAN_TYPE,
  AO_FUNC_GEN_CHAN_TYPE,
  AO_VOLTAGE_CHAN_TYPE,
] as const;

export const AO_CHANNEL_TYPE_NAMES: Record<AOChannelType, string> = {
  [AO_CURRENT_CHAN_TYPE]: "Current",
  [AO_FUNC_GEN_CHAN_TYPE]: "Function Generator",
  [AO_VOLTAGE_CHAN_TYPE]: "Voltage",
};

export const AO_CHANNEL_TYPE_ICONS: Record<AOChannelType, Icon.FC> = {
  [AO_CURRENT_CHAN_TYPE]: Icon.Units.Current,
  [AO_FUNC_GEN_CHAN_TYPE]: Icon.Function,
  [AO_VOLTAGE_CHAN_TYPE]: Icon.Units.Voltage,
};

export const ZERO_AO_CHANNELS: Record<AOChannelType, AOChannel> = {
  [AO_CURRENT_CHAN_TYPE]: ZERO_AO_CURRENT_CHAN,
  [AO_FUNC_GEN_CHAN_TYPE]: ZERO_AO_FUNC_GEN_CHAN,
  [AO_VOLTAGE_CHAN_TYPE]: ZERO_AO_VOLTAGE_CHAN,
};
export const ZERO_AO_CHANNEL = ZERO_AO_CHANNELS[AO_VOLTAGE_CHAN_TYPE];

const DIGITAL_INPUT_TYPE = "digital_input";
const DIGITAL_OUTPUT_TYPE = "digital_output";

const diChannelZ = Common.Task.readChannelZ
  .extend(digitalChannelExtensionShape)
  .extend({
    type: z.literal(DIGITAL_INPUT_TYPE),
  });
export interface DIChannel extends z.infer<typeof diChannelZ> {}
export const ZERO_DI_CHANNEL: DIChannel = {
  ...Common.Task.ZERO_READ_CHANNEL,
  ...ZERO_DIGITAL_CHANNEL_EXTENSION,
  type: DIGITAL_INPUT_TYPE,
};

const doChannelZ = Common.Task.writeChannelZ
  .extend(digitalChannelExtensionShape)
  .extend({
    type: z.literal(DIGITAL_OUTPUT_TYPE),
  });
export interface DOChannel extends z.infer<typeof doChannelZ> {}
export const ZERO_DO_CHANNEL: DOChannel = {
  ...Common.Task.ZERO_WRITE_CHANNEL,
  ...ZERO_DIGITAL_CHANNEL_EXTENSION,
  type: DIGITAL_OUTPUT_TYPE,
};

export type DigitalChannel = DIChannel | DOChannel;

const baseReadConfigZ = Common.Task.baseReadConfigZ.extend({
  sampleRate: z.number().positive().max(100000),
  streamRate: z.number().positive().max(20000),
});
interface BaseReadConfig extends z.infer<typeof baseReadConfigZ> {}
const ZERO_BASE_READ_CONFIG: BaseReadConfig = {
  ...Common.Task.ZERO_BASE_READ_CONFIG,
  sampleRate: 10,
  streamRate: 5,
};

const baseWriteConfigZ = Common.Task.baseConfigZ.extend({
  dataSaving: z.boolean().default(true),
  stateRate: z.number().positive().max(50000),
});
interface BaseWriteConfig extends z.infer<typeof baseWriteConfigZ> {}
const ZERO_BASE_WRITE_CONFIG: BaseWriteConfig = {
  ...Common.Task.ZERO_BASE_CONFIG,
  dataSaving: true,
  stateRate: 10,
};

const validateAnalogPorts = ({
  value: channels,
  issues,
}: z.core.ParsePayload<{ port: number }[]>) => {
  const portsToIndexMap = new Map<number, number>();
  channels.forEach(({ port }, i) => {
    if (!portsToIndexMap.has(port)) {
      portsToIndexMap.set(port, i);
      return;
    }
    const index = portsToIndexMap.get(port) as number;
    const code = "custom";
    const message = `Port ${port} has already been used on another channel`;
    issues.push({ code, message, path: [index, "port"], input: channels[index] });
    issues.push({ code, message, path: [i, "port"], input: channels[i] });
  });
};

const validateDigitalPortsAndLines = ({
  value: channels,
  issues,
}: z.core.ParsePayload<DigitalChannel[]>) => {
  const portLineToIndexMap = new Map<string, number>();
  channels.forEach(({ line, port }, i) => {
    const key = `${port}/${line}`;
    if (!portLineToIndexMap.has(key)) {
      portLineToIndexMap.set(key, i);
      return;
    }
    const index = portLineToIndexMap.get(key) as number;
    const code = "custom";
    const message = `Port ${port}, line ${line} has already been used on another channel`;
    issues.push({ code, message, path: [index, "line"], input: channels[index] });
    issues.push({ code, message, path: [i, "line"], input: channels[i] });
  });
};

export const baseAnalogReadConfigZ = baseReadConfigZ.extend({
  channels: z
    .array(aiChannelZ)
    .check(Common.Task.validateReadChannels)
    .check(validateAnalogPorts),
});

export const analogReadConfigZ = baseAnalogReadConfigZ.check(
  Common.Task.validateStreamRate,
);
export interface AnalogReadConfig extends z.infer<typeof analogReadConfigZ> {}
export const ZERO_ANALOG_READ_CONFIG: AnalogReadConfig = {
  ...ZERO_BASE_READ_CONFIG,
  channels: [],
};

export const analogReadStatusDataZ = z
  .object({
    errors: z.array(z.object({ message: z.string(), path: z.string() })),
  })
  .or(z.null());

export type AnalogReadStatusDetails = task.Status<typeof analogReadStatusDataZ>;

export const ANALOG_READ_TYPE = `${PREFIX}_analog_read`;
export const analogReadTypeZ = z.literal(ANALOG_READ_TYPE);
export type AnalogReadType = z.infer<typeof analogReadTypeZ>;

interface AnalogReadPayload
  extends task.Payload<
    typeof analogReadTypeZ,
    typeof analogReadConfigZ,
    typeof analogReadStatusDataZ
  > {}
export const ZERO_ANALOG_READ_PAYLOAD: AnalogReadPayload = {
  key: "",
  name: "NI Analog Read Task",
  config: ZERO_ANALOG_READ_CONFIG,
  type: ANALOG_READ_TYPE,
};

// ==================== Counter Read Task ====================

const validateCounterPorts = ({
  value: channels,
  issues,
}: z.core.ParsePayload<CIChannel[]>) => {
  const portToIndexMap = new Map<number, number>();
  channels.forEach(({ port }, i) => {
    if (!portToIndexMap.has(port)) {
      portToIndexMap.set(port, i);
      return;
    }
    const index = portToIndexMap.get(port) as number;
    const code = "custom";
    const message = `Port ${port} has already been used on another channel`;
    issues.push({ path: [index, "port"], code, message, input: channels });
    issues.push({ path: [i, "port"], code, message, input: channels });
  });
};

export const counterReadConfigZ = baseReadConfigZ
  .extend({
    channels: z
      .array(ciChannelZ)
      .check(Common.Task.validateReadChannels)
      .check(validateCounterPorts),
  })
  .check(Common.Task.validateStreamRate);
export interface CounterReadConfig extends z.infer<typeof counterReadConfigZ> {}
export const ZERO_COUNTER_READ_CONFIG: CounterReadConfig = {
  ...ZERO_BASE_READ_CONFIG,
  channels: [],
};

export const counterReadStatusDataZ = z.unknown();
export type CounterReadStatusDetails = task.Status<typeof counterReadStatusDataZ>;

export const COUNTER_READ_TYPE = `${PREFIX}_counter_read`;
export const counterReadTypeZ = z.literal(COUNTER_READ_TYPE);
export type CounterReadType = z.infer<typeof counterReadTypeZ>;

interface CounterReadPayload
  extends task.Payload<
    typeof counterReadTypeZ,
    typeof counterReadConfigZ,
    typeof counterReadStatusDataZ
  > {}
export const ZERO_COUNTER_READ_PAYLOAD: CounterReadPayload = {
  key: "",
  name: "NI Counter Read Task",
  config: ZERO_COUNTER_READ_CONFIG,
  type: COUNTER_READ_TYPE,
};

export const analogWriteConfigZ = baseWriteConfigZ.extend({
  channels: z
    .array(aoChannelZ)
    .check(Common.Task.validateWriteChannels)
    .check(validateAnalogPorts),
});
export interface AnalogWriteConfig extends z.infer<typeof analogWriteConfigZ> {}
const ZERO_ANALOG_WRITE_CONFIG: AnalogWriteConfig = {
  ...ZERO_BASE_WRITE_CONFIG,
  channels: [],
};

export const analogWriteStatusDataZ = z.unknown();

export const ANALOG_WRITE_TYPE = `${PREFIX}_analog_write`;
export const analogWriteTypeZ = z.literal(ANALOG_WRITE_TYPE);
export type AnalogWriteType = z.infer<typeof analogWriteTypeZ>;

export interface AnalogWritePayload
  extends task.Payload<
    typeof analogWriteTypeZ,
    typeof analogWriteConfigZ,
    typeof analogWriteStatusDataZ
  > {}
export const ZERO_ANALOG_WRITE_PAYLOAD: AnalogWritePayload = {
  key: "",
  name: "NI Analog Write Task",
  config: ZERO_ANALOG_WRITE_CONFIG,
  type: ANALOG_WRITE_TYPE,
};

export interface AnalogWriteTask
  extends task.Task<
    typeof analogWriteTypeZ,
    typeof analogWriteConfigZ,
    typeof analogWriteStatusDataZ
  > {}
export interface NewAnalogWriteTask
  extends task.New<typeof analogWriteTypeZ, typeof analogWriteConfigZ> {}

export const digitalReadConfigZ = baseReadConfigZ
  .extend({
    channels: z
      .array(diChannelZ)
      .check(Common.Task.validateReadChannels)
      .check(validateDigitalPortsAndLines),
  })
  .check(Common.Task.validateStreamRate);
export interface DigitalReadConfig extends z.infer<typeof digitalReadConfigZ> {}
const ZERO_DIGITAL_READ_CONFIG: DigitalReadConfig = {
  ...ZERO_BASE_READ_CONFIG,
  channels: [],
};

export const digitalReadStatusDataZ = z.unknown();
export type DigitalReadStatusDetails = task.Status<typeof digitalReadStatusDataZ>;

export const DIGITAL_READ_TYPE = `${PREFIX}_digital_read`;
export const digitalReadTypeZ = z.literal(DIGITAL_READ_TYPE);
export type DigitalReadType = z.infer<typeof digitalReadTypeZ>;

export interface DigitalReadPayload
  extends task.Payload<
    typeof digitalReadTypeZ,
    typeof digitalReadConfigZ,
    typeof digitalReadStatusDataZ
  > {}
export const ZERO_DIGITAL_READ_PAYLOAD: DigitalReadPayload = {
  key: "",
  name: "NI Digital Read Task",
  config: ZERO_DIGITAL_READ_CONFIG,
  type: DIGITAL_READ_TYPE,
};

export interface DigitalReadTask
  extends task.Task<
    typeof digitalReadTypeZ,
    typeof digitalReadConfigZ,
    typeof digitalReadStatusDataZ
  > {}
export interface NewDigitalReadTask
  extends task.New<typeof digitalReadTypeZ, typeof digitalReadConfigZ> {}

export const digitalWriteConfigZ = baseWriteConfigZ.extend({
  channels: z
    .array(doChannelZ)
    .check(Common.Task.validateWriteChannels)
    .check(validateDigitalPortsAndLines),
});
export interface DigitalWriteConfig extends z.infer<typeof digitalWriteConfigZ> {}
const ZERO_DIGITAL_WRITE_CONFIG: DigitalWriteConfig = {
  ...ZERO_BASE_WRITE_CONFIG,
  channels: [],
};

export const digitalWriteStatusDataZ = z.unknown();
export type DigitalWriteStatusDetails = task.Status<typeof digitalWriteStatusDataZ>;

export const DIGITAL_WRITE_TYPE = `${PREFIX}_digital_write`;
export const digitalWriteTypeZ = z.literal(DIGITAL_WRITE_TYPE);
export type DigitalWriteType = z.infer<typeof digitalWriteTypeZ>;

export interface DigitalWritePayload
  extends task.Payload<
    typeof digitalWriteTypeZ,
    typeof digitalWriteConfigZ,
    typeof digitalWriteStatusDataZ
  > {}
export const ZERO_DIGITAL_WRITE_PAYLOAD: DigitalWritePayload = {
  key: "",
  name: "NI Digital Write Task",
  config: ZERO_DIGITAL_WRITE_CONFIG,
  type: DIGITAL_WRITE_TYPE,
};

export interface DigitalWriteTask
  extends task.Task<
    typeof digitalWriteTypeZ,
    typeof digitalWriteConfigZ,
    typeof digitalWriteStatusDataZ
  > {}
export interface NewDigitalWriteTask
  extends task.New<typeof digitalWriteTypeZ, typeof digitalWriteConfigZ> {}

const validateCounterWritePorts = ({
  value: channels,
  issues,
}: z.core.ParsePayload<COChannel[]>) => {
  const portToIndexMap = new Map<number, number>();
  channels.forEach(({ port }, i) => {
    if (!portToIndexMap.has(port)) {
      portToIndexMap.set(port, i);
      return;
    }
    const index = portToIndexMap.get(port) as number;
    const code = "custom";
    const message = `Port ${port} has already been used on another channel`;
    issues.push({ path: [index, "port"], code, message, input: channels });
    issues.push({ path: [i, "port"], code, message, input: channels });
  });
};

export const counterWriteConfigZ = baseWriteConfigZ.extend({
  channels: z
    .array(coChannelZ)
    .check(Common.Task.validateWriteChannels)
    .check(validateCounterWritePorts),
});
export interface CounterWriteConfig extends z.infer<typeof counterWriteConfigZ> {}
const ZERO_COUNTER_WRITE_CONFIG: CounterWriteConfig = {
  ...ZERO_BASE_WRITE_CONFIG,
  channels: [],
};

export const counterWriteStatusDataZ = z.unknown();
export type CounterWriteStatusDetails = task.Status<typeof counterWriteStatusDataZ>;

export const COUNTER_WRITE_TYPE = `${PREFIX}_counter_write`;
export const counterWriteTypeZ = z.literal(COUNTER_WRITE_TYPE);
export type CounterWriteType = z.infer<typeof counterWriteTypeZ>;

export interface CounterWritePayload
  extends task.Payload<
    typeof counterWriteTypeZ,
    typeof counterWriteConfigZ,
    typeof counterWriteStatusDataZ
  > {}
export const ZERO_COUNTER_WRITE_PAYLOAD: CounterWritePayload = {
  key: "",
  name: "NI Counter Write Task",
  config: ZERO_COUNTER_WRITE_CONFIG,
  type: COUNTER_WRITE_TYPE,
};

export interface CounterWriteTask
  extends task.Task<
    typeof counterWriteTypeZ,
    typeof counterWriteConfigZ,
    typeof counterWriteStatusDataZ
  > {}
export interface NewCounterWriteTask
  extends task.New<typeof counterWriteTypeZ, typeof counterWriteConfigZ> {}

export const scanStatusDataZ = z.unknown();
export type ScanStatusDetails = task.Status<typeof scanStatusDataZ>;

export const SCAN_TYPE = `${PREFIX}_scanner`;
export const scanTypeZ = z.literal(SCAN_TYPE);
export type ScanType = z.infer<typeof scanTypeZ>;

export const scanConfigZ = z.object({
  enabled: z.boolean(),
});
export interface ScanConfig extends z.infer<typeof scanConfigZ> {}

export interface ScanPayload
  extends task.Payload<typeof scanTypeZ, typeof scanConfigZ, typeof scanStatusDataZ> {}

export interface ScanTask
  extends task.Task<typeof scanTypeZ, typeof scanConfigZ, typeof scanStatusDataZ> {}
export interface NewScanTask extends task.New<typeof scanTypeZ, typeof scanConfigZ> {}
