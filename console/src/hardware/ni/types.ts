// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { current } from "@reduxjs/toolkit";
import { type task } from "@synnaxlabs/client";
import { xy } from "@synnaxlabs/x";
import { clientXY } from "node_modules/@synnaxlabs/x/dist/src/spatial/base";
import { z } from "zod";

const linearScaleTypeZ = z.enum(["linear", "none"]);

export type LinearScaleType = z.infer<typeof linearScaleTypeZ>;

const linearScaleZ = z.object({
  type: linearScaleTypeZ,
  one: xy.xy,
  two: xy.xy,
});

export type LinearScale = z.infer<typeof linearScaleZ>;

const DEFAULT_LINEAR_SCALE: LinearScale = {
  type: "none",
  one: { x: 0, y: 0 },
  two: { x: 0, y: 0 },
};

export const DEFAULT_SCALES: Record<LinearScaleType, LinearScale> = {
  linear: DEFAULT_LINEAR_SCALE,
  none: DEFAULT_LINEAR_SCALE,
};

const terminalConfigZ = z.enum(["Cfg_Default", "RSE", "NRSE", "PseudoDiff"]);

export type TerminalConfig = z.infer<typeof terminalConfigZ>;

const excitSourceZ = z.enum(["Internal", "External", "None"]);

export type ExcitationSource = z.infer<typeof excitSourceZ>;

const baseAIChanZ = z.object({
  key: z.string(),
  channel: z.number(),
  port: z.number(),
  enabled: z.boolean(),
});

const minMaxValZ = z
  .object({
    minVal: z.number(),
    maxVal: z.number(),
  })

export const sensitivityUnitsZ = z.enum(["mVoltsPerG", "VoltsPerG"]);

export type AccelSensitivityUnits = z.infer<typeof sensitivityUnitsZ>;

export const accelerationUnitsZ = z.enum([
  "g",
  "MetersPerSecondSquared",
  "InchesPerSecondSquared",
]);

export type AccelerationUnits = z.infer<typeof accelerationUnitsZ>;

const baseAiAccelChanZ = 
  baseAIChanZ.merge(minMaxValZ).extend({
    terminalConfig: terminalConfigZ,
    sensitivity: z.number(),
    sensitivityUnits: sensitivityUnitsZ,
  })


// 1 - https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaiaccelchan.html
export const aiAccelChanZ = 
  baseAiAccelChanZ.extend({
    type: z.literal("ai_accel"),
    currentExcitSource: excitSourceZ,
    currentExcitVal: z.number(),
  });

export type AIAccelChan = z.infer<typeof aiAccelChanZ>;

export const ZERO_AI_ACCEL_CHAN: AIAccelChan = {
  key: "",
  type: "ai_accel",
  channel: 0,
  port: 0,
  enabled: false,
  minVal: 0,
  maxVal: 0,
  terminalConfig: "Cfg_Default",
  sensitivity: 0,
  sensitivityUnits: "mVoltsPerG",
  currentExcitSource: "None",
  currentExcitVal: 0,
};

// 2 - https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaiaccel4wiredcvoltagechan.html
const aiAccel4WireDCVoltageChanZ = 
  baseAiAccelChanZ.extend({
    type: z.literal("ai_accel_4_wire_dc_voltage"),
    units: accelerationUnitsZ,
    voltageExcitSource: excitSourceZ,
    voltageExcitVal: z.number(),
    useExcitForScaling: z.boolean(),
  });

export type AIAccel4WireDCVoltageChan = z.infer<typeof aiAccel4WireDCVoltageChanZ>;

export const ZERO_AI_ACCEL_4WIRE_DC_VOLTAGE_CHAN: AIAccel4WireDCVoltageChan = {
  key: "",
  type: "ai_accel_4_wire_dc_voltage",
  units: "g",
  channel: 0,
  port: 0,
  enabled: false,
  minVal: 0,
  maxVal: 0,
  terminalConfig: "Cfg_Default",
  sensitivity: 0,
  sensitivityUnits: "mVoltsPerG",
  voltageExcitSource: "None",
  voltageExcitVal: 0,
  useExcitForScaling: false,
};

// 3 - https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaiaccelchargechan.html
const aiAccelChargeChanZ = baseAiAccelChanZ.extend({
    type: z.literal("ai_accel_charge"),
    units: accelerationUnitsZ,
  });

export type AIAccelChargeChan = z.infer<typeof aiAccelChargeChanZ>;

export const ZERO_AI_ACCEL_CHARGE_CHAN: AIAccelChargeChan = {
  key: "",
  type: "ai_accel_charge",
  channel: 0,
  port: 0,
  enabled: false,
  minVal: 0,
  maxVal: 0,
  terminalConfig: "Cfg_Default",
  sensitivity: 0,
  sensitivityUnits: "mVoltsPerG",
  units: "g",
};

