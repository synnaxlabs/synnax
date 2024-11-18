// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { device, type task } from "@synnaxlabs/client";
import { migrate } from "@synnaxlabs/x";
import { z } from "zod";

import type * as v0 from "@/hardware/ni/task/migrations/v0";
import { thermocoupleTypeZ } from "@/hardware/task/common/thermocouple";

export const unitsVoltsZ = z.literal("Volts");
export type UnitsVolts = z.infer<typeof unitsVoltsZ>;
export const unitsAmpsZ = z.literal("Amps");
export type UnitsAmps = z.infer<typeof unitsAmpsZ>;
export const unitsDegFZ = z.literal("DegF");
export type UnitsDegF = z.infer<typeof unitsDegFZ>;
export const unitsDegCZ = z.literal("DegC");
export type UnitsDegC = z.infer<typeof unitsDegCZ>;
export const unitsDegRZ = z.literal("DegR");
export type UnitsDegR = z.infer<typeof unitsDegRZ>;
export const unitsKelvinsZ = z.literal("Kelvins");
export type UnitsKelvins = z.infer<typeof unitsKelvinsZ>;
export const unitsStrainZ = z.literal("Strain");
export type UnitsStrain = z.infer<typeof unitsStrainZ>;
export const unitsOhmsZ = z.literal("Ohms");
export type UnitsOhms = z.infer<typeof unitsOhmsZ>;
export const unitsHzZ = z.literal("Hz");
export type UnitsHz = z.infer<typeof unitsHzZ>;
export const unitsSecondsZ = z.literal("Seconds");
export type UnitsSeconds = z.infer<typeof unitsSecondsZ>;
export const unitsMetersZ = z.literal("Meters");
export type UnitsMeters = z.infer<typeof unitsMetersZ>;
export const unitsInchesZ = z.literal("Inches");
export type UnitsInches = z.infer<typeof unitsInchesZ>;
export const unitsDegAngleZ = z.literal("Degrees");
export type UnitsDegAngle = z.infer<typeof unitsDegAngleZ>;
export const unitsRadiansAngleZ = z.literal("Radians");
export type UnitsRadiansAngle = z.infer<typeof unitsRadiansAngleZ>;
export const unitsGravityZ = z.literal("g");
export type UnitsGravity = z.infer<typeof unitsGravityZ>;
export const unitsMetersPerSecondSquaredZ = z.literal("MetersPerSecondSquared");
export type UnitsMetersPerSecondSquared = z.infer<typeof unitsMetersPerSecondSquaredZ>;
export const unitsNewtonsZ = z.literal("Newtons");
export type UnitsNewtons = z.infer<typeof unitsNewtonsZ>;
export const unitsPoundsZ = z.literal("Pounds");
export type UnitsPounds = z.infer<typeof unitsPoundsZ>;
export const unitsKgForceZ = z.literal("KilogramForce");
export type UnitsKgForce = z.infer<typeof unitsKgForceZ>;
export const unitsLbsPerSquareInch = z.literal("PoundsPerSquareInch");
export type UnitsLbsPerSquareInch = z.infer<typeof unitsLbsPerSquareInch>;
export const unitsBarZ = z.literal("Bar");
export type UnitsBar = z.infer<typeof unitsBarZ>;
export const unitsPascalsZ = z.literal("Pascals");
export type UnitsPascals = z.infer<typeof unitsPascalsZ>;
export const unitsVoltsPerVoltZ = z.literal("VoltsPerVolt");
export type UnitsVoltsPerVolt = z.infer<typeof unitsVoltsPerVoltZ>;
export const unitsmVoltsPerVoltZ = z.literal("mVoltsPerVolt");
export type UnitsmVoltsPerVolt = z.infer<typeof unitsmVoltsPerVoltZ>;
export const unitsNewtonMetersZ = z.literal("NewtonMeters");
export type UnitsNewtonMeters = z.infer<typeof unitsNewtonMetersZ>;
export const unitsInchLbsZ = z.literal("InchPounds");
export type UnitsInchLbs = z.infer<typeof unitsInchLbsZ>;
export const unitsInOzZ = z.literal("InchOunces");
export type UnitsInOz = z.infer<typeof unitsInOzZ>;
export const unitsFtLbsZ = z.literal("FootPounds");
export type UnitsFtLbs = z.infer<typeof unitsFtLbsZ>;

export const unitsZ = z.union([
  unitsVoltsZ,
  unitsAmpsZ,
  unitsDegFZ,
  unitsDegCZ,
  unitsDegRZ,
  unitsKelvinsZ,
  unitsStrainZ,
  unitsOhmsZ,
  unitsHzZ,
  unitsSecondsZ,
  unitsMetersZ,
  unitsInchesZ,
  unitsDegAngleZ,
  unitsRadiansAngleZ,
  unitsGravityZ,
  unitsMetersPerSecondSquaredZ,
  unitsNewtonsZ,
  unitsPoundsZ,
  unitsKgForceZ,
  unitsLbsPerSquareInch,
  unitsBarZ,
  unitsPascalsZ,
  unitsVoltsPerVoltZ,
  unitsmVoltsPerVoltZ,
  unitsNewtonMetersZ,
  unitsInchLbsZ,
  unitsInOzZ,
  unitsFtLbsZ,
]);

export type Units = z.infer<typeof unitsZ>;

export const linScaleZ = z.object({
  type: z.literal("linear"),
  slope: z.number().refine((val) => val !== 0, {
    message: "Value must be non-zero",
  }),
  yIntercept: z.number(),
  preScaledUnits: unitsZ,
  scaledUnits: unitsZ,
});

export type LinScale = z.infer<typeof linScaleZ>;

export const ZERO_LIN_SCALE: LinScale = {
  type: "linear",
  slope: 1,
  yIntercept: 0,
  preScaledUnits: "Volts",
  scaledUnits: "Volts",
};

export const mapScaleZ = z.object({
  type: z.literal("map"),
  preScaledMin: z.number(),
  preScaledMax: z.number(),
  scaledMin: z.number(),
  scaledMax: z.number(),
  preScaledUnits: unitsZ,
});

export type MapScale = z.infer<typeof mapScaleZ>;

export const ZERO_MAP_SCALE: MapScale = {
  type: "map",
  preScaledMin: 0,
  preScaledMax: 0,
  scaledMin: 0,
  scaledMax: 0,
  preScaledUnits: "Volts",
};

export const tableScaleZ = z.object({
  type: z.literal("table"),
  preScaledVals: z.array(z.number()),
  scaledVals: z.array(z.number()),
  preScaledUnits: unitsZ,
});

export type TableScale = z.infer<typeof tableScaleZ>;

export const ZERO_TABLE_SCALE: z.infer<typeof tableScaleZ> = {
  type: "table",
  preScaledVals: [],
  scaledVals: [],
  preScaledUnits: "Volts",
};

export const noScaleZ = z.object({
  type: z.literal("none"),
});

export const ZERO_NO_SCALE: z.infer<typeof noScaleZ> = {
  type: "none",
};

export const scaleZ = z.union([linScaleZ, mapScaleZ, tableScaleZ, noScaleZ]);
export type Scale = z.infer<typeof scaleZ>;
export type ScaleType = Scale["type"];

export const ZERO_SCALES: Record<ScaleType, Scale> = {
  linear: ZERO_LIN_SCALE,
  map: ZERO_MAP_SCALE,
  table: ZERO_TABLE_SCALE,
  none: ZERO_NO_SCALE,
};

export const SCALE_SCHEMAS: Record<ScaleType, z.ZodType<Scale>> = {
  linear: linScaleZ,
  map: mapScaleZ,
  table: tableScaleZ,
  none: noScaleZ,
};

const terminalConfigZ = z.enum(["Cfg_Default", "RSE", "NRSE", "Diff", "PseudoDiff"]);

export type TerminalConfig = z.infer<typeof terminalConfigZ>;

