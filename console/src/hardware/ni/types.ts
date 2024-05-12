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

const baseChanZ = z.object({
  key: z.string(),
  channel: z.number(),
});

const minMaxValZ = z
  .object({
    minVal: z.number(),
    maxVal: z.number(),
  })
  .refine(({ minVal, maxVal }) => minVal < maxVal, {
    message: "Min value must be less than max value",
  });

export const sensitivityUnitsZ = z.enum(["mVoltsPerG", "VoltsPerG"]);

export type AccelSensitivityUnits = z.infer<typeof sensitivityUnitsZ>;

export const accelerationUnitsZ = z.enum([
  "g",
  "MetersPerSecondSquared",
  "InchesPerSecondSquared",
]);

export type AccelerationUnits = z.infer<typeof accelerationUnitsZ>;

const baseAiAccelChanZ = z.union([
  baseChanZ,
  minMaxValZ,
  z.object({
    terminalConfig: terminalConfigZ,
    sensitivity: z.number(),
    sensitivityUnits: sensitivityUnitsZ,
  }),
]);

// 1 - https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaiaccelchan.html
const aiAccelChanZ = z.union([
  baseAiAccelChanZ,
  z.object({
    type: z.literal("ai_accel"),
    currentExcitSource: excitSourceZ,
    currentExcitVal: z.number(),
  }),
]);

// 2 - https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaiaccel4wiredcvoltagechan.html
const aiAccel4WireDCVoltageChanZ = z.union([
  baseAiAccelChanZ,
  z.object({
    type: z.literal("ai_accel_4_wire_dc_voltage"),
    units: accelerationUnitsZ,
    voltageExcitSource: excitSourceZ,
    voltageExcitVal: z.number(),
    useExcitForScaling: z.boolean(),
  }),
]);

// 3 - https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaiaccelchargechan.html
const aiAccelChargeChanZ = z.union([
  baseAiAccelChanZ,
  z.object({
    type: z.literal("ai_accel_charge"),
    units: accelerationUnitsZ,
  }),
]);

export const bridgeConfigZ = z.enum(["FullBridge", "HalfBridge", "QuarterBridge"]);
export type BridgeConfig = z.infer<typeof bridgeConfigZ>;

// 4 - https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaibridgechan.html
const aiBridgeChanZ = z.union([
  baseChanZ,
  z.object({
    type: z.literal("ai_bridge"),
    terminalConfig: terminalConfigZ,
    minVal: z.number(),
    maxVal: z.number(),
    bridgeConfig: bridgeConfigZ,
    voltageExcitSource: excitSourceZ,
    voltageExcitVal: z.number(),
    nominalBridgeResistance: z.number(),
  }),
]);

// 5 - https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaichargechan.html
const aiChargeChan = z.object({
  key: z.string(),
  type: z.literal("ai_charge"),
  terminalConfig: terminalConfigZ,
  minVal: z.number(),
  maxVal: z.number(),
  units: z.enum(["C, uC"]),
});

// 6 - https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaicurrentchan.html
const aiCurrentChanZ = z.object({
  key: z.string(),
  type: z.literal("ai_current"),
  terminalConfig: terminalConfigZ,
  minVal: z.number(),
  maxVal: z.number(),
  units: z.enum(["A", "mA"]),
  shuntResistorLoc: z.enum(["default", "internal", "external"]),
  extShuntResistorVal: z.number(),
});

// 7 - https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaicurrentrmschan.html
const aiCurrentRMSChanZ = z.object({
  key: z.string(),
  type: z.literal("ai_current_rms"),
  terminalConfig: terminalConfigZ,
  minVal: z.number(),
  maxVal: z.number(),
  units: z.enum(["A", "mA"]),
  shuntResistorLoc: z.enum(["default", "internal", "external"]),
  extShuntResistorVal: z.number(),
});

export const forceUnitsZ = z.enum(["Newtons", "Pounds", "KilogramForce"]);
export type ForceUnits = z.infer<typeof forceUnitsZ>;

export const electricalUnitsZ = z.enum(["mVoltsPerVolt", "VoltsPerVolt"]);
export type ElectricalUnits = z.infer<typeof electricalUnitsZ>;