export const bridgeConfigZ = z.enum(["FullBridge", "HalfBridge", "QuarterBridge"]);
export type BridgeConfig = z.infer<typeof bridgeConfigZ>;

// 4 - https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaibridgechan.html
const aiBridgeChanZ = baseAIChanZ.extend({
    type: z.literal("ai_bridge"),
    terminalConfig: terminalConfigZ,
    minVal: z.number(),
    maxVal: z.number(),
    bridgeConfig: bridgeConfigZ,
    voltageExcitSource: excitSourceZ,
    voltageExcitVal: z.number(),
    nominalBridgeResistance: z.number(),
});

export type AIBridgeChan = z.infer<typeof aiBridgeChanZ>;

export const ZERO_AI_BRIDGE_CHAN: AIBridgeChan = {
  key: "",
  type: "ai_bridge",
  channel: 0,
  port: 0,
  enabled: false,
  minVal: 0,
  maxVal: 0,
  terminalConfig: "Cfg_Default",
  bridgeConfig: "FullBridge",
  voltageExcitSource: "None",
  voltageExcitVal: 0,
  nominalBridgeResistance: 0,
};

// 5 - https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaichargechan.html
const aiChargeChan = baseAIChanZ.extend({
  type: z.literal("ai_charge"),
  terminalConfig: terminalConfigZ,
  minVal: z.number(),
  maxVal: z.number(),
  units: z.enum(["C", "uC"]),
});

export type AIChargeChan = z.infer<typeof aiChargeChan>;

export const ZERO_AI_CHARGE_CHAN: AIChargeChan = {
  key: "",
  channel: 0,
  type: "ai_charge",
  port: 0,
  enabled: false,
  minVal: 0,
  maxVal: 0,
  terminalConfig: "Cfg_Default",
  units: "C",
};

// 6 - https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaicurrentchan.html
const aiCurrentChanZ = baseAIChanZ.extend({
  type: z.literal("ai_current"),
  terminalConfig: terminalConfigZ,
  minVal: z.number(),
  maxVal: z.number(),
  units: z.enum(["A", "mA"]),
  shuntResistorLoc: z.enum(["default", "internal", "external"]),
  extShuntResistorVal: z.number(),
});

export type AICurrentChan = z.infer<typeof aiCurrentChanZ>;

export const ZERO_AI_CURRENT_CHAN: AICurrentChan = {
  key: "",
  channel: 0,
  port: 0,
  type: "ai_current",
  enabled: false,
  minVal: 0,
  maxVal: 0,
  terminalConfig: "Cfg_Default",
  units: "A",
  shuntResistorLoc: "default",
  extShuntResistorVal: 0,
};

// 7 - https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaicurrentrmschan.html
const aiCurrentRMSChanZ = baseAIChanZ.extend({
    type: z.literal("ai_current_rms"),
    terminalConfig: terminalConfigZ,
    minVal: z.number(),
    maxVal: z.number(),
    units: z.enum(["A", "mA"]),
    shuntResistorLoc: z.enum(["default", "internal", "external"]),
    extShuntResistorVal: z.number(),
  });

export type AICurrentRMSChan = z.infer<typeof aiCurrentRMSChanZ>;

export const ZERO_AI_CURRENT_RMS_CHAN: AICurrentRMSChan = {
  key: "",
  channel: 0,
  type: "ai_current_rms",
  port: 0,
  enabled: false,
  minVal: 0,
  maxVal: 0,
  terminalConfig: "Cfg_Default",
  units: "A",
  shuntResistorLoc: "default",
  extShuntResistorVal: 0,
};

export const forceUnitsZ = z.enum(["Newtons", "Pounds", "KilogramForce"]);
export type ForceUnits = z.infer<typeof forceUnitsZ>;

export const electricalUnitsZ = z.enum(["mVoltsPerVolt", "VoltsPerVolt"]);
export type ElectricalUnits = z.infer<typeof electricalUnitsZ>;

// 8 - https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaiforcebridgepolynomialchan.html
const aiForceBridgePolynomialChanZ = baseAIChanZ.extend({
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
});

export type AIForceBridgePolynomialChan = z.infer<typeof aiForceBridgePolynomialChanZ>;

export const ZERO_AI_FORCE_BRIDGE_POLYNOMIAL_CHAN: AIForceBridgePolynomialChan = {
  key: "",
  type: "ai_force_bridge_polynomial",
  channel: 0,
  port: 0,
  enabled: false,
  minVal: 0,
  maxVal: 0,
  terminalConfig: "Cfg_Default",
  units: "Newtons",
  bridgeConfig: "FullBridge",
  voltageExcitSource: "None",
  voltageExcitVal: 0,
  nominalBridgeResistance: 0,
  forwardCoeffs: [],
  reverseCoeffs: [],
  electricalUnits: "mVoltsPerVolt",
  physicalUnits: "Newtons",
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
});