const excitSourceZ = z.enum(["Internal", "External", "None"]);

export type ExcitationSource = z.infer<typeof excitSourceZ>;

const baseAIChanZ = z.object({
  device: device.deviceKeyZ,
  name: z.string(),
  key: z.string(),
  channel: z.number().optional(),
  port: z.number(),
  enabled: z.boolean(),
});

const minMaxValZ = z.object({
  minVal: z.number(),
  maxVal: z.number(),
});

export const sensitivityUnitsZ = z.enum(["mVoltsPerG", "VoltsPerG"]);

export type AccelSensitivityUnits = z.infer<typeof sensitivityUnitsZ>;

export const accelerationUnitsZ = z.enum([
  "g",
  "MetersPerSecondSquared",
  "InchesPerSecondSquared",
]);

export type AccelerationUnits = z.infer<typeof accelerationUnitsZ>;

const baseAiAccelChanZ = baseAIChanZ.merge(minMaxValZ).extend({
  terminalConfig: terminalConfigZ,
  sensitivity: z.number(),
  sensitivityUnits: sensitivityUnitsZ,
});

export const forceUnitsZ = z.enum(["Newtons", "Pounds", "KilogramForce"]);
export type ForceUnits = z.infer<typeof forceUnitsZ>;

export const electricalUnitsZ = z.enum(["mVoltsPerVolt", "VoltsPerVolt"]);
export type ElectricalUnits = z.infer<typeof electricalUnitsZ>;

// 1 - https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaiaccelchan.html
export const aiAccelChanZ = baseAiAccelChanZ.extend({
  type: z.literal("ai_accel"),
  units: accelerationUnitsZ,
  currentExcitSource: excitSourceZ,
  currentExcitVal: z.number(),
  customScale: scaleZ,
});

export type AIAccelChan = z.infer<typeof aiAccelChanZ>;

export const ZERO_AI_ACCEL_CHAN: AIAccelChan = {
  name: "",
  key: "",
  device: "",
  type: "ai_accel",
  channel: 0,
  port: 0,
  units: "g",
  enabled: true,
  minVal: 0,
  maxVal: 1,
  terminalConfig: "Cfg_Default",
  sensitivity: 0,
  sensitivityUnits: "mVoltsPerG",
  currentExcitSource: "Internal",
  currentExcitVal: 0,
  customScale: ZERO_NO_SCALE,
};

// 2 - https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaiaccel4wiredcvoltagechan.html
export const aiAccel4WireDCVoltageChanZ = baseAiAccelChanZ.extend({
  type: z.literal("ai_accel_4_wire_dc_voltage"),
  units: accelerationUnitsZ,
  voltageExcitSource: excitSourceZ,
  voltageExcitVal: z.number(),
  useExcitForScaling: z.boolean(),
  customScale: scaleZ,
});

export type AIAccel4WireDCVoltageChan = z.infer<typeof aiAccel4WireDCVoltageChanZ>;

export const ZERO_AI_ACCEL_4WIRE_DC_VOLTAGE_CHAN: AIAccel4WireDCVoltageChan = {
  name: "",
  key: "",
  device: "",
  type: "ai_accel_4_wire_dc_voltage",
  units: "g",
  channel: 0,
  port: 0,
  enabled: true,
  minVal: 0,
  maxVal: 1,
  terminalConfig: "Cfg_Default",
  sensitivity: 0,
  sensitivityUnits: "mVoltsPerG",
  voltageExcitSource: "Internal",
  voltageExcitVal: 0,
  useExcitForScaling: false,
  customScale: ZERO_NO_SCALE,
};

// 3 - https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaiaccelchargechan.html
export const aiAccelChargeChanZ = baseAiAccelChanZ.extend({
  type: z.literal("ai_accel_charge"),
  units: accelerationUnitsZ,
  customScale: scaleZ,
});

export type AIAccelChargeChan = z.infer<typeof aiAccelChargeChanZ>;

export const ZERO_AI_ACCEL_CHARGE_CHAN: AIAccelChargeChan = {
  name: "",
  key: "",
  device: "",
  type: "ai_accel_charge",
  channel: 0,
  port: 0,
  enabled: true,
  minVal: 0,
  maxVal: 1,
  terminalConfig: "Cfg_Default",
  sensitivity: 0,
  sensitivityUnits: "mVoltsPerG",
  units: "g",
  customScale: ZERO_NO_SCALE,
};

export const bridgeConfigZ = z.enum(["FullBridge", "HalfBridge", "QuarterBridge"]);
export type BridgeConfig = z.infer<typeof bridgeConfigZ>;

// 4 - https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaibridgechan.html
const aiBridgeChanZ = baseAIChanZ.extend({
  type: z.literal("ai_bridge"),
  terminalConfig: terminalConfigZ,
  units: electricalUnitsZ,
  minVal: z.number(),
  maxVal: z.number(),
  bridgeConfig: bridgeConfigZ,
  voltageExcitSource: excitSourceZ,
  voltageExcitVal: z.number(),
  nominalBridgeResistance: z.number().refine((val) => val > 0, {
    message: "Value must be greater than 0",
  }),
  customScale: scaleZ,
});

export type AIBridgeChan = z.infer<typeof aiBridgeChanZ>;

export const ZERO_AI_BRIDGE_CHAN: AIBridgeChan = {
  name: "",
  key: "",
  device: "",
  type: "ai_bridge",
  units: "mVoltsPerVolt",
  channel: 0,
  port: 0,
  enabled: true,
  minVal: 0,
  maxVal: 1,
  terminalConfig: "Cfg_Default",
  bridgeConfig: "FullBridge",
  voltageExcitSource: "Internal",
  voltageExcitVal: 0,
  nominalBridgeResistance: 1,
  customScale: ZERO_NO_SCALE,
};

// 5 - https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaichargechan.html
export const aiChargeChan = baseAIChanZ.extend({
  type: z.literal("ai_charge"),
  terminalConfig: terminalConfigZ,
  minVal: z.number(),
  maxVal: z.number(),
  units: z.enum(["C", "uC"]),
  customScale: scaleZ,
});

export type AIChargeChan = z.infer<typeof aiChargeChan>;

export const ZERO_AI_CHARGE_CHAN: AIChargeChan = {
  name: "",
  key: "",
  device: "",
  channel: 0,
  type: "ai_charge",
  port: 0,
  enabled: true,
  minVal: 0,
  maxVal: 1,
  terminalConfig: "Cfg_Default",
  units: "C",
  customScale: ZERO_NO_SCALE,
};

const currentUnitsZ = z.enum(["Amps"]);

const shuntResistorLocZ = z.enum(["Default", "Internal", "External"]);

const shuntResistorValZ = z.number().refine((val) => val > 0, {
  message: "Value must be greater than 0",
});

export type ShuntResistorLoc = z.infer<typeof shuntResistorLocZ>;

// 6 - https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaicurrentchan.html
const aiCurrentChanZ = baseAIChanZ.extend({
  type: z.literal("ai_current"),
  terminalConfig: terminalConfigZ,
  minVal: z.number(),
  maxVal: z.number(),
  units: currentUnitsZ,
  shuntResistorLoc: shuntResistorLocZ,
  extShuntResistorVal: shuntResistorValZ,
  customScale: scaleZ,
});

export type AICurrentChan = z.infer<typeof aiCurrentChanZ>;

export const ZERO_AI_CURRENT_CHAN: AICurrentChan = {
  name: "",
  key: "",
  device: "",
  channel: 0,
  port: 0,
  type: "ai_current",
  enabled: true,
  minVal: 0,
  maxVal: 1,
  terminalConfig: "Cfg_Default",
  units: "Amps",
  shuntResistorLoc: "Default",
  extShuntResistorVal: 1,
  customScale: ZERO_NO_SCALE,
};

