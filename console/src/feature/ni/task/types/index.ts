// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ni, type task } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/pluto";
import { z } from "zod";

import { createPortValidator } from "@/feature/ni/task/types/validation";
import { Task } from "@/platform/task";

export const PREFIX = "ni";

export type Units = ni.Units;
export type AccelSensitivityUnits = ni.AccelSensitivityUnits;
export type ForceUnits = ni.ForceUnits;
export type ElectricalUnits = ni.ElectricalUnits;
export type ShuntResistorLoc = ni.ShuntResistorLocation;
export type PressureUnits = ni.PressureUnits;
export type TemperatureUnits = ni.TemperatureUnits;
export type TorqueUnits = ni.TorqueUnits;
export type VelocityUnits = ni.VelocityUnits;
export type VelocitySensitivityUnits = ni.VelocitySensitivityUnits;

export const WAVE_TYPES = ni.WAVE_TYPES;
export type WaveType = ni.WaveType;

export type Scale = ni.Scale;
export type ScaleType = ni.ScaleType;
export type CJC = ni.CJC;
export type CJCType = ni.CJCType;

// Cross-field scale validation is not generated; re-attach it to the map and table
// scale schemas the console exposes.
const mapScaleZ = ni.scaleMapZ
  .refine(({ preScaledMin, preScaledMax }) => preScaledMin < preScaledMax, {
    message: "Pre-scaled min must be less than pre-scaled max",
    path: ["preScaledMin"],
  })
  .refine(({ scaledMin, scaledMax }) => scaledMin < scaledMax, {
    message: "Scaled min must be less than scaled max",
    path: ["scaledMin"],
  });

const tableScaleZ = ni.scaleTableZ
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

export const SCALE_SCHEMAS = {
  ...ni.SCALE_SCHEMAS,
  map: mapScaleZ,
  table: tableScaleZ,
} as Record<ScaleType, z.ZodType<Scale>>;

const NO_SCALE: Scale = { type: "none" };
const BUILT_IN_CJC: CJC = { source: "built_in" };

export const ZERO_SCALES = Object.fromEntries(
  ni.SCALE_TYPES.map((t) => [t, ni.SCALE_SCHEMAS[t].parse({ type: t })]),
) as Record<ScaleType, Scale>;

export type AIChannel = ni.AIChannel;
export type AIChannelType = ni.AIChannelType;
export const AI_CHANNEL_SCHEMAS = ni.AI_CHANNEL_SCHEMAS;

// A complete default channel for every generated type, derived from the schema
// defaults. The console only offers the types in AI_CHANNEL_TYPE_NAMES; the rest exist
// so configs created by the Python client or the Driver still round-trip.
export const ZERO_AI_CHANNELS = Object.fromEntries(
  ni.AI_CHANNEL_TYPES.map((t) => [
    t,
    ni.AI_CHANNEL_SCHEMAS[t].parse({
      type: t,
      customScale: NO_SCALE,
      cjc: BUILT_IN_CJC,
    }),
  ]),
) as Record<AIChannelType, AIChannel>;
export const ZERO_AI_CHANNEL = ZERO_AI_CHANNELS.ai_voltage;