export type AIForceBridgeTableChan = z.infer<typeof aiForceBridgeTableChanZ>;

export const ZERO_AI_FORCE_BRIDGE_TABLE_CHAN: AIForceBridgeTableChan = {
  key: "",
  type: "ai_force_bridge_table",
  channel: 0,
  port: 0,
  enabled: false,
  minVal: 0,
  maxVal: 0,
  units: "Newtons",
  bridgeConfig: "FullBridge",
  voltageExcitSource: "None",
  voltageExcitVal: 0,
  nominalBridgeResistance: 0,
  electricalUnits: "mVoltsPerVolt",
  electricalVals: [],
  physicalUnits: "Newtons",
  physicalVals: [],
};

// 10 - https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaiforcebridgetwopointlinchan.html
const aiForceBridgeTwoPointLinChan = baseAIChanZ.extend({
  type: z.literal("ai_force_bridge_two_point_lin"),
  terminalConfig: terminalConfigZ,
  minVal: z.number(),
  maxVal: z.number(),
  units: forceUnitsZ,
  bridgeConfig: z.enum(["full", "half", "quarter"]),
  voltageExcitSource: excitSourceZ,
  voltageExcitVal: z.number(),
  nominalBridgeResistance: z.number(),
  electricalUnits: electricalUnitsZ,
  physicalUnits: forceUnitsZ,
  firstElectricalVal: z.number(),
  firstPhysicalVal: z.number(),
  secondElectricalVal: z.number(),
  secondPhysicalVal: z.number(),
});

export type AIForceBridgeTwoPointLinChan = z.infer<typeof aiForceBridgeTwoPointLinChan>;

export const ZERO_AI_FORCE_BRIDGE_TWO_POINT_LIN_CHAN: AIForceBridgeTwoPointLinChan = {
  key: "",
  type: "ai_force_bridge_two_point_lin",
  channel: 0,
  port: 0,
  enabled: false,
  minVal: 0,
  maxVal: 0,
  terminalConfig: "Cfg_Default",
  units: "Newtons",
  bridgeConfig: "full",
  voltageExcitSource: "None",
  voltageExcitVal: 0,
  nominalBridgeResistance: 0,
  electricalUnits: "mVoltsPerVolt",
  physicalUnits: "Newtons",
  firstElectricalVal: 0,
  firstPhysicalVal: 0,
  secondElectricalVal: 0,
  secondPhysicalVal: 0,
};

// 11 - https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaiforceiepechan.html
const aiForceEPEChanZ = baseAIChanZ.extend({
  type: z.literal("ai_force_epe"),
  terminalConfig: terminalConfigZ,
  minVal: z.number(),
  maxVal: z.number(),
  units: forceUnitsZ,
  sensitivity: z.number(),
  sensitivityUnits: electricalUnitsZ,
  currExcitSource: excitSourceZ,
  currentExcitVal: z.number(),
});

export type AIForceEPEChan = z.infer<typeof aiForceEPEChanZ>;

export const ZERO_AI_FORCE_EPE_CHAN: AIForceEPEChan = {
  key: "",
  type: "ai_force_epe",
  channel: 0,
  port: 0,
  enabled: false,
  minVal: 0,
  maxVal: 0,
  terminalConfig: "Cfg_Default",
  units: "Newtons",
  sensitivity: 0,
  sensitivityUnits: "mVoltsPerVolt",
  currExcitSource: "None",
  currentExcitVal: 0,
};


// 12 - https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaifreqvoltagechan.html
const aiFreqVoltageChanZ = baseAIChanZ.extend({ 
  type: z.literal("ai_freq_voltage"),
  minVal: z.number(),
  maxVal: z.number(),
  units: z.enum(["Hz"]),
  thresholdLevel: z.number(),
  hysteresis: z.number(),
});

export type AIFreqVoltageChan = z.infer<typeof aiFreqVoltageChanZ>;

export const ZERO_AI_FREQ_VOLTAGE_CHAN: AIFreqVoltageChan = {
  key: "",
  type: "ai_freq_voltage",
  channel: 0,
  port: 0,
  enabled: false,
  minVal: 0,
  maxVal: 0,
  units: "Hz",
  thresholdLevel: 0,
  hysteresis: 0,
};

// 13 - https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaimicrophonechan.html
const aiMicrophoneChanZ = baseAIChanZ.extend({
  type: z.literal("ai_microphone"),
  terminalConfig: terminalConfigZ,
  micSensitivity: z.number(),
  maxSndPressLevel: z.number(),
  currentExcitSource: excitSourceZ,
  currentExcitVal: z.number(),
});

export type AIMicrophoneChan = z.infer<typeof aiMicrophoneChanZ>;