// 7 - https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaicurrentrmschan.html
export const aiCurrentRMSChanZ = baseAIChanZ.extend({
  type: z.literal("ai_current_rms"),
  terminalConfig: terminalConfigZ,
  minVal: z.number(),
  maxVal: z.number(),
  units: currentUnitsZ,
  shuntResistorLoc: shuntResistorLocZ,
  extShuntResistorVal: shuntResistorValZ,
  customScale: scaleZ,
});

export type AICurrentRMSChan = z.infer<typeof aiCurrentRMSChanZ>;

export const ZERO_AI_CURRENT_RMS_CHAN: AICurrentRMSChan = {
  name: "",
  key: "",
  device: "",
  channel: 0,
  type: "ai_current_rms",
  port: 0,
  enabled: true,
  minVal: 0,
  maxVal: 1,
  terminalConfig: "Cfg_Default",
  units: "Amps",
  shuntResistorLoc: "Default",
  extShuntResistorVal: 0,
  customScale: ZERO_NO_SCALE,
};

// 8 - https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaiforcebridgepolynomialchan.html
export const aiForceBridgePolynomialChanZ = baseAIChanZ.extend({
  type: z.literal("ai_force_bridge_polynomial"),
  terminalConfig: terminalConfigZ,
  minVal: z.number(),
  maxVal: z.number(),
  units: forceUnitsZ,
  bridgeConfig: bridgeConfigZ,
  voltageExcitSource: excitSourceZ,
  voltageExcitVal: z.number(),
  nominalBridgeResistance: z.number(),
  forwardCoeffs: z.array(z.number()),
  reverseCoeffs: z.array(z.number()),
  electricalUnits: electricalUnitsZ,
  physicalUnits: forceUnitsZ,
  customScale: scaleZ,
});

export type AIForceBridgePolynomialChan = z.infer<typeof aiForceBridgePolynomialChanZ>;

export const ZERO_AI_FORCE_BRIDGE_POLYNOMIAL_CHAN: AIForceBridgePolynomialChan = {
  name: "",
  key: "",
  device: "",
  type: "ai_force_bridge_polynomial",
  channel: 0,
  port: 0,
  enabled: true,
  minVal: 0,
  maxVal: 1,
  terminalConfig: "Cfg_Default",
  units: "Newtons",
  bridgeConfig: "FullBridge",
  voltageExcitSource: "Internal",
  voltageExcitVal: 0,
  nominalBridgeResistance: 0,
  forwardCoeffs: [],
  reverseCoeffs: [],
  electricalUnits: "mVoltsPerVolt",
  physicalUnits: "Newtons",
  customScale: ZERO_NO_SCALE,
};

// 9 - https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaiforcebridgetablechan.html
const aiForceBridgeTableChanZ = baseAIChanZ.extend({
  type: z.literal("ai_force_bridge_table"),
  minVal: z.number(),
  maxVal: z.number(),
  units: forceUnitsZ,
  bridgeConfig: bridgeConfigZ,
  voltageExcitSource: excitSourceZ,
  voltageExcitVal: z.number(),
  nominalBridgeResistance: z.number(),
  electricalUnits: electricalUnitsZ,
  electricalVals: z.array(z.number()),
  physicalUnits: forceUnitsZ,
  physicalVals: z.array(z.number()),
  customScale: scaleZ,
});

export type AIForceBridgeTableChan = z.infer<typeof aiForceBridgeTableChanZ>;

export const ZERO_AI_FORCE_BRIDGE_TABLE_CHAN: AIForceBridgeTableChan = {
  name: "",
  key: "",
  device: "",
  type: "ai_force_bridge_table",
  channel: 0,
  port: 0,
  enabled: true,
  minVal: 0,
  maxVal: 1,
  units: "Newtons",
  bridgeConfig: "FullBridge",
  voltageExcitSource: "Internal",
  voltageExcitVal: 0,
  nominalBridgeResistance: 0,
  electricalUnits: "mVoltsPerVolt",
  electricalVals: [],
  physicalUnits: "Newtons",
  physicalVals: [],
  customScale: ZERO_NO_SCALE,
};

// 10 - https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaiforcebridgetwopointlinchan.html
const aiForceBridgeTwoPointLinChan = baseAIChanZ.extend({
  type: z.literal("ai_force_bridge_two_point_lin"),
  terminalConfig: terminalConfigZ,
  minVal: z.number(),
  maxVal: z.number(),
  units: forceUnitsZ,
  bridgeConfig: bridgeConfigZ,
  voltageExcitSource: excitSourceZ,
  voltageExcitVal: z.number(),
  nominalBridgeResistance: z.number(),
  electricalUnits: electricalUnitsZ,
  physicalUnits: forceUnitsZ,
  firstElectricalVal: z.number(),
  firstPhysicalVal: z.number(),
  secondElectricalVal: z.number(),
  secondPhysicalVal: z.number(),
  customScale: scaleZ,
});

export type AIForceBridgeTwoPointLinChan = z.infer<typeof aiForceBridgeTwoPointLinChan>;

export const ZERO_AI_FORCE_BRIDGE_TWO_POINT_LIN_CHAN: AIForceBridgeTwoPointLinChan = {
  name: "",
  key: "",
  device: "",
  type: "ai_force_bridge_two_point_lin",
  channel: 0,
  port: 0,
  enabled: true,
  minVal: 0,
  maxVal: 1,
  terminalConfig: "Cfg_Default",
  units: "Newtons",
  bridgeConfig: "FullBridge",
  voltageExcitSource: "Internal",
  voltageExcitVal: 0,
  nominalBridgeResistance: 0,
  electricalUnits: "mVoltsPerVolt",
  physicalUnits: "Newtons",
  firstElectricalVal: 0,
  firstPhysicalVal: 0,
  secondElectricalVal: 1,
  secondPhysicalVal: 1,
  customScale: ZERO_NO_SCALE,
};

// 11 - https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaiforceiepechan.html
const aiForgeIEPEChanZ = baseAIChanZ.extend({
  type: z.literal("ai_force_iepe"),
  terminalConfig: terminalConfigZ,
  minVal: z.number(),
  maxVal: z.number(),
  units: forceUnitsZ,
  sensitivity: z.number(),
  sensitivityUnits: electricalUnitsZ,
  currentExcitSource: excitSourceZ,
  currentExcitVal: z.number(),
  customScale: scaleZ,
});

export type AIForceEPEChan = z.infer<typeof aiForgeIEPEChanZ>;

export const ZERO_AI_FORCE_IEPE_CHAN: AIForceEPEChan = {
  name: "",
  key: "",
  device: "",
  type: "ai_force_iepe",
  channel: 0,
  port: 0,
  enabled: true,
  minVal: 0,
  maxVal: 1,
  terminalConfig: "Cfg_Default",
  units: "Newtons",
  sensitivity: 0,
  sensitivityUnits: "mVoltsPerVolt",
  currentExcitSource: "Internal",
  currentExcitVal: 0,
  customScale: ZERO_NO_SCALE,
};

// 12 - https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaifreqvoltagechan.html
export const aiFreqVoltageChanZ = baseAIChanZ.extend({
  type: z.literal("ai_freq_voltage"),
  minVal: z.number(),
  maxVal: z.number(),
  units: z.enum(["Hz"]),
  thresholdLevel: z.number(),
  hysteresis: z.number(),
  customScale: scaleZ,
});

export type AIFreqVoltageChan = z.infer<typeof aiFreqVoltageChanZ>;

export const ZERO_AI_FREQ_VOLTAGE_CHAN: AIFreqVoltageChan = {
  name: "",
  key: "",
  device: "",
  type: "ai_freq_voltage",
  channel: 0,
  port: 0,
  enabled: true,
  minVal: 0,
  maxVal: 1,
  units: "Hz",
  thresholdLevel: 0,
  hysteresis: 0,
  customScale: ZERO_NO_SCALE,
};