// 8 - https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaiforcebridgepolynomialchan.html
const aiForceBridgePolynomialChanZ = z.object({
  key: z.string(),
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

// 9 - https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaiforcebridgetablechan.html
const aiForceBridgeTableChanZ = z.object({
  key: z.string(),
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

// 10 - https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaiforcebridgetwopointlinchan.html
const aiForceBridgeTwoPointLinChan = z.object({
  key: z.string(),
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

// 11 - https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaiforceiepechan.html
const aiForceEPEChanZ = z.object({
  key: z.string(),
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

// 12 - https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaifreqvoltagechan.html
const aiFreqVoltageChanZ = z.object({
  key: z.string(),
  type: z.literal("ai_freq_voltage"),
  minVal: z.number(),
  maxVal: z.number(),
  units: z.enum(["Hz"]),
  thresholdLevel: z.number(),
  hysteresis: z.number(),
});

// 13 - https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaimicrophonechan.html
const aiMicrophoneChanZ = z.object({
  key: z.string(),
  type: z.literal("ai_microphone"),
  terminalConfig: terminalConfigZ,
  micSensitivity: z.number(),
  maxSndPressLevel: z.number(),
  currentExcitSource: excitSourceZ,
  currentExcitVal: z.number(),
});

export const pressureUnitsZ = z.enum(["psi", "Pa", "bar"]);
export type PressureUnits = z.infer<typeof pressureUnitsZ>;

// 14 - https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaipressurebridgepolynomialchan.html
const aiPressureBridgePolynomialChanZ = z.object({
  key: z.string(),
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

// 15 - https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaipressurebridgetablechan.html
const aiPressureBridgeTableChanZ = z.object({
  key: z.string(),
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

// 16 - https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaipressurebridgetwopointlinchan.html
const aiPressureBridgeTwoPointLinChanZ = z.object({
  key: z.string(),
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

export const resistanceConfigZ = z.enum(["2Wire", "3Wire", "4Wire"]);
export type ResistanceConfig = z.infer<typeof resistanceConfigZ>;

// 17 - https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateairesistancechan.html
const aiResistanceChanZ = z.object({
  key: z.string(),
  type: z.literal("ai_resistance"),
  terminalConfig: terminalConfigZ,
  minVal: z.number(),
  maxVal: z.number(),
  units: z.enum(["Ohms"]),
  resistanceConfig: resistanceConfigZ,
  currentExcitSource: excitSourceZ,
  currentExcitVal: z.number(),
});

// 18 -  https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateairosettestraingagechan.html
const aiRosetteStrainGageChanZ = z.object({
  key: z.string(),
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

export const temperatureUnitsZ = z.enum(["DegC", "DegF", "Kelvins", "DegR"]);
export type TemperatureUnits = z.infer<typeof temperatureUnitsZ>;

// 19 -  https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateairtdchan.html
const aiRTDChanZ = z.object({
  key: z.string(),
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

// 20 - https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaistraingagechan.html
const aiStrainGageChan = z.object({
  key: z.string(),
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

// 21 - https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaitempbuiltinsensorchan.html
const aiTempBuiltInChanZ = z.object({
  key: z.string(),
  type: z.literal("ai_temp_built_in"),
  units: temperatureUnitsZ,
});

// 22 - https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaithrmcplchan.html
const aiThermocoupleChanZ = z.object({
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

// 23 - https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaithrmstrchaniex.html
const aiThermistorChanIex = z.object({
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

// 24 - https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaitorquebridgepolynomialchan.html
const aiThermistorChanVex = z.object({
  key: z.string(),
  type: z.literal("ai_thermistor_vex"),
  minVal: z.number(),
  maxVal: z.number(),
  units: terminalConfigZ,
  resistanceConfig: resistanceConfigZ,
  voltageExcitSource: excitSourceZ,
  voltageExcitVal: z.number(),
  a: z.number(),
  b: z.number(),
  c: z.number(),
  r1: z.number(),
});

export const torqueUnitsZ = z.enum(["NewtonMeters", "InchOunces", "FootPounds"]);
export type TorqueUnits = z.infer<typeof torqueUnitsZ>;

// 25 - https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaitorquebridgepolynomialchan.html
const aiTorqueBridgePolynomialChanZ = z.object({
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

// 25 - https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaitorquebridgetablechan.html
const aiTorqueBridgeTableChanZ = z.object({
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

// 27 - https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaitorquebridgetwopointlinchan.html
const aiTorqueBridgeTwoPointLinChanZ = z.object({
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

export const velocityUnitsZ = z.enum(["m/s", "in/s"]);
export type VelocityUnits = z.infer<typeof velocityUnitsZ>;

// 28 - https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaivelocityiepechan.html
const aiVelocityEPEChanZ = z.object({
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

const analogInputScaleZ = linearScaleZ;

// 29 - https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaivoltagechan.html
const aiVoltageChanZ = z.object({
  key: z.string(),
  type: z.literal("ai_voltage"),
  enabled: z.boolean(),
  port: z.number(),
  channel: z.number(),
  scale: analogInputScaleZ,
});

// 30 - https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaivoltagermschan.html
const aiVoltageRMSChanZ = z.object({
  key: z.string(),
  type: z.literal("ai_voltage_rms"),
  terminalConfig: terminalConfigZ,
  minVal: z.number(),
  maxVal: z.number(),
  units: z.enum(["V", "mV"]),
});

// 31 - https://www.ni.com/docs/en-US/bundle/ni-daqmx-c-api-ref/page/daqmxcfunc/daqmxcreateaivoltagechanwithexcit.html
const aiVoltageChanWithExcitZ = z.object({
  key: z.string(),
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