export const ZERO_AI_MICROPHONE_CHAN: AIMicrophoneChan = {
  key: "",
  type: "ai_microphone",
  channel: 0,
  port: 0,
  enabled: false,
  terminalConfig: "Cfg_Default",
  micSensitivity: 0,
  maxSndPressLevel: 0,
  currentExcitSource: "None",
  currentExcitVal: 0,
};

export const pressureUnitsZ = z.enum(["psi", "Pa", "bar"]);
export type PressureUnits = z.infer<typeof pressureUnitsZ>;

// 14 - https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaipressurebridgepolynomialchan.html
const aiPressureBridgePolynomialChanZ = baseAIChanZ.extend({
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
});

export type AIPressureBridgePolynomialChan = z.infer<typeof aiPressureBridgePolynomialChanZ>;

export const ZERO_AI_PRESSURE_BRIDGE_POLYNOMIAL_CHAN: AIPressureBridgePolynomialChan = {
  key: "",
  type: "ai_pressure_bridge_polynomial",
  channel: 0,
  port: 0,
  enabled: false,
  minVal: 0,
  maxVal: 0,
  units: "psi",
  bridgeConfig: "FullBridge",
  voltageExcitSource: "None",
  voltageExcitVal: 0,
  nominalBridgeResistance: 0,
  forwardCoeffs: [],
  reverseCoeffs: [],
  electricalUnits: "mVoltsPerVolt",
  physicalUnits: "psi",
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
});

export type AIPressureBridgeTableChan = z.infer<typeof aiPressureBridgeTableChanZ>;

export const ZERO_AI_PRESSURE_BRIDGE_TABLE_CHAN: AIPressureBridgeTableChan = {
  key: "",
  type: "ai_pressure_bridge_table",
  channel: 0,
  port: 0,
  enabled: false,
  minVal: 0,
  maxVal: 0,
  units: "psi",
  bridgeConfig: "FullBridge",
  voltageExcitSource: "None",
  voltageExcitVal: 0,
  nominalBridgeResistance: 0,
  electricalUnits: "mV/V",
  electricalVals: [],
  physicalUnits: "psi",
  physicalVals: [],
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
});

export type AIPressureBridgeTwoPointLinChan = z.infer<typeof aiPressureBridgeTwoPointLinChanZ>;

export const ZERO_AI_PRESSURE_BRIDGE_TWO_POINT_LIN_CHAN: AIPressureBridgeTwoPointLinChan = {
  key: "",
  type: "ai_pressure_bridge_two_point_lin",
  channel: 0,
  port: 0,
  enabled: false,
  minVal: 0,
  maxVal: 0,
  terminalConfig: "Cfg_Default",
  units: "psi",
  bridgeConfig: "FullBridge",
  voltageExcitSource: "None",
  voltageExcitVal: 0,
  nominalBridgeResistance: 0,
  electricalUnits: "mVoltsPerVolt",
  physicalUnits: "psi",
  firstElectricalVal: 0,
  firstPhysicalVal: 0,
  secondElectricalVal: 0,
  secondPhysicalVal: 0,
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
});

export type AIResistanceChan = z.infer<typeof aiResistanceChanZ>;

export const ZERO_AI_RESISTANCE_CHAN: AIResistanceChan = {
  key: "",
  type: "ai_resistance",
  channel: 0,
  port: 0,
  enabled: false,
  minVal: 0,
  maxVal: 0,
  terminalConfig: "Cfg_Default",
  units: "Ohms",
  resistanceConfig: "2Wire",
  currentExcitSource: "None",
  currentExcitVal: 0,
};

// 18 -  https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateairosettestraingagechan.html
const aiRosetteStrainGageChanZ = baseAIChanZ.extend({
  type: z.literal("ai_rosette_strain_gage"),
  terminalConfig: terminalConfigZ,
  minVal: z.number(),
  maxVal: z.number(),
  rosetteType: z.enum(["rectangular", "delta", "tee"]),
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
  units: z.enum(["strain"]),
  gageConfig: z.enum(["120", "45", "60"]),
  voltageExcitSource: excitSourceZ,
  voltageExcitVal: z.number(),
  nominalGageResistance: z.number(),
});

export type AIRosetteStrainGageChan = z.infer<typeof aiRosetteStrainGageChanZ>;

export const ZERO_AI_ROSETTE_STRAIN_GAGE_CHAN: AIRosetteStrainGageChan = {
  key: "",
  type: "ai_rosette_strain_gage",
  channel: 0,
  port: 0,
  enabled: false,
  minVal: 0,
  maxVal: 0,
  terminalConfig: "Cfg_Default",
  rosetteType: "rectangular",
  gageOrientation: 0,
  rosetteMeasTypes: [],
  units: "strain",
  gageConfig: "120",
  voltageExcitSource: "None",
  voltageExcitVal: 0,
  nominalGageResistance: 0,
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
    "pt3750",
    "pt3851",
    "pt3911",
    "pt3916",
    "pt3920",
    "pt3928",
    "pt3850",
  ]),
  resistanceConfig: resistanceConfigZ,
  currentExcitSource: excitSourceZ,
  currentExcitVal: z.number(),
});