// 13 - https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaimicrophonechan.html
const aiMicrophoneChanZ = baseAIChanZ.extend({
  type: z.literal("ai_microphone"),
  terminalConfig: terminalConfigZ,
  micSensitivity: z.number(),
  maxSndPressLevel: z.number(),
  currentExcitSource: excitSourceZ,
  currentExcitVal: z.number(),
  units: z.enum(["Pascals"]),
  customScale: scaleZ,
});

export type AIMicrophoneChan = z.infer<typeof aiMicrophoneChanZ>;

export const ZERO_AI_MICROPHONE_CHAN: AIMicrophoneChan = {
  name: "",
  key: "",
  device: "",
  type: "ai_microphone",
  channel: 0,
  port: 0,
  enabled: true,
  terminalConfig: "Cfg_Default",
  micSensitivity: 0,
  maxSndPressLevel: 0,
  currentExcitSource: "Internal",
  currentExcitVal: 0,
  units: "Pascals",
  customScale: ZERO_NO_SCALE,
};

export const pressureUnitsZ = z.enum(["PoundsPerSquareInch", "Pascals", "Bar"]);
export type PressureUnits = z.infer<typeof pressureUnitsZ>;

// 14 - https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaipressurebridgepolynomialchan.html
export const aiPressureBridgePolynomialChanZ = baseAIChanZ.extend({
  type: z.literal("ai_pressure_bridge_polynomial"),
  minVal: z.number(),
  maxVal: z.number(),
  units: pressureUnitsZ,
  bridgeConfig: bridgeConfigZ,
  voltageExcitSource: excitSourceZ,
  voltageExcitVal: z.number(),
  nominalBridgeResistance: z.number(),
  forwardCoeffs: z.array(z.number()),
  reverseCoeffs: z.array(z.number()),
  electricalUnits: electricalUnitsZ,
  physicalUnits: pressureUnitsZ,
  customScale: scaleZ,
});

export type AIPressureBridgePolynomialChan = z.infer<
  typeof aiPressureBridgePolynomialChanZ
>;

export const ZERO_AI_PRESSURE_BRIDGE_POLYNOMIAL_CHAN: AIPressureBridgePolynomialChan = {
  name: "",
  key: "",
  device: "",
  type: "ai_pressure_bridge_polynomial",
  channel: 0,
  port: 0,
  enabled: true,
  minVal: 0,
  maxVal: 1,
  units: "PoundsPerSquareInch",
  bridgeConfig: "FullBridge",
  voltageExcitSource: "Internal",
  voltageExcitVal: 0,
  nominalBridgeResistance: 0,
  forwardCoeffs: [],
  reverseCoeffs: [],
  electricalUnits: "mVoltsPerVolt",
  physicalUnits: "PoundsPerSquareInch",
  customScale: ZERO_NO_SCALE,
};

// 15 - https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaipressurebridgetablechan.html
const aiPressureBridgeTableChanZ = baseAIChanZ.extend({
  type: z.literal("ai_pressure_bridge_table"),
  minVal: z.number(),
  maxVal: z.number(),
  units: pressureUnitsZ,
  bridgeConfig: bridgeConfigZ,
  voltageExcitSource: excitSourceZ,
  voltageExcitVal: z.number(),
  nominalBridgeResistance: z.number(),
  electricalUnits: z.enum(["mV/V", "V/V"]),
  electricalVals: z.array(z.number()),
  physicalUnits: pressureUnitsZ,
  physicalVals: z.array(z.number()),
  customScale: scaleZ,
});

export type AIPressureBridgeTableChan = z.infer<typeof aiPressureBridgeTableChanZ>;

export const ZERO_AI_PRESSURE_BRIDGE_TABLE_CHAN: AIPressureBridgeTableChan = {
  name: "",
  key: "",
  device: "",
  type: "ai_pressure_bridge_table",
  channel: 0,
  port: 0,
  enabled: true,
  minVal: 0,
  maxVal: 1,
  units: "PoundsPerSquareInch",
  bridgeConfig: "FullBridge",
  voltageExcitSource: "Internal",
  voltageExcitVal: 0,
  nominalBridgeResistance: 0,
  electricalUnits: "mV/V",
  electricalVals: [],
  physicalUnits: "PoundsPerSquareInch",
  physicalVals: [],
  customScale: ZERO_NO_SCALE,
};

// 16 - https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaipressurebridgetwopointlinchan.html
const aiPressureBridgeTwoPointLinChanZ = baseAIChanZ.extend({
  type: z.literal("ai_pressure_bridge_two_point_lin"),
  terminalConfig: terminalConfigZ,
  minVal: z.number(),
  maxVal: z.number(),
  units: pressureUnitsZ,
  bridgeConfig: bridgeConfigZ,
  voltageExcitSource: excitSourceZ,
  voltageExcitVal: z.number(),
  nominalBridgeResistance: z.number(),
  electricalUnits: electricalUnitsZ,
  physicalUnits: pressureUnitsZ,
  firstElectricalVal: z.number(),
  firstPhysicalVal: z.number(),
  secondElectricalVal: z.number(),
  secondPhysicalVal: z.number(),
  customScale: scaleZ,
});

export type AIPressureBridgeTwoPointLinChan = z.infer<
  typeof aiPressureBridgeTwoPointLinChanZ
>;

export const ZERO_AI_PRESSURE_BRIDGE_TWO_POINT_LIN_CHAN: AIPressureBridgeTwoPointLinChan =
  {
    name: "",
    key: "",
    device: "",
    type: "ai_pressure_bridge_two_point_lin",
    channel: 0,
    port: 0,
    enabled: true,
    minVal: 0,
    maxVal: 1,
    terminalConfig: "Cfg_Default",
    units: "PoundsPerSquareInch",
    bridgeConfig: "FullBridge",
    voltageExcitSource: "Internal",
    voltageExcitVal: 0,
    nominalBridgeResistance: 0,
    electricalUnits: "mVoltsPerVolt",
    physicalUnits: "PoundsPerSquareInch",
    firstElectricalVal: 0,
    firstPhysicalVal: 0,
    secondElectricalVal: 1,
    secondPhysicalVal: 1,
    customScale: ZERO_NO_SCALE,
  };

export const resistanceConfigZ = z.enum(["2Wire", "3Wire", "4Wire"]);
export type ResistanceConfig = z.infer<typeof resistanceConfigZ>;

// 17 - https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateairesistancechan.html
const aiResistanceChanZ = baseAIChanZ.extend({
  type: z.literal("ai_resistance"),
  terminalConfig: terminalConfigZ,
  minVal: z.number(),
  maxVal: z.number(),
  units: z.enum(["Ohms"]),
  resistanceConfig: resistanceConfigZ,
  currentExcitSource: excitSourceZ,
  currentExcitVal: z.number(),
  customScale: scaleZ,
});

export type AIResistanceChan = z.infer<typeof aiResistanceChanZ>;

export const ZERO_AI_RESISTANCE_CHAN: AIResistanceChan = {
  name: "",
  key: "",
  device: "",
  type: "ai_resistance",
  channel: 0,
  port: 0,
  enabled: true,
  minVal: 0,
  maxVal: 1,
  terminalConfig: "Cfg_Default",
  units: "Ohms",
  resistanceConfig: "2Wire",
  currentExcitSource: "Internal",
  currentExcitVal: 0,
  customScale: ZERO_NO_SCALE,
};

// 18 -  https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateairosettestraingagechan.html
export const aiRosetteStrainGageChanZ = baseAIChanZ.extend({
  type: z.literal("ai_rosette_strain_gage"),
  terminalConfig: terminalConfigZ,
  minVal: z.number(),
  maxVal: z.number(),
  rosetteType: z.enum(["RectangularRosette", "DeltaRosette", "TeeRosette"]),
  gageOrientation: z.number(),
  rosetteMeasTypes: z.array(
    z.enum([
      "PrincipleStrain1",
      "PrincipleStrain2",
      "PrincipleStrainAngle",
      "CartesianStrainX",
      "CartesianStrainY",
      "CartesianShearStrainXY",
      "MaxShearStrain",
      "MaxShearStrainAngle",
    ]),
  ),
  strainConfig: z.enum([
    "FullBridgeI",
    "FullBridgeII",
    "FullBridgeIII",
    "HalfBridgeI",
    "HalfBridgeII",
    "QuarterBridgeI",
    "QuarterBridgeII",
  ]),
  units: z.enum(["strain"]),
  voltageExcitSource: excitSourceZ,
  voltageExcitVal: z.number(),
  nominalGageResistance: z.number(),
  poissonRatio: z.number(),
  leadWireResistance: z.number(),
  gageFactor: z.number(),
});

