// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type task } from "@synnaxlabs/client";
import { z } from "zod";

import { Common } from "@/hardware/common";

export const PREFIX = "ni";

const portZ = z.number().int().nonnegative();
const lineZ = z.number().int().nonnegative();

const analogChannelExtensionShape = { port: portZ };
interface AnalogChannelExtension
  extends z.infer<z.ZodObject<typeof analogChannelExtensionShape>> {}
const ZERO_ANALOG_CHANNEL_EXTENSION: AnalogChannelExtension = { port: 0 };

const digitalChannelExtensionShape = { port: portZ, line: lineZ };
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
  slope: z.number().finite(),
  yIntercept: z.number().finite(),
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
    preScaledMin: z.number().finite(),
    preScaledMax: z.number().finite(),
    scaledMin: z.number().finite(),
    scaledMax: z.number().finite(),
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
    preScaledVals: z.number().finite().array(),
    scaledVals: z.number().finite().array(),
    preScaledUnits: unitsZ,
    scaledUnits: z.string(),
  })
  .superRefine(({ preScaledVals, scaledVals }, { addIssue }) => {
    if (preScaledVals.length !== scaledVals.length) {
      const baseIssue = {
        code: z.ZodIssueCode.custom,
        message: "Pre-scaled and scaled values must have the same length",
      };
      addIssue({ ...baseIssue, path: ["preScaledVals"] });
      addIssue({ ...baseIssue, path: ["scaledVals"] });
    }
  })
  .superRefine(({ preScaledVals }, { addIssue }) => {
    if (preScaledVals.length === 0) return;
    let lastVal = preScaledVals[0];
    for (let i = 1; i < preScaledVals.length; i++) {
      if (preScaledVals[i] <= lastVal)
        addIssue({
          code: z.ZodIssueCode.custom,
          message: "Pre-scaled values must be monotonically increasing",
          path: ["preScaledVals"],
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
  cjcVal: z.number().finite(),
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
  [AI_STRAIN_GAGE_CHAN_TYPE]: "Strain Gage",
  [AI_TEMP_BUILT_IN_CHAN_TYPE]: "Temperature Built-In Sensor",
  [AI_THRMCPL_CHAN_TYPE]: "Thermocouple",
  [AI_TORQUE_BRIDGE_TABLE_CHAN_TYPE]: "Torque Bridge Table",
  [AI_TORQUE_BRIDGE_TWO_POINT_LIN_CHAN_TYPE]: "Torque Bridge Two-Point Linear",
  [AI_VELOCITY_IEPE_CHAN_TYPE]: "Velocity IEPE",
  [AI_VOLTAGE_CHAN_TYPE]: "Voltage",
};

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
export type WaveType =
  | typeof SINE_WAVE_TYPE
  | typeof TRIANGLE_WAVE_TYPE
  | typeof SQUARE_WAVE_TYPE
  | typeof SAWTOOTH_WAVE_TYPE;

const aoFuncGenChanZ = baseAOChanZ.extend({
  type: z.literal(AO_FUNC_GEN_CHAN_TYPE),
  // note that waveType is called type in DAQmx, but this conflicts with our convention
  waveType: z.enum([
    SINE_WAVE_TYPE,
    SQUARE_WAVE_TYPE,
    TRIANGLE_WAVE_TYPE,
    SAWTOOTH_WAVE_TYPE,
  ]),
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

export const AO_CHANNEL_TYPE_NAMES: Record<AOChannelType, string> = {
  [AO_CURRENT_CHAN_TYPE]: "Current",
  [AO_FUNC_GEN_CHAN_TYPE]: "Function Generator",
  [AO_VOLTAGE_CHAN_TYPE]: "Voltage",
};

export const ZERO_AO_CHANNELS: Record<AOChannelType, AOChannel> = {
  [AO_CURRENT_CHAN_TYPE]: ZERO_AO_CURRENT_CHAN,
  [AO_FUNC_GEN_CHAN_TYPE]: ZERO_AO_FUNC_GEN_CHAN,
  [AO_VOLTAGE_CHAN_TYPE]: ZERO_AO_VOLTAGE_CHAN,
};
export const ZERO_AO_CHANNEL = ZERO_AO_CHANNELS[AO_VOLTAGE_CHAN_TYPE];

const diChannelZ = Common.Task.readChannelZ.extend(digitalChannelExtensionShape);
export interface DIChannel extends z.infer<typeof diChannelZ> {}
export const ZERO_DI_CHANNEL: DIChannel = {
  ...Common.Task.ZERO_READ_CHANNEL,
  ...ZERO_DIGITAL_CHANNEL_EXTENSION,
};

const doChannelZ = Common.Task.writeChannelZ.extend(digitalChannelExtensionShape);
export interface DOChannel extends z.infer<typeof doChannelZ> {}
export const ZERO_DO_CHANNEL: DOChannel = {
  ...Common.Task.ZERO_WRITE_CHANNEL,
  ...ZERO_DIGITAL_CHANNEL_EXTENSION,
};

export type DigitalChannel = DIChannel | DOChannel;

const baseReadConfigZ = Common.Task.baseConfigZ.extend({
  sampleRate: z.number().positive().max(50000),
  streamRate: z.number().positive().max(50000),
});
interface BaseReadConfig extends z.infer<typeof baseReadConfigZ> {}
const ZERO_BASE_READ_CONFIG: BaseReadConfig = {
  ...Common.Task.ZERO_BASE_CONFIG,
  sampleRate: 10,
  streamRate: 5,
};

const baseWriteConfigZ = Common.Task.baseConfigZ.extend({
  stateRate: z.number().positive().max(50000),
});
interface BaseWriteConfig extends z.infer<typeof baseWriteConfigZ> {}
const ZERO_BASE_WRITE_CONFIG: BaseWriteConfig = {
  ...Common.Task.ZERO_BASE_CONFIG,
  stateRate: 10,
};

export const validateAnalogPorts = (
  channels: { port: number }[],
  { addIssue }: z.RefinementCtx,
) => {
  const portsToIndexMap = new Map<number, number>();
  channels.forEach(({ port }, i) => {
    if (!portsToIndexMap.has(port)) {
      portsToIndexMap.set(port, i);
      return;
    }
    const index = portsToIndexMap.get(port) as number;
    const baseIssue = {
      code: z.ZodIssueCode.custom,
      message: `Port ${port} has already been used on another channel`,
    };
    addIssue({ ...baseIssue, path: [index, "port"] });
    addIssue({ ...baseIssue, path: [i, "port"] });
  });
};

const validateDigitalPortsAndLines = (
  channels: DigitalChannel[],
  { addIssue }: z.RefinementCtx,
) => {
  const portLineToIndexMap = new Map<string, number>();
  channels.forEach(({ line, port }, i) => {
    const key = `${port}/${line}`;
    if (!portLineToIndexMap.has(key)) {
      portLineToIndexMap.set(key, i);
      return;
    }
    const index = portLineToIndexMap.get(key) as number;
    const baseIssue = {
      code: z.ZodIssueCode.custom,
      message: `Port ${port}, line ${line} has already been used on another channel`,
    };
    addIssue({ ...baseIssue, path: [index, "line"] });
    addIssue({ ...baseIssue, path: [i, "line"] });
  });
};

export interface BaseStateDetails {
  running: boolean;
}

export const baseAnalogReadConfigZ = baseReadConfigZ.extend({
  channels: z
    .array(aiChannelZ)
    .superRefine(Common.Task.validateReadChannels)
    .superRefine(validateAnalogPorts),
});

export const analogReadConfigZ = baseAnalogReadConfigZ.superRefine(
  Common.Task.validateStreamRate,
);
export interface AnalogReadConfig extends z.infer<typeof analogReadConfigZ> {}
export const ZERO_ANALOG_READ_CONFIG: AnalogReadConfig = {
  ...ZERO_BASE_READ_CONFIG,
  channels: [],
};

export interface AnalogReadStateDetails extends BaseStateDetails {
  message: string;
  errors?: { message: string; path: string }[];
}
export interface AnalogReadState extends task.State<AnalogReadStateDetails> {}

export const ANALOG_READ_TYPE = `${PREFIX}_analog_read`;
export type AnalogReadType = typeof ANALOG_READ_TYPE;

interface AnalogReadPayload
  extends task.Payload<AnalogReadConfig, AnalogReadStateDetails, AnalogReadType> {}
export const ZERO_ANALOG_READ_PAYLOAD: AnalogReadPayload = {
  key: "",
  name: "NI Analog Read Task",
  config: ZERO_ANALOG_READ_CONFIG,
  type: ANALOG_READ_TYPE,
};

export const analogWriteConfigZ = baseWriteConfigZ.extend({
  channels: z
    .array(aoChannelZ)
    .superRefine(Common.Task.validateWriteChannels)
    .superRefine(validateAnalogPorts),
});
export interface AnalogWriteConfig extends z.infer<typeof analogWriteConfigZ> {}
const ZERO_ANALOG_WRITE_CONFIG: AnalogWriteConfig = {
  ...ZERO_BASE_WRITE_CONFIG,
  channels: [],
};

export interface AnalogWriteStateDetails extends BaseStateDetails {}
export interface AnalogWriteState extends task.State<AnalogWriteStateDetails> {}

export const ANALOG_WRITE_TYPE = `${PREFIX}_analog_write`;
export type AnalogWriteType = typeof ANALOG_WRITE_TYPE;

export interface AnalogWritePayload
  extends task.Payload<AnalogWriteConfig, AnalogWriteStateDetails, AnalogWriteType> {}
export const ZERO_ANALOG_WRITE_PAYLOAD: AnalogWritePayload = {
  key: "",
  name: "NI Analog Write Task",
  config: ZERO_ANALOG_WRITE_CONFIG,
  type: ANALOG_WRITE_TYPE,
};

export interface AnalogWriteTask
  extends task.Task<AnalogWriteConfig, AnalogWriteStateDetails, AnalogWriteType> {}
export interface NewAnalogWriteTask
  extends task.New<AnalogWriteConfig, AnalogWriteType> {}

export const digitalReadConfigZ = baseReadConfigZ
  .extend({
    channels: z
      .array(diChannelZ)
      .superRefine(Common.Task.validateReadChannels)
      .superRefine(validateDigitalPortsAndLines),
  })
  .superRefine(Common.Task.validateStreamRate);
export interface DigitalReadConfig extends z.infer<typeof digitalReadConfigZ> {}
const ZERO_DIGITAL_READ_CONFIG: DigitalReadConfig = {
  ...ZERO_BASE_READ_CONFIG,
  channels: [],
};

export interface DigitalReadStateDetails extends BaseStateDetails {}
export interface DigitalReadState extends task.State<DigitalReadStateDetails> {}

export const DIGITAL_READ_TYPE = `${PREFIX}_digital_read`;
export type DigitalReadType = typeof DIGITAL_READ_TYPE;

export interface DigitalReadPayload
  extends task.Payload<DigitalReadConfig, DigitalReadStateDetails, DigitalReadType> {}
export const ZERO_DIGITAL_READ_PAYLOAD: DigitalReadPayload = {
  key: "",
  name: "NI Digital Read Task",
  config: ZERO_DIGITAL_READ_CONFIG,
  type: DIGITAL_READ_TYPE,
};

export interface DigitalReadTask
  extends task.Task<DigitalReadConfig, DigitalReadStateDetails, DigitalReadType> {}
export interface NewDigitalReadTask
  extends task.New<DigitalReadConfig, DigitalReadType> {}

export const digitalWriteConfigZ = baseWriteConfigZ.extend({
  channels: z
    .array(doChannelZ)
    .superRefine(Common.Task.validateWriteChannels)
    .superRefine(validateDigitalPortsAndLines),
});
export interface DigitalWriteConfig extends z.infer<typeof digitalWriteConfigZ> {}
const ZERO_DIGITAL_WRITE_CONFIG: DigitalWriteConfig = {
  ...ZERO_BASE_WRITE_CONFIG,
  channels: [],
};

export interface DigitalWriteStateDetails extends BaseStateDetails {}
export interface DigitalWriteState extends task.State<DigitalWriteStateDetails> {}

export const DIGITAL_WRITE_TYPE = `${PREFIX}_digital_write`;
export type DigitalWriteType = typeof DIGITAL_WRITE_TYPE;

export interface DigitalWritePayload
  extends task.Payload<
    DigitalWriteConfig,
    DigitalWriteStateDetails,
    DigitalWriteType
  > {}
export const ZERO_DIGITAL_WRITE_PAYLOAD: DigitalWritePayload = {
  key: "",
  name: "NI Digital Write Task",
  config: ZERO_DIGITAL_WRITE_CONFIG,
  type: DIGITAL_WRITE_TYPE,
};

export interface DigitalWriteTask
  extends task.Task<DigitalWriteConfig, DigitalWriteStateDetails, DigitalWriteType> {}
export interface NewDigitalWriteTask
  extends task.New<DigitalWriteConfig, DigitalWriteType> {}

export type ScanConfig = { enabled: boolean };

export interface ScanStateDetails {
  error?: string;
  message?: string;
}
export interface ScanState extends task.State<ScanStateDetails> {}

export const SCAN_TYPE = `${PREFIX}_scanner`;
export type ScanType = typeof SCAN_TYPE;

export const SCAN_NAME = "ni scanner";

export interface ScanPayload
  extends task.Payload<ScanConfig, ScanStateDetails, ScanType> {}

export interface ScanTask extends task.Task<ScanConfig, ScanStateDetails, ScanType> {}
export interface NewScanTask extends task.New<ScanConfig, ScanType> {}