export type AIRTDChan = z.infer<typeof aiRTDChanZ>;

export const ZERO_AI_RTD_CHAN: AIRTDChan = {
  key: "",
  channel: 0,
  type: "ai_rtd",
  port: 0,
  enabled: false,
  minVal: 0,
  maxVal: 0,
  units: "DegC",
  rtdType: "pt3750",
  resistanceConfig: "2Wire",
  currentExcitSource: "None",
  currentExcitVal: 0,
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
});

export type AIStrainGageChan = z.infer<typeof aiStrainGageChan>;

export const ZERO_AI_STRAIN_GAGE_CHAN: AIStrainGageChan = {
  key: "",
  type: "ai_strain_gauge",
  channel: 0,
  port: 0,
  enabled: false,
  minVal: 0,
  maxVal: 0,
  terminalConfig: "Cfg_Default",
  units: "strain",
  strainConfig: "full-bridge-I",
  voltageExcitSource: "None",
  voltageExcitVal: 0,
  gageFactor: 0,
  initialBridgeVoltage: 0,
  nominalGageResistance: 0,
  poissonRatio: 0,
  leadWireResistance: 0,
};

// 21 - https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaitempbuiltinsensorchan.html
const aiTempBuiltInChanZ = baseAIChanZ.extend({
  type: z.literal("ai_temp_builtin"),
  units: temperatureUnitsZ,
});

export type AITempBuiltInChan = z.infer<typeof aiTempBuiltInChanZ>;

export const ZERO_ai_temp_builtin_CHAN: AITempBuiltInChan = {
  key: "",
  type: "ai_temp_builtin",
  channel: 0,
  port: 0,
  enabled: false,
  units: "DegC",
};

// 22 - https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaithrmcplchan.html
const aiThermocoupleChanZ = baseAIChanZ.extend({
  key: z.string(),
  type: z.literal("ai_thermocouple"),
  minVal: z.number(),
  maxVal: z.number(),
  units: temperatureUnitsZ,
  thermocoupleType: z.enum(["J", "K", "N", "R", "S", "T", "B", "E"]),
  cjsSource: z.enum(["BuiltIn", "ConstVal"]),
  cjcVal: z.number(),
  cjcPort: z.number(),
});

export type AIThermocoupleChan = z.infer<typeof aiThermocoupleChanZ>;

export const ZERO_AI_THERMOCOUPLE_CHAN: AIThermocoupleChan = {
  key: "",
  type: "ai_thermocouple",
  channel: 0,
  port: 0,
  enabled: false,
  minVal: 0,
  maxVal: 0,
  units: "DegC",
  thermocoupleType: "J",
  cjsSource: "BuiltIn",
  cjcVal: 0,
  cjcPort: 0,
};

// 23 - https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaithrmstrchaniex.html
const aiThermistorChanIex = baseAIChanZ.extend({
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
  key: "",
  type: "ai_thermistor_iex",
  channel: 0,
  port: 0,
  enabled: false,
  minVal: 0,
  maxVal: 0,
  units: "DegC",
  resistanceConfig: "2Wire",
  currentExcitSource: "None",
  currentExcitVal: 0,
  a: 0,
  b: 0,
  c: 0,
};

// 24 - https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaitorquebridgepolynomialchan.html
const aiThermistorChanVex = baseAIChanZ.extend({
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
  key: "",
  channel: 0,
  type: "ai_thermistor_vex",
  port: 0,
  enabled: false,
  minVal: 0,
  maxVal: 0,
  units: "DegC",
  resistanceConfig: "2Wire",
  voltageExcitSource: "None",
  voltageExcitVal: 0,
  a: 0,
  b: 0,
  c: 0,
  r1: 0,
};

export const torqueUnitsZ = z.enum(["NewtonMeters", "InchOunces", "FootPounds"]);
export type TorqueUnits = z.infer<typeof torqueUnitsZ>;

// 25 - https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaitorquebridgepolynomialchan.html
const aiTorqueBridgePolynomialChanZ = baseAIChanZ.extend({
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
});

export type AITorqueBridgePolynomialChan = z.infer<typeof aiTorqueBridgePolynomialChanZ>;

export const ZERO_AI_TORQUE_BRIDGE_POLYNOMIAL_CHAN: AITorqueBridgePolynomialChan = {
  key: "",
  type: "ai_torque_bridge_polynomial",
  channel: 0,
  port: 0,
  enabled: false,
  minVal: 0,
  maxVal: 0,
  units: "NewtonMeters",
  bridgeConfig: "FullBridge",
  voltageExcitSource: "None",
  voltageExcitVal: 0,
  nominalBridgeResistance: 0,
  forwardCoeffs: [],
  reverseCoeffs: [],
  electricalUnits: "mVoltsPerVolt",
  physicalUnits: "NewtonMeters",
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
});

export type AITorqueBridgeTableChan = z.infer<typeof aiTorqueBridgeTableChanZ>;

export const ZERO_AI_TORQUE_BRIDGE_TABLE_CHAN: AITorqueBridgeTableChan = {
  key: "",
  type: "ai_torque_bridge_table",
  channel: 0,
  port: 0,
  enabled: false,
  minVal: 0,
  maxVal: 0,
  units: "NewtonMeters",
  bridgeConfig: "FullBridge",
  voltageExcitSource: "None",
  voltageExcitVal: 0,
  nominalBridgeResistance: 0,
  electricalUnits: "mVoltsPerVolt",
  electricalVals: [],
  physicalUnits: "NewtonMeters",
  physicalVals: [],
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
});