export type AIRosetteStrainGageChan = z.infer<typeof aiRosetteStrainGageChanZ>;

export const ZERO_AI_ROSETTE_STRAIN_GAGE_CHAN: AIRosetteStrainGageChan = {
  name: "",
  key: "",
  device: "",
  type: "ai_rosette_strain_gage",
  channel: 0,
  port: 0,
  enabled: true,
  minVal: 0,
  maxVal: 1,
  terminalConfig: "Cfg_Default",
  rosetteType: "RectangularRosette",
  strainConfig: "FullBridgeI",
  gageOrientation: 0,
  rosetteMeasTypes: [],
  units: "strain",
  voltageExcitSource: "Internal",
  voltageExcitVal: 0,
  nominalGageResistance: 0,
  poissonRatio: 0,
  leadWireResistance: 0,
  gageFactor: 0,
};

export const temperatureUnitsZ = z.enum(["DegC", "DegF", "Kelvins", "DegR"]);
export type TemperatureUnits = z.infer<typeof temperatureUnitsZ>;

// 19 -  https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateairtdchan.html
const aiRTDChanZ = baseAIChanZ.extend({
  type: z.literal("ai_rtd"),
  minVal: z.number(),
  maxVal: z.number(),
  units: temperatureUnitsZ,
  rtdType: z.enum([
    "Pt3750",
    "Pt3851",
    "Pt3911",
    "Pt3916",
    "Pt3920",
    "Pt3928",
    "Pt3850",
  ]),
  resistanceConfig: resistanceConfigZ,
  currentExcitSource: excitSourceZ,
  currentExcitVal: z.number(),
  r0: z.number(),
});

export type AIRTDChan = z.infer<typeof aiRTDChanZ>;

export const ZERO_AI_RTD_CHAN: AIRTDChan = {
  name: "",
  key: "",
  device: "",
  channel: 0,
  type: "ai_rtd",
  port: 0,
  enabled: true,
  minVal: 0,
  maxVal: 1,
  units: "DegC",
  rtdType: "Pt3750",
  resistanceConfig: "2Wire",
  currentExcitSource: "Internal",
  currentExcitVal: 0,
  r0: 0,
};

// 20 - https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaistraingagechan.html
const aiStrainGageChan = baseAIChanZ.extend({
  type: z.literal("ai_strain_gauge"),
  terminalConfig: terminalConfigZ,
  minVal: z.number(),
  maxVal: z.number(),
  units: z.enum(["strain"]),
  strainConfig: z.enum([
    "full-bridge-I",
    "full-bridge-II",
    "full-bridge-III",
    "half-bridge-I",
    "half-bridge-II",
    "quarter-bridge-I",
    "quarter-bridge-II",
  ]),
  voltageExcitSource: excitSourceZ,
  voltageExcitVal: z.number(),
  gageFactor: z.number(),
  initialBridgeVoltage: z.number(),
  nominalGageResistance: z.number(),
  poissonRatio: z.number(),
  leadWireResistance: z.number(),
  customScale: scaleZ,
});

export type AIStrainGageChan = z.infer<typeof aiStrainGageChan>;

export const ZERO_AI_STRAIN_GAGE_CHAN: AIStrainGageChan = {
  name: "",
  key: "",
  device: "",
  type: "ai_strain_gauge",
  channel: 0,
  port: 0,
  enabled: true,
  minVal: 0,
  maxVal: 1,
  terminalConfig: "Cfg_Default",
  units: "strain",
  strainConfig: "full-bridge-I",
  voltageExcitSource: "Internal",
  voltageExcitVal: 0,
  gageFactor: 0,
  initialBridgeVoltage: 0,
  nominalGageResistance: 0,
  poissonRatio: 0,
  leadWireResistance: 0,
  customScale: ZERO_NO_SCALE,
};

// 21 - https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaitempbuiltinsensorchan.html
const aiTempBuiltInChanZ = baseAIChanZ.extend({
  type: z.literal("ai_temp_builtin"),
  units: temperatureUnitsZ,
});

export type AITempBuiltInChan = z.infer<typeof aiTempBuiltInChanZ>;

export const ZERO_AI_TEMP_BUILTIN_CHAN: AITempBuiltInChan = {
  name: "",
  key: "",
  device: "",
  type: "ai_temp_builtin",
  channel: 0,
  port: 0,
  enabled: true,
  units: "DegC",
};

// 22 - https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaithrmcplchan.html
const aiThermocoupleChanZ = baseAIChanZ
  .extend({
    key: z.string(),
    type: z.literal("ai_thermocouple"),
    minVal: z.number(),
    maxVal: z.number(),
    units: temperatureUnitsZ,
    thermocoupleType: thermocoupleTypeZ,
    cjcSource: z.enum(["BuiltIn", "ConstVal", "Chan"]),
    cjcVal: z.number(),
    cjcPort: z.number(),
  })
  .refine(
    (v) => {
      if (v.cjcSource === "ConstVal") return v.cjcVal !== undefined;
      return true;
    },
    {
      path: ["cjcVal"],
      message: "CJC Value must be defined when CJC Source is ConstVal",
    },
  )
  .refine(
    (v) => {
      if (v.cjcSource === "Chan") return v.cjcPort !== undefined;
      return true;
    },
    {
      path: ["cjcPort"],
      message: "CJC Port must be defined when CJC Source is ConstVal",
    },
  );

export type AIThermocoupleChan = z.infer<typeof aiThermocoupleChanZ>;

export const ZERO_AI_THERMOCOUPLE_CHAN: AIThermocoupleChan = {
  name: "",
  key: "",
  device: "",
  type: "ai_thermocouple",
  channel: 0,
  port: 0,
  enabled: true,
  minVal: 0,
  maxVal: 1,
  units: "DegC",
  thermocoupleType: "J",
  cjcSource: "BuiltIn",
  cjcVal: 0,
  cjcPort: 0,
};

// 23 - https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaithrmstrchaniex.html
export const aiThermistorChanIex = baseAIChanZ.extend({
  key: z.string(),
  type: z.literal("ai_thermistor_iex"),
  minVal: z.number(),
  maxVal: z.number(),
  units: temperatureUnitsZ,
  resistanceConfig: resistanceConfigZ,
  currentExcitSource: excitSourceZ,
  currentExcitVal: z.number(),
  a: z.number(),
  b: z.number(),
  c: z.number(),
});

export type AIThermistorChanIex = z.infer<typeof aiThermistorChanIex>;

export const ZERO_AI_THERMISTOR_CHAN_IEX: AIThermistorChanIex = {
  name: "",
  key: "",
  device: "",
  type: "ai_thermistor_iex",
  channel: 0,
  port: 0,
  enabled: true,
  minVal: 0,
  maxVal: 1,
  units: "DegC",
  resistanceConfig: "2Wire",
  currentExcitSource: "Internal",
  currentExcitVal: 0,
  a: 0,
  b: 0,
  c: 0,
};

// 24 - https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaitorquebridgepolynomialchan.html
export const aiThermistorChanVex = baseAIChanZ.extend({
  key: z.string(),
  type: z.literal("ai_thermistor_vex"),
  minVal: z.number(),
  maxVal: z.number(),
  units: temperatureUnitsZ,
  resistanceConfig: resistanceConfigZ,
  voltageExcitSource: excitSourceZ,
  voltageExcitVal: z.number(),
  a: z.number(),
  b: z.number(),
  c: z.number(),
  r1: z.number(),
});