export const AI_CHANNEL_TYPE_NAMES: Partial<Record<AIChannelType, string>> = {
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

// Total over every AIChannelType, including the Driver/Python-only types the console
// does not offer in its select. Those reuse their measurement category's icon so a task
// created elsewhere still renders.
export const AI_CHANNEL_TYPE_ICONS: Record<AIChannelType, Icon.FC> = {
  ai_accel: Icon.Units.Acceleration,
  ai_accel_4_wire_dc_voltage: Icon.Units.Acceleration,
  ai_accel_charge: Icon.Units.Acceleration,
  ai_bridge: Icon.Bridge,
  ai_charge: Icon.Units.Voltage,
  ai_current: Icon.Units.Current,
  ai_current_rms: Icon.Units.Current,
  ai_force_bridge_polynomial: Icon.Units.Force,
  ai_force_bridge_table: Icon.Units.Force,
  ai_force_bridge_two_point_lin: Icon.Units.Force,
  ai_force_iepe: Icon.Units.Force,
  ai_freq_voltage: Icon.Units.Voltage,
  ai_microphone: Icon.Sound,
  ai_pressure_bridge_polynomial: Icon.Units.Pressure,
  ai_pressure_bridge_table: Icon.Units.Pressure,
  ai_pressure_bridge_two_point_lin: Icon.Units.Pressure,
  ai_resistance: Icon.Units.Resistance,
  ai_rtd: Icon.Units.Temperature,
  ai_strain_gauge: Icon.Units.Strain,
  ai_temp_builtin: Icon.Units.Temperature,
  ai_thermistor_iex: Icon.Units.Temperature,
  ai_thermistor_vex: Icon.Units.Temperature,
  ai_thermocouple: Icon.Units.Temperature,
  ai_torque_bridge_polynomial: Icon.Units.Torque,
  ai_torque_bridge_table: Icon.Units.Torque,
  ai_torque_bridge_two_point_lin: Icon.Units.Torque,
  ai_velocity_iepe: Icon.Units.Velocity,
  ai_voltage: Icon.Units.Voltage,
  ai_voltage_rms: Icon.Units.Voltage,
  ai_voltage_with_excit: Icon.Units.Voltage,
};

export type CIChannel = ni.CIChannel;
export type CIChannelType = ni.CIChannelType;
export type CIFreqUnits = ni.CIFreqUnits;
export type CIPeriodUnits = ni.CITimeUnits;
export type CIPulseWidthUnits = ni.CITimeUnits;
export type CISemiPeriodUnits = ni.CITimeUnits;
export type CITwoEdgeSepUnits = ni.CITimeUnits;
export type CILinearVelocityUnits = ni.CILinearVelocityUnits;
export type CIAngularVelocityUnits = ni.CIAngularVelocityUnits;
export type CILinearPositionUnits = ni.CILinearPositionUnits;
export type CIAngularPositionUnits = ni.CIAngularPositionUnits;
export type ZIndexPhase = ni.ZIndexPhase;
export type CIEdge = ni.CIEdge;
export type CIMeasMethod = ni.CIMeasMethod;
export type CICountDirection = ni.CICountDirection;
export type CIDecodingType = ni.CIDecodingType;
export const CI_CHANNEL_SCHEMAS = ni.CI_CHANNEL_SCHEMAS;

export const ZERO_CI_CHANNELS = Object.fromEntries(
  ni.CI_CHANNEL_TYPES.map((t) => [
    t,
    ni.CI_CHANNEL_SCHEMAS[t].parse({ type: t, customScale: NO_SCALE }),
  ]),
) as Record<CIChannelType, CIChannel>;
export const ZERO_CI_CHANNEL = ZERO_CI_CHANNELS.ci_frequency;

export const CI_CHANNEL_TYPE_NAMES: Record<CIChannelType, string> = {
  ci_frequency: "Frequency",
  ci_edge_count: "Edge Count",
  ci_period: "Period",
  ci_pulse_width: "Pulse Width",
  ci_semi_period: "Semi Period",
  ci_two_edge_sep: "Two Edge Separation",
  ci_position_angular: "Position Angular",
  ci_position_linear: "Position Linear",
  ci_velocity_angular: "Velocity Angular",
  ci_velocity_linear: "Velocity Linear",
  ci_duty_cycle: "Duty Cycle",
};

export const CI_CHANNEL_TYPE_ICONS: Record<CIChannelType, Icon.FC> = {
  ci_frequency: Icon.Wave.Square,
  ci_edge_count: Icon.Value,
  ci_period: Icon.Time,
  ci_pulse_width: Icon.AutoFitWidth,
  ci_semi_period: Icon.Range,
  ci_two_edge_sep: Icon.AutoFitWidth,
  ci_position_angular: Icon.Rotate,
  ci_position_linear: Icon.Linear,
  ci_velocity_angular: Icon.Rotate,
  ci_velocity_linear: Icon.Linear,
  ci_duty_cycle: Icon.Wave.Square,
};

export const AO_CURRENT_CHAN_TYPE = "ao_current";
export const AO_FUNC_GEN_CHAN_TYPE = "ao_func_gen";
export const AO_VOLTAGE_CHAN_TYPE = "ao_voltage";

export type AOChannel = ni.AOChannel;
export type AOChannelType = ni.AOChannelType;
export const AO_CHANNEL_SCHEMAS = ni.AO_CHANNEL_SCHEMAS;
export const AO_CHANNEL_TYPES = ni.AO_CHANNEL_TYPES;

export const ZERO_AO_CHANNELS = Object.fromEntries(
  ni.AO_CHANNEL_TYPES.map((t) => [
    t,
    ni.AO_CHANNEL_SCHEMAS[t].parse({ type: t, customScale: NO_SCALE }),
  ]),
) as Record<AOChannelType, AOChannel>;
export const ZERO_AO_CHANNEL = ZERO_AO_CHANNELS.ao_voltage;

export const AO_CHANNEL_TYPE_NAMES: Record<AOChannelType, string> = {
  ao_current: "Current",
  ao_func_gen: "Function Generator",
  ao_voltage: "Voltage",
};

export const AO_CHANNEL_TYPE_ICONS: Record<AOChannelType, Icon.FC> = {
  ao_current: Icon.Units.Current,
  ao_func_gen: Icon.Function,
  ao_voltage: Icon.Units.Voltage,
};

export type AnalogChannel = AIChannel | AOChannel;

export type DIChannel = ni.DIChannel;
export const ZERO_DI_CHANNEL: DIChannel = ni.diChannelZ.parse({
  type: "digital_input",
});

export type DOChannel = ni.DOChannel;
export const ZERO_DO_CHANNEL: DOChannel = ni.doChannelZ.parse({
  type: "digital_output",
});

export type DigitalChannel = DIChannel | DOChannel;

export type Channel = AnalogChannel | DigitalChannel;

const deployReadRateShape = {
  sampleRate: z.number().positive().max(1000000),
  streamRate: z.number().positive().max(20000),
} as const;

const deployStateRateZ = z.number().positive().max(50000);

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

export const analogReadConfigZ = ni.analogReadConfigZ;

export const deployAnalogReadConfigZ = ni.analogReadConfigZ
  .extend({
    ...deployReadRateShape,
    channels: z
      .array(ni.aiChannelZ)
      .check(Task.validateReadChannels)
      .check(createPortValidator())
      .check(Task.validateChannelDevices),
  })
  .check(Task.validateStreamRate);

const ZERO_ANALOG_READ_CONFIG = ni.analogReadConfigZ.parse({});

const analogReadStatusDataZ = z
  .object({ errors: z.array(z.object({ message: z.string(), path: z.string() })) })
  .nullish()
  .optional();

export const ANALOG_READ_TYPE = `${PREFIX}_analog_read`;

export const ANALOG_READ_SCHEMAS = {
  type: z.literal(ANALOG_READ_TYPE),
  config: analogReadConfigZ,
  statusData: analogReadStatusDataZ,
} as const satisfies task.Schemas;

export type AnalogReadSchemas = typeof ANALOG_READ_SCHEMAS;

interface AnalogReadPayload extends task.Payload<AnalogReadSchemas> {}
export const ZERO_ANALOG_READ_PAYLOAD: AnalogReadPayload = {
  key: "",
  rack: 0,
  name: "NI Analog Read Task",
  config: ZERO_ANALOG_READ_CONFIG,
  configHash: "",
  type: ANALOG_READ_TYPE,
  internal: false,
  snapshot: false,
};

const validateCounterPorts = createPortValidator("Counter");

export const counterReadConfigZ = ni.counterReadConfigZ;

export const deployCounterReadConfigZ = ni.counterReadConfigZ
  .extend({
    ...deployReadRateShape,
    channels: z
      .array(ni.ciChannelZ)
      .check(Task.validateReadChannels)
      .check(validateCounterPorts)
      .check(Task.validateChannelDevices),
  })
  .check(Task.validateStreamRate);

const ZERO_COUNTER_READ_CONFIG = ni.counterReadConfigZ.parse({});

export const COUNTER_READ_TYPE = `${PREFIX}_counter_read`;

export const COUNTER_READ_SCHEMAS = {
  type: z.literal(COUNTER_READ_TYPE),
  config: counterReadConfigZ,
  statusData: z.unknown().optional(),
} as const satisfies task.Schemas;

export type CounterReadSchemas = typeof COUNTER_READ_SCHEMAS;

interface CounterReadPayload extends task.Payload<CounterReadSchemas> {}
export const ZERO_COUNTER_READ_PAYLOAD: CounterReadPayload = {
  key: "",
  rack: 0,
  name: "NI Counter Read Task",
  config: ZERO_COUNTER_READ_CONFIG,
  configHash: "",
  type: COUNTER_READ_TYPE,
  internal: false,
  snapshot: false,
};

export const analogWriteConfigZ = ni.analogWriteConfigZ;

export const deployAnalogWriteConfigZ = ni.analogWriteConfigZ.extend({
  device: Task.deviceKeyZ,
  stateRate: deployStateRateZ,
  channels: z
    .array(ni.aoChannelZ)
    .check(Task.validateWriteChannels)
    .check(validateAnalogPorts),
});

const ZERO_ANALOG_WRITE_CONFIG = ni.analogWriteConfigZ.parse({});

export const ANALOG_WRITE_TYPE = `${PREFIX}_analog_write`;

export const ANALOG_WRITE_SCHEMAS = {
  type: z.literal(ANALOG_WRITE_TYPE),
  config: analogWriteConfigZ,
  statusData: z.unknown().optional(),
} as const satisfies task.Schemas;

export type AnalogWriteSchemas = typeof ANALOG_WRITE_SCHEMAS;

interface AnalogWritePayload extends task.Payload<AnalogWriteSchemas> {}
export const ZERO_ANALOG_WRITE_PAYLOAD: AnalogWritePayload = {
  key: "",
  rack: 0,
  name: "NI Analog Write Task",
  config: ZERO_ANALOG_WRITE_CONFIG,
  configHash: "",
  type: ANALOG_WRITE_TYPE,
  internal: false,
  snapshot: false,
};

export const digitalReadConfigZ = ni.digitalReadConfigZ;

export const deployDigitalReadConfigZ = ni.digitalReadConfigZ
  .extend({
    ...deployReadRateShape,
    device: Task.deviceKeyZ,
    channels: z
      .array(ni.diChannelZ)
      .check(Task.validateReadChannels)
      .check(validateDigitalPortsAndLines),
  })
  .check(Task.validateStreamRate);

const ZERO_DIGITAL_READ_CONFIG = ni.digitalReadConfigZ.parse({});

export const DIGITAL_READ_TYPE = `${PREFIX}_digital_read`;

export const DIGITAL_READ_SCHEMAS = {
  type: z.literal(DIGITAL_READ_TYPE),
  config: digitalReadConfigZ,
  statusData: z.unknown().optional(),
} as const satisfies task.Schemas;

export type DigitalReadSchemas = typeof DIGITAL_READ_SCHEMAS;

interface DigitalReadPayload extends task.Payload<DigitalReadSchemas> {}
export const ZERO_DIGITAL_READ_PAYLOAD: DigitalReadPayload = {
  key: "",
  rack: 0,
  name: "NI Digital Read Task",
  config: ZERO_DIGITAL_READ_CONFIG,
  configHash: "",
  type: DIGITAL_READ_TYPE,
  internal: false,
  snapshot: false,
};

export const digitalWriteConfigZ = ni.digitalWriteConfigZ;

export const deployDigitalWriteConfigZ = ni.digitalWriteConfigZ.extend({
  device: Task.deviceKeyZ,
  stateRate: deployStateRateZ,
  channels: z
    .array(ni.doChannelZ)
    .check(Task.validateWriteChannels)
    .check(validateDigitalPortsAndLines),
});

const ZERO_DIGITAL_WRITE_CONFIG = ni.digitalWriteConfigZ.parse({});

export const DIGITAL_WRITE_TYPE = `${PREFIX}_digital_write`;

export const DIGITAL_WRITE_SCHEMAS = {
  type: z.literal(DIGITAL_WRITE_TYPE),
  config: digitalWriteConfigZ,
  statusData: z.unknown().optional(),
} as const satisfies task.Schemas;

export type DigitalWriteSchemas = typeof DIGITAL_WRITE_SCHEMAS;

interface DigitalWritePayload extends task.Payload<DigitalWriteSchemas> {}
export const ZERO_DIGITAL_WRITE_PAYLOAD: DigitalWritePayload = {
  key: "",
  rack: 0,
  name: "NI Digital Write Task",
  config: ZERO_DIGITAL_WRITE_CONFIG,
  configHash: "",
  type: DIGITAL_WRITE_TYPE,
  internal: false,
  snapshot: false,
};

export const SCAN_TYPE = `${PREFIX}_scanner`;

export const SCAN_SCHEMAS = {
  type: z.literal(SCAN_TYPE),
  config: z.object({ enabled: z.boolean().default(true) }),
  statusData: z.unknown().optional(),
} as const satisfies task.Schemas;