export type AITorqueBridgeTwoPointLinChan = z.infer<typeof aiTorqueBridgeTwoPointLinChanZ>;

export const ZERO_AI_TORQUE_BRIDGE_TWO_POINT_LIN_CHAN: AITorqueBridgeTwoPointLinChan = {
  key: "",
  type: "ai_torque_bridge_two_point_lin",
  channel: 0,
  port: 0,
  enabled: false,
  minVal: 0,
  maxVal: 0,
  units: "NewtonMeters",
  bridgeConfig: "FullBridge",
  voltageExcitSource: "None",
  voltageExcitVal: 0,
  nominalBridgeResistance: 0,
  electricalUnits: "mVoltsPerVolt",
  physicalUnits: "NewtonMeters",
  firstElectricalVal: 0,
  firstPhysicalVal: 0,
  secondElectricalVal: 0,
  secondPhysicalVal: 0,
};

export const velocityUnitsZ = z.enum(["m/s", "in/s"]);
export type VelocityUnits = z.infer<typeof velocityUnitsZ>;

// 28 - https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaivelocityiepechan.html
const aiVelocityEPEChanZ = baseAIChanZ.extend({
  key: z.string(),
  type: z.literal("ai_velocity_epe"),
  terminalConfig: terminalConfigZ,
  minVal: z.number(),
  maxVal: z.number(),
  units: velocityUnitsZ,
  sensitivity: z.number(),
  sensitivityUnits: z.enum(["mV/m/s", "V/m/s"]),
  currentExcitSource: excitSourceZ,
  currentExcitVal: z.number(),
});

export type AIVelocityEPEChan = z.infer<typeof aiVelocityEPEChanZ>;

export const ZERO_AI_VELOCITY_EPE_CHAN: AIVelocityEPEChan = {
  key: "",
  type: "ai_velocity_epe",
  channel: 0,
  port: 0,
  enabled: false,
  terminalConfig: "Cfg_Default",
  minVal: 0,
  maxVal: 0,
  units: "m/s",
  sensitivity: 0,
  sensitivityUnits: "mV/m/s",
  currentExcitSource: "None",
  currentExcitVal: 0,
};

const analogInputScaleZ = linearScaleZ;

// 29 - https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaivoltagechan.html
const aiVoltageChanZ = baseAIChanZ.extend({
  key: z.string(),
  type: z.literal("ai_voltage"),
  port: z.number(),
  channel: z.number(),
  enabled: z.boolean(),
  minVal: z.number(),
  maxVal: z.number(),
  units: z.enum(["Volts"]),
});

export type AIVoltageChan = z.infer<typeof aiVoltageChanZ>;

export const ZERO_AI_VOLTAGE_CHAN: AIVoltageChan = {
  key: "",
  type: "ai_voltage",
  channel: 0,
  port: 0,
  enabled: false,
  minVal: 0,
  maxVal: 0,
  units: "Volts",
};

// 30 - https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaivoltagermschan.html
const aiVoltageRMSChanZ = baseAIChanZ.extend({
  key: z.string(),
  type: z.literal("ai_voltage_rms"),
  terminalConfig: terminalConfigZ,
  minVal: z.number(),
  maxVal: z.number(),
  units: z.enum(["V", "mV"]),
});

export type AIVoltageRMSChan = z.infer<typeof aiVoltageRMSChanZ>;

export const ZERO_AI_VOLTAGE_RMS_CHAN: AIVoltageRMSChan = {
  key: "",
  type: "ai_voltage_rms",
  channel: 0,
  port: 0,
  enabled: false,
  terminalConfig: "Cfg_Default",
  minVal: 0,
  maxVal: 0,
  units: "V",
};