export type AIThermistorChanVex = z.infer<typeof aiThermistorChanVex>;

export const ZERO_AI_THERMISTOR_CHAN_VEX: AIThermistorChanVex = {
  name: "",
  key: "",
  device: "",
  channel: 0,
  type: "ai_thermistor_vex",
  port: 0,
  enabled: true,
  minVal: 0,
  maxVal: 1,
  units: "DegC",
  resistanceConfig: "2Wire",
  voltageExcitSource: "Internal",
  voltageExcitVal: 0,
  a: 0,
  b: 0,
  c: 0,
  r1: 0,
};

export const torqueUnitsZ = z.enum(["NewtonMeters", "InchOunces", "FootPounds"]);
export type TorqueUnits = z.infer<typeof torqueUnitsZ>;

// 25 - https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaitorquebridgepolynomialchan.html
export const aiTorqueBridgePolynomialChanZ = baseAIChanZ.extend({
  key: z.string(),
  type: z.literal("ai_torque_bridge_polynomial"),
  minVal: z.number(),
  maxVal: z.number(),
  units: torqueUnitsZ,
  bridgeConfig: bridgeConfigZ,
  voltageExcitSource: excitSourceZ,
  voltageExcitVal: z.number(),
  nominalBridgeResistance: z.number(),
  forwardCoeffs: z.array(z.number()),
  reverseCoeffs: z.array(z.number()),
  electricalUnits: electricalUnitsZ,
  physicalUnits: torqueUnitsZ,
  customScale: scaleZ,
});

export type AITorqueBridgePolynomialChan = z.infer<
  typeof aiTorqueBridgePolynomialChanZ
>;

export const ZERO_AI_TORQUE_BRIDGE_POLYNOMIAL_CHAN: AITorqueBridgePolynomialChan = {
  name: "",
  key: "",
  device: "",
  type: "ai_torque_bridge_polynomial",
  channel: 0,
  port: 0,
  enabled: true,
  minVal: 0,
  maxVal: 1,
  units: "NewtonMeters",
  bridgeConfig: "FullBridge",
  voltageExcitSource: "Internal",
  voltageExcitVal: 0,
  nominalBridgeResistance: 0,
  forwardCoeffs: [],
  reverseCoeffs: [],
  electricalUnits: "mVoltsPerVolt",
  physicalUnits: "NewtonMeters",
  customScale: ZERO_NO_SCALE,
};

// 26 - https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaitorquebridgetablechan.html
const aiTorqueBridgeTableChanZ = baseAIChanZ.extend({
  key: z.string(),
  type: z.literal("ai_torque_bridge_table"),
  minVal: z.number(),
  maxVal: z.number(),
  units: torqueUnitsZ,
  bridgeConfig: bridgeConfigZ,
  voltageExcitSource: excitSourceZ,
  voltageExcitVal: z.number(),
  nominalBridgeResistance: z.number(),
  electricalUnits: electricalUnitsZ,
  electricalVals: z.array(z.number()),
  physicalUnits: torqueUnitsZ,
  physicalVals: z.array(z.number()),
  customScale: scaleZ,
});

export type AITorqueBridgeTableChan = z.infer<typeof aiTorqueBridgeTableChanZ>;

export const ZERO_AI_TORQUE_BRIDGE_TABLE_CHAN: AITorqueBridgeTableChan = {
  name: "",
  key: "",
  device: "",
  type: "ai_torque_bridge_table",
  channel: 0,
  port: 0,
  enabled: true,
  minVal: 0,
  maxVal: 1,
  units: "NewtonMeters",
  bridgeConfig: "FullBridge",
  voltageExcitSource: "Internal",
  voltageExcitVal: 0,
  nominalBridgeResistance: 0,
  electricalUnits: "mVoltsPerVolt",
  electricalVals: [],
  physicalUnits: "NewtonMeters",
  physicalVals: [],
  customScale: ZERO_NO_SCALE,
};

// 27 - https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaitorquebridgetwopointlinchan.html
const aiTorqueBridgeTwoPointLinChanZ = baseAIChanZ.extend({
  key: z.string(),
  type: z.literal("ai_torque_bridge_two_point_lin"),
  minVal: z.number(),
  maxVal: z.number(),
  units: torqueUnitsZ,
  bridgeConfig: bridgeConfigZ,
  voltageExcitSource: excitSourceZ,
  voltageExcitVal: z.number(),
  nominalBridgeResistance: z.number(),
  electricalUnits: electricalUnitsZ,
  physicalUnits: torqueUnitsZ,
  firstElectricalVal: z.number(),
  firstPhysicalVal: z.number(),
  secondElectricalVal: z.number(),
  secondPhysicalVal: z.number(),
  customScale: scaleZ,
});

export type AITorqueBridgeTwoPointLinChan = z.infer<
  typeof aiTorqueBridgeTwoPointLinChanZ
>;

export const ZERO_AI_TORQUE_BRIDGE_TWO_POINT_LIN_CHAN: AITorqueBridgeTwoPointLinChan = {
  name: "",
  key: "",
  device: "",
  type: "ai_torque_bridge_two_point_lin",
  channel: 0,
  port: 0,
  enabled: true,
  minVal: 0,
  maxVal: 1,
  units: "NewtonMeters",
  bridgeConfig: "FullBridge",
  voltageExcitSource: "Internal",
  voltageExcitVal: 0,
  nominalBridgeResistance: 0,
  electricalUnits: "mVoltsPerVolt",
  physicalUnits: "NewtonMeters",
  firstElectricalVal: 0,
  firstPhysicalVal: 0,
  secondElectricalVal: 1,
  secondPhysicalVal: 1,
  customScale: ZERO_NO_SCALE,
};

export const velocityUnitsZ = z.enum(["MetersPerSecond", "InchesPerSecond"]);
export type VelocityUnits = z.infer<typeof velocityUnitsZ>;

export const velocitySensitivityUnitsZ = z.enum([
  "MillivoltsPerMillimeterPerSecond",
  "MilliVoltsPerInchPerSecond",
]);
export type VelocitySensitivityUnits = z.infer<typeof velocitySensitivityUnitsZ>;

// 28 - https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaivelocityiepechan.html
const aiVelocityIEPEChanZ = baseAIChanZ.extend({
  key: z.string(),
  type: z.literal("ai_velocity_iepe"),
  terminalConfig: terminalConfigZ,
  minVal: z.number(),
  maxVal: z.number(),
  units: velocityUnitsZ,
  sensitivity: z.number(),
  sensitivityUnits: velocitySensitivityUnitsZ,
  currentExcitSource: excitSourceZ,
  currentExcitVal: z.number(),
  customScale: scaleZ,
});

export type AIVelocityEPEChan = z.infer<typeof aiVelocityIEPEChanZ>;

export const ZERO_AI_VELOCITY_EPE_CHAN: AIVelocityEPEChan = {
  name: "",
  key: "",
  device: "",
  type: "ai_velocity_iepe",
  channel: 0,
  port: 0,
  enabled: true,
  terminalConfig: "Cfg_Default",
  minVal: 0,
  maxVal: 1,
  units: "MetersPerSecond",
  sensitivity: 0,
  sensitivityUnits: "MillivoltsPerMillimeterPerSecond",
  currentExcitSource: "Internal",
  currentExcitVal: 0,
  customScale: ZERO_NO_SCALE,
};

// 29 - https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaivoltagechan.html
const aiVoltageChanZ = baseAIChanZ.extend({
  key: z.string(),
  type: z.literal("ai_voltage"),
  terminalConfig: terminalConfigZ,
  port: z.number(),
  channel: z.number(),
  enabled: z.boolean(),
  minVal: z.number(),
  maxVal: z.number(),
  units: z.enum(["Volts"]),
  customScale: scaleZ,
});