// 31 - https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaivoltagechanwithexcit.html
const aiVoltageChanWithExcitZ = baseAIChanZ.extend({
  type: z.literal("ai_voltage_with_excit"),
  terminalConfig: terminalConfigZ,
  minVal: z.number(),
  maxVal: z.number(),
  units: z.enum(["V", "mV"]),
  bridgeConfig: z.enum(["full", "half", "quarter", "none"]),
  voltageExcitSource: excitSourceZ,
  voltageExcitVal: z.number(),
  useExcitForScaling: z.boolean(),
});

export type AIVoltageChanWithExcit = z.infer<typeof aiVoltageChanWithExcitZ>;

export const ZERO_AI_VOLTAGE_CHAN_WITH_EXCIT: AIVoltageChanWithExcit = {
  key: "",
  type: "ai_voltage_with_excit",
  channel: 0,
  port: 0,
  enabled: false,
  terminalConfig: "Cfg_Default",
  minVal: 0,
  maxVal: 0,
  units: "V",
  bridgeConfig: "full",
  voltageExcitSource: "None",
  voltageExcitVal: 0,
  useExcitForScaling: false,
};

export const aiChan = z.union([
  aiAccelChanZ,
  aiAccel4WireDCVoltageChanZ,
  aiAccelChargeChanZ,
  aiBridgeChanZ,
  aiChargeChan,
  aiCurrentChanZ,
  aiCurrentRMSChanZ,
  aiForceBridgePolynomialChanZ,
  aiForceBridgeTableChanZ,
  aiForceBridgeTwoPointLinChan,
  aiForceEPEChanZ,
  aiFreqVoltageChanZ,
  aiMicrophoneChanZ,
  aiPressureBridgePolynomialChanZ,
  aiPressureBridgeTableChanZ,
  aiPressureBridgeTwoPointLinChanZ,
  aiResistanceChanZ,
  aiRosetteStrainGageChanZ,
  aiRTDChanZ,
  aiStrainGageChan,
  aiTempBuiltInChanZ,
  aiThermocoupleChanZ,
  aiThermistorChanIex,
  aiThermistorChanVex,
  aiTorqueBridgePolynomialChanZ,
  aiTorqueBridgeTableChanZ,
  aiTorqueBridgeTwoPointLinChanZ,
  aiVelocityEPEChanZ,
  aiVoltageChanZ,
  aiVoltageRMSChanZ,
  aiVoltageChanWithExcitZ,
]);

export type AIChan = z.infer<typeof aiChan>;
export type AIChanType = AIChan["type"];

export const ZERO_AI_CHANNELS: Record<AIChanType, AIChan> = {
  ai_accel: ZERO_AI_ACCEL_CHAN,
  ai_accel_4_wire_dc_voltage: ZERO_AI_ACCEL_4WIRE_DC_VOLTAGE_CHAN,
  ai_accel_charge: ZERO_AI_ACCEL_CHARGE_CHAN,
  ai_bridge: ZERO_AI_BRIDGE_CHAN,
  ai_charge: ZERO_AI_CHARGE_CHAN,
  ai_current: ZERO_AI_CURRENT_CHAN,
  ai_current_rms: ZERO_AI_CURRENT_RMS_CHAN,
  ai_force_bridge_polynomial: ZERO_AI_FORCE_BRIDGE_POLYNOMIAL_CHAN,
  ai_force_bridge_table: ZERO_AI_FORCE_BRIDGE_TABLE_CHAN,
  ai_force_bridge_two_point_lin: ZERO_AI_FORCE_BRIDGE_TWO_POINT_LIN_CHAN,
  ai_force_epe: ZERO_AI_FORCE_EPE_CHAN,
  ai_freq_voltage: ZERO_AI_FREQ_VOLTAGE_CHAN,
  ai_microphone: ZERO_AI_MICROPHONE_CHAN,
  ai_pressure_bridge_polynomial: ZERO_AI_PRESSURE_BRIDGE_POLYNOMIAL_CHAN,
  ai_pressure_bridge_table: ZERO_AI_PRESSURE_BRIDGE_TABLE_CHAN,
  ai_pressure_bridge_two_point_lin: ZERO_AI_PRESSURE_BRIDGE_TWO_POINT_LIN_CHAN,
  ai_resistance: ZERO_AI_RESISTANCE_CHAN,
  ai_rosette_strain_gage: ZERO_AI_ROSETTE_STRAIN_GAGE_CHAN,
  ai_rtd: ZERO_AI_RTD_CHAN,
  ai_strain_gauge: ZERO_AI_STRAIN_GAGE_CHAN,
  ai_temp_builtin: ZERO_ai_temp_builtin_CHAN,
  ai_thermocouple: ZERO_AI_THERMOCOUPLE_CHAN,
  ai_thermistor_iex: ZERO_AI_THERMISTOR_CHAN_IEX,
  ai_thermistor_vex: ZERO_AI_THERMISTOR_CHAN_VEX,
  ai_torque_bridge_polynomial: ZERO_AI_TORQUE_BRIDGE_POLYNOMIAL_CHAN,
  ai_torque_bridge_table: ZERO_AI_TORQUE_BRIDGE_TABLE_CHAN,
  ai_torque_bridge_two_point_lin: ZERO_AI_TORQUE_BRIDGE_TWO_POINT_LIN_CHAN,
  ai_velocity_epe: ZERO_AI_VELOCITY_EPE_CHAN,
  ai_voltage: ZERO_AI_VOLTAGE_CHAN,
  ai_voltage_rms: ZERO_AI_VOLTAGE_RMS_CHAN,
  ai_voltage_with_excit: ZERO_AI_VOLTAGE_CHAN_WITH_EXCIT,
};