export type AIVoltageChan = z.infer<typeof aiVoltageChanZ>;

export const ZERO_AI_VOLTAGE_CHAN: AIVoltageChan = {
  name: "",
  key: "",
  device: "",
  type: "ai_voltage",
  terminalConfig: "Cfg_Default",
  channel: 0,
  port: 0,
  enabled: true,
  minVal: 0,
  maxVal: 1,
  units: "Volts",
  customScale: ZERO_NO_SCALE,
};

// 30 - https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaivoltagermschan.html
export const aiVoltageRMSChanZ = baseAIChanZ.extend({
  key: z.string(),
  type: z.literal("ai_voltage_rms"),
  terminalConfig: terminalConfigZ,
  minVal: z.number(),
  maxVal: z.number(),
  units: z.enum(["V", "mV"]),
  customScale: scaleZ,
});

export type AIVoltageRMSChan = z.infer<typeof aiVoltageRMSChanZ>;

export const ZERO_AI_VOLTAGE_RMS_CHAN: AIVoltageRMSChan = {
  name: "",
  key: "",
  device: "",
  type: "ai_voltage_rms",
  channel: 0,
  port: 0,
  enabled: true,
  terminalConfig: "Cfg_Default",
  minVal: 0,
  maxVal: 1,
  units: "V",
  customScale: ZERO_NO_SCALE,
};

// 31 - https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaivoltagechanwithexcit.html
export const aiVoltageChanWithExcitZ = baseAIChanZ.extend({
  type: z.literal("ai_voltage_with_excit"),
  terminalConfig: terminalConfigZ,
  minVal: z.number(),
  maxVal: z.number(),
  units: z.enum(["V", "mV"]),
  bridgeConfig: z.enum(["full", "half", "quarter", "none"]),
  voltageExcitSource: excitSourceZ,
  voltageExcitVal: z.number(),
  useExcitForScaling: z.boolean(),
  customScale: scaleZ,
});

export type AIVoltageChanWithExcit = z.infer<typeof aiVoltageChanWithExcitZ>;

export const ZERO_AI_VOLTAGE_CHAN_WITH_EXCIT: AIVoltageChanWithExcit = {
  name: "",
  key: "",
  device: "",
  type: "ai_voltage_with_excit",
  channel: 0,
  port: 0,
  enabled: true,
  terminalConfig: "Cfg_Default",
  minVal: 0,
  maxVal: 1,
  units: "V",
  bridgeConfig: "full",
  voltageExcitSource: "Internal",
  voltageExcitVal: 0,
  useExcitForScaling: false,
  customScale: ZERO_NO_SCALE,
};

export const aiChan = z.union([
  aiAccelChanZ,
  aiBridgeChanZ,
  aiCurrentChanZ,
  aiForceBridgeTableChanZ,
  aiForceBridgeTwoPointLinChan,
  aiForgeIEPEChanZ,
  aiMicrophoneChanZ,
  aiPressureBridgeTableChanZ,
  aiPressureBridgeTwoPointLinChanZ,
  aiResistanceChanZ,
  aiRTDChanZ,
  aiStrainGageChan,
  aiTempBuiltInChanZ,
  aiThermocoupleChanZ,
  aiTorqueBridgeTableChanZ,
  aiTorqueBridgeTwoPointLinChanZ,
  aiVelocityIEPEChanZ,
  aiVoltageChanZ,
]);

export type AIChan = z.infer<typeof aiChan>;
export type AIChanType = AIChan["type"];

export const AI_CHANNEL_SCHEMAS: Record<AIChanType, z.ZodType<AIChan>> = {
  ai_accel: aiAccelChanZ,
  ai_bridge: aiBridgeChanZ,
  ai_current: aiCurrentChanZ,
  ai_force_bridge_table: aiForceBridgeTableChanZ,
  ai_force_bridge_two_point_lin: aiForceBridgeTwoPointLinChan,
  ai_force_iepe: aiForgeIEPEChanZ,
  ai_microphone: aiMicrophoneChanZ,
  ai_pressure_bridge_table: aiPressureBridgeTableChanZ,
  ai_pressure_bridge_two_point_lin: aiPressureBridgeTwoPointLinChanZ,
  ai_resistance: aiResistanceChanZ,
  ai_rtd: aiRTDChanZ,
  ai_strain_gauge: aiStrainGageChan,
  ai_temp_builtin: aiTempBuiltInChanZ,
  ai_thermocouple: aiThermocoupleChanZ,
  ai_torque_bridge_table: aiTorqueBridgeTableChanZ,
  ai_torque_bridge_two_point_lin: aiTorqueBridgeTwoPointLinChanZ,
  ai_velocity_iepe: aiVelocityIEPEChanZ,
  ai_voltage: aiVoltageChanZ,
};

export const ZERO_AI_CHANNELS: Record<AIChanType, AIChan> = {
  ai_accel: ZERO_AI_ACCEL_CHAN,
  ai_bridge: ZERO_AI_BRIDGE_CHAN,
  ai_current: ZERO_AI_CURRENT_CHAN,
  ai_force_bridge_table: ZERO_AI_FORCE_BRIDGE_TABLE_CHAN,
  ai_force_bridge_two_point_lin: ZERO_AI_FORCE_BRIDGE_TWO_POINT_LIN_CHAN,
  ai_force_iepe: ZERO_AI_FORCE_IEPE_CHAN,
  ai_microphone: ZERO_AI_MICROPHONE_CHAN,
  ai_pressure_bridge_table: ZERO_AI_PRESSURE_BRIDGE_TABLE_CHAN,
  ai_pressure_bridge_two_point_lin: ZERO_AI_PRESSURE_BRIDGE_TWO_POINT_LIN_CHAN,
  ai_resistance: ZERO_AI_RESISTANCE_CHAN,
  ai_rtd: ZERO_AI_RTD_CHAN,
  ai_strain_gauge: ZERO_AI_STRAIN_GAGE_CHAN,
  ai_temp_builtin: ZERO_AI_TEMP_BUILTIN_CHAN,
  ai_thermocouple: ZERO_AI_THERMOCOUPLE_CHAN,
  ai_torque_bridge_table: ZERO_AI_TORQUE_BRIDGE_TABLE_CHAN,
  ai_torque_bridge_two_point_lin: ZERO_AI_TORQUE_BRIDGE_TWO_POINT_LIN_CHAN,
  ai_velocity_iepe: ZERO_AI_VELOCITY_EPE_CHAN,
  ai_voltage: ZERO_AI_VOLTAGE_CHAN,
};

export const AI_CHANNEL_TYPE_NAMES: Record<AIChanType, string> = {
  ai_accel: "Accelerometer",
  ai_bridge: "Bridge",
  ai_current: "Current",
  ai_force_bridge_table: "Force Bridge Table",
  ai_force_bridge_two_point_lin: "Force Bridge Two-Point Linear",
  ai_force_iepe: "Force IEPE",
  ai_microphone: "Microphone",
  ai_pressure_bridge_table: "Pressure Bridge Table",
  ai_pressure_bridge_two_point_lin: "Pressure Bridge Two-Point Linear",
  ai_resistance: "Resistance",
  ai_rtd: "RTD",
  ai_strain_gauge: "Strain Gauge",
  ai_temp_builtin: "Temperature Built-In Sensor",
  ai_thermocouple: "Thermocouple",
  ai_torque_bridge_table: "Torque Bridge Table",
  ai_torque_bridge_two_point_lin: "Torque Bridge Two-Point Linear",
  ai_velocity_iepe: "Velocity IEPE",
  ai_voltage: "Voltage",
};

export type AnalogInputVoltageChannel = z.infer<typeof aiVoltageChanZ>;

const doChanZ = z.object({
  key: z.string(),
  type: z.literal("digital_output"),
  enabled: z.boolean(),
  cmdChannel: z.number(),
  stateChannel: z.number(),
  port: z.number(),
  line: z.number(),
});