export type AnalogInputVoltageChannel = z.infer<typeof aiVoltageChanZ>;

export const analogReadTaskConfigZ = z
  .object({
    device: z.string().min(1),
    sampleRate: z.number().min(0).max(50000),
    streamRate: z.number().min(0).max(50000),
    channels: z.array(aiChan),
  })
  .refine(
    (c) =>
      // Ensure that the stream Rate is lower than the sample rate
      c.sampleRate > c.streamRate,
    {
      path: ["streamRate"],
      message: "Stream rate must be lower than sample rate",
    },
  )
  .superRefine((cfg, ctx) => {
    const ports = new Map<number, number>();
    cfg.channels.forEach(({ port }) => ports.set(port, (ports.get(port) ?? 0) + 1));
    cfg.channels.forEach((channel, i) => {
      if ((ports.get(channel.port) ?? 0) < 2) return;
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        path: ["channels", i, "port"],
        message: `Port ${channel.port} has already been used`,
      });
    });
  })
  .superRefine((cfg, ctx) => {
    const channels = new Map<number, number>();
    cfg.channels.forEach(({ channel }) =>
      channels.set(channel, (channels.get(channel) ?? 0) + 1),
    );
    cfg.channels.forEach((channel, i) => {
      if ((channels.get(channel.channel) ?? 0) < 2) return;
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        path: ["channels", i, "channel"],
        message: `Channel has already been used on port ${channel.port}`,
      });
    });
  });

export type AnalogReadTaskConfig = z.infer<typeof analogReadTaskConfigZ>;
export const analogReadTaskStateZ = z.object({
  running: z.boolean(),
});
export type AnalogReadTaskState = task.State<z.infer<typeof analogReadTaskStateZ>>;

export const ZERO_ANALOG_READ_TASK_CONFIG: AnalogReadTaskConfig = {
  device: "Dev1",
  sampleRate: 10,
  streamRate: 5,
  channels: [],
};

export type AnalogReadTask = task.Task<"ni.analogRead", AnalogReadTaskConfig>;

const digitalOutputChannelZ = z.object({
  key: z.string(),
  type: z.literal("digitalOutput"),
  enabled: z.boolean(),
  port: z.number(),
  line: z.number(),
  channel: z.number(),
});

export type DigitalOutputChannel = z.infer<typeof digitalOutputChannelZ>;

const digitalInputChannelZ = z.object({
  key: z.string(),
  type: z.literal("digitalInput"),
  enabled: z.boolean(),
  port: z.number(),
  line: z.number(),
  channel: z.number(),
});

export type DigitalInputChannel = z.infer<typeof digitalInputChannelZ>;

const digitalWriteChannelZ = z.union([digitalOutputChannelZ, digitalInputChannelZ]);

export const digitalWriteTaskConfigZ = z.object({
  device: z.string().min(1),
  channels: z.array(digitalWriteChannelZ),
});

export type DigitalWriteTaskConfig = z.infer<typeof digitalWriteTaskConfigZ>;

export type DigitalWriteTask = task.Task<"ni.digitalWrite", DigitalWriteTaskConfig>;

const digitalReadChannelZ = digitalInputChannelZ;

export const digitalReadTaskConfigZ = z.object({
  device: z.string().min(1),
  channels: z.array(digitalReadChannelZ),
});

export type DigitalReadTaskConfig = z.infer<typeof digitalReadTaskConfigZ>;

export type DigitalReadTask = task.Task<"ni.analogWrite", DigitalReadTaskConfig>;

export type NITask = AnalogReadTask | DigitalWriteTask | DigitalReadTask;

export type NIChannel =
  | DigitalInputChannel
  | AnalogInputVoltageChannel
  | DigitalOutputChannel;

export const CHANNEL_TYPE_DISPLAY: Record<NIChannel["type"], string> = {
  analogVoltageInput: "Analog Voltage Input",
  digitalInput: "Digital Input",
  digitalOutput: "Digital Output",
};