export const ZERO_DO_CHAN: DOChan = {
  key: "",
  type: "digital_output",
  enabled: true,
  cmdChannel: 0,
  stateChannel: 0,
  port: 0,
  line: 0,
};

export type DOChan = z.infer<typeof doChanZ>;
export type DOChanType = DOChan["type"];

const diChanZ = z.object({
  key: z.string(),
  type: z.literal("digital_input"),
  enabled: z.boolean(),
  port: z.number(),
  line: z.number(),
  channel: z.number(),
});
export const ZERO_DI_CHAN: DIChan = {
  key: "",
  type: "digital_input",
  enabled: true,
  port: 0,
  line: 0,
  channel: 0,
};

export type DIChan = z.infer<typeof diChanZ>;
export type DIChanType = DIChan["type"];

const deviceKeyZ = device.deviceKeyZ.min(1, "Must specify a device");

export const analogReadTaskConfigZ = z
  .object({
    version: z.literal("1.0.0"),
    sampleRate: z.number().min(0).max(50000),
    streamRate: z.number().min(0).max(50000),
    channels: z.array(aiChan),
    dataSaving: z.boolean(),
  })
  .refine(
    (c) =>
      // Ensure that the stream Rate is lower than the sample rate
      c.sampleRate >= c.streamRate,
    {
      path: ["streamRate"],
      message: "Stream rate must be less than or equal to the sample rate",
    },
  )
  .superRefine((cfg, ctx) => {
    const ports = new Map<string, number>();
    cfg.channels.forEach(({ port, device }) =>
      ports.set(`${device}/${port}`, (ports.get(`${device}/${port}`) ?? 0) + 1),
    );
    cfg.channels.forEach((channel, i) => {
      if ((ports.get(`${channel.device}/${channel.port}`) ?? 0) < 2) return;
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        path: ["channels", i, "port"],
        message: `Port ${channel.port} has already been used on device`,
      });
    });
  })
  .superRefine((cfg, ctx) => {
    const channels = new Map<number, number>();
    cfg.channels.forEach(({ channel }) => {
      if (channel === 0 || channel == null) return;
      channels.set(channel, (channels.get(channel) ?? 0) + 1);
    });
    cfg.channels.forEach((cfg, i) => {
      if (cfg.channel === 0 || cfg.channel == null) return;
      if ((channels.get(cfg.channel) ?? 0) < 2) return;
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        path: ["channels", i, "channel"],
        message: `Channel has already been used on port ${cfg.port}`,
      });
    });
  });

export type AnalogReadTaskConfig = z.infer<typeof analogReadTaskConfigZ>;

export const baseAnalogReadStateDetailsZ = z.object({
  running: z.boolean(),
  message: z.string(),
});

export const errorAnalogReadStateDetailZ = baseAnalogReadStateDetailsZ.extend({
  errors: z.array(
    z.object({
      message: z.string(),
      path: z.string(),
    }),
  ),
});

type BaseAnalogReadStateDetails = z.infer<typeof baseAnalogReadStateDetailsZ>;
type ErrorAnalogReadStateDetails = z.infer<typeof errorAnalogReadStateDetailZ>;

export type AnalogReadStateDetails =
  | BaseAnalogReadStateDetails
  | ErrorAnalogReadStateDetails;
export type AnalogReadTaskState = task.State<AnalogReadStateDetails>;

export const ANALOG_READ_TYPE = "ni_analog_read";
export type AnalogReadType = typeof ANALOG_READ_TYPE;

export const ZERO_ANALOG_READ_CONFIG: AnalogReadTaskConfig = {
  version: "1.0.0",
  sampleRate: 10,
  streamRate: 5,
  channels: [],
  dataSaving: true,
};
export type AnalogRead = task.Task<
  AnalogReadTaskConfig,
  AnalogReadStateDetails,
  AnalogReadType
>;
export type AnalogReadPayload = task.Payload<
  AnalogReadTaskConfig,
  AnalogReadStateDetails,
  AnalogReadType
>;
export const ZERO_ANALOG_READ_PAYLOAD: AnalogReadPayload = {
  key: "",
  name: "NI Analog Read Task",
  config: ZERO_ANALOG_READ_CONFIG,
  type: ANALOG_READ_TYPE,
};

export type DigitalWriteConfig = z.infer<typeof digitalWriteConfigZ>;
export const DIGITAL_WRITE_TYPE = "ni_digital_write";
export type DigitalWriteType = typeof DIGITAL_WRITE_TYPE;
export const digitalWriteConfigZ = z.object({
  device: deviceKeyZ,
  channels: z.array(doChanZ),
  stateRate: z.number().min(0).max(50000),
  dataSaving: z.boolean(),
});

export const digitalWriteStateDetailsZ = z.object({
  running: z.boolean(),
});
export type DigitalWriteStateDetails = z.infer<typeof digitalWriteStateDetailsZ>;
export type DigitalWrite = task.Task<
  DigitalWriteConfig,
  DigitalWriteStateDetails,
  DigitalWriteType
>;
export type DigitalWritePayload = task.Payload<
  DigitalWriteConfig,
  DigitalWriteStateDetails,
  DigitalWriteType
>;
export const ZERO_DIGITAL_WRITE_CONFIG: DigitalWriteConfig = {
  device: "",
  stateRate: 10,
  channels: [],
  dataSaving: true,
};
export const ZERO_DIGITAL_WRITE_PAYLOAD: DigitalWritePayload = {
  key: "",
  name: "NI Digital Write Task",
  config: ZERO_DIGITAL_WRITE_CONFIG,
  type: DIGITAL_WRITE_TYPE,
};

const digitalReadChannelZ = diChanZ;
export const digitalReadConfigZ = z.object({
  device: deviceKeyZ,
  sampleRate: z.number().min(0).max(50000),
  streamRate: z.number().min(0).max(50000),
  dataSaving: z.boolean(),
  channels: z.array(digitalReadChannelZ),
});
export type DigitalReadConfig = z.infer<typeof digitalReadConfigZ>;
export const DIGITAL_READ_TYPE = "ni_digital_read";
export type DigitalReadType = typeof DIGITAL_READ_TYPE;
export const digitalReadStateDetailsZ = z.object({
  running: z.boolean(),
});
export type DigitalReadStateDetails = z.infer<typeof digitalReadStateDetailsZ>;
export type DigitalRead = task.Task<
  DigitalReadConfig,
  DigitalReadStateDetails,
  DigitalReadType
>;
export type DigitalReadPayload = task.Payload<
  DigitalReadConfig,
  DigitalReadStateDetails,
  DigitalReadType
>;
export const ZERO_DIGITAL_READ_CONFIG: DigitalReadConfig = {
  device: "",
  channels: [],
  sampleRate: 50,
  streamRate: 25,
  dataSaving: true,
};
export const ZERO_DIGITAL_READ_PAYLOAD: DigitalReadPayload = {
  key: "",
  name: "NI Digital Read Task",
  config: ZERO_DIGITAL_READ_CONFIG,
  type: DIGITAL_READ_TYPE,
};

export const scanConfigZ = z.object({
  enabled: z.boolean().optional().default(true),
});
export type ScanConfig = z.infer<typeof scanConfigZ>;
export const SCAN_TYPE = "ni_scanner";
export type ScanType = typeof SCAN_TYPE;
export type Scan = task.Task<ScanConfig, task.State, ScanType>;

export type Task = AnalogRead | DigitalWrite | DigitalRead;
export type Chan = DIChan | AIChan | DOChan;

export const analogReadTaskMigration = migrate.createMigration<
  v0.AnalogReadTaskConfig,
  AnalogReadTaskConfig
>({
  name: "hardware.ni.task.analogRead",
  migrate: (s) => ({
    ...s,
    version: "1.0.0",
    device: undefined,
    channels: s.channels.map((c) => ({
      device: s.device,
      ...c,
    })),
  }),
});
