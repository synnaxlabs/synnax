// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device, type task } from "@synnaxlabs/client";
import { z } from "zod";

import { Common } from "@/hardware/common";
import * as v0 from "@/hardware/ni/task/types/v0";

type PortToIndexMap = Map<number, number>;

const validateAnalogPorts = ({
  value: channels,
  issues,
}: z.core.ParsePayload<{ port: number; device: device.Key }[]>) => {
  const deviceToPortMap = new Map<device.Key, PortToIndexMap>();
  channels.forEach(({ device, port }, i) => {
    if (!deviceToPortMap.has(device)) deviceToPortMap.set(device, new Map());
    const portToIndexMap = deviceToPortMap.get(device) as PortToIndexMap;
    if (!portToIndexMap.has(port)) {
      portToIndexMap.set(port, i);
      return;
    }
    const index = portToIndexMap.get(port) as number;
    const code = "custom";
    const message = `Port ${port} has already been used on another channel on the same device`;
    issues.push({ path: [index, "port"], code, message, input: channels });
    issues.push({ path: [i, "port"], code, message, input: channels });
  });
};

const aiChanExtensionShape = { device: Common.Device.keyZ };
interface AIChanExtension extends z.infer<z.ZodObject<typeof aiChanExtensionShape>> {}
const ZERO_AI_CHAN_EXTENSION: AIChanExtension = { device: "" };

const aiAccelChanZ = v0.aiAccelChanZ.extend(aiChanExtensionShape);
interface AIAccelChan extends z.infer<typeof aiAccelChanZ> {}
const ZERO_AI_ACCEL_CHAN: AIAccelChan = {
  ...v0.ZERO_AI_ACCEL_CHAN,
  ...ZERO_AI_CHAN_EXTENSION,
};

const aiBridgeChanZ = v0.aiBridgeChanZ.extend(aiChanExtensionShape);
interface AIBridgeChan extends z.infer<typeof aiBridgeChanZ> {}
const ZERO_AI_BRIDGE_CHAN: AIBridgeChan = {
  ...v0.ZERO_AI_BRIDGE_CHAN,
  ...ZERO_AI_CHAN_EXTENSION,
};

const aiCurrentChanZ = v0.aiCurrentChanZ.extend(aiChanExtensionShape);
interface AICurrentChan extends z.infer<typeof aiCurrentChanZ> {}
const ZERO_AI_CURRENT_CHAN: AICurrentChan = {
  ...v0.ZERO_AI_CURRENT_CHAN,
  ...ZERO_AI_CHAN_EXTENSION,
};

const aiForceBridgeTableChanZ = v0.aiForceBridgeTableChanZ.extend(aiChanExtensionShape);
interface AIForceBridgeTableChan extends z.infer<typeof aiForceBridgeTableChanZ> {}
const ZERO_AI_FORCE_BRIDGE_TABLE_CHAN: AIForceBridgeTableChan = {
  ...v0.ZERO_AI_FORCE_BRIDGE_TABLE_CHAN,
  ...ZERO_AI_CHAN_EXTENSION,
};

const aiForceBridgeTwoPointLinChanZ =
  v0.aiForceBridgeTwoPointLinChanZ.extend(aiChanExtensionShape);
interface AIForceBridgeTwoPointLinChan
  extends z.infer<typeof aiForceBridgeTwoPointLinChanZ> {}
const ZERO_AI_FORCE_BRIDGE_TWO_POINT_LIN_CHAN: AIForceBridgeTwoPointLinChan = {
  ...v0.ZERO_AI_FORCE_BRIDGE_TWO_POINT_LIN_CHAN,
  ...ZERO_AI_CHAN_EXTENSION,
};

const aiForceIEPEChanZ = v0.aiForceIEPEChanZ.extend(aiChanExtensionShape);
interface AIForceIEPEChan extends z.infer<typeof aiForceIEPEChanZ> {}
const ZERO_AI_FORCE_IEPE_CHAN: AIForceIEPEChan = {
  ...v0.ZERO_AI_FORCE_IEPE_CHAN,
  ...ZERO_AI_CHAN_EXTENSION,
};

const aiMicrophoneChanZ = v0.aiMicrophoneChanZ.extend(aiChanExtensionShape);
interface AIMicrophoneChan extends z.infer<typeof aiMicrophoneChanZ> {}
const ZERO_AI_MICROPHONE_CHAN: AIMicrophoneChan = {
  ...v0.ZERO_AI_MICROPHONE_CHAN,
  ...ZERO_AI_CHAN_EXTENSION,
};

const aiPressureBridgeTableChanZ =
  v0.aiPressureBridgeTableChanZ.extend(aiChanExtensionShape);
interface AIPressureBridgeTableChan
  extends z.infer<typeof aiPressureBridgeTableChanZ> {}
const ZERO_AI_PRESSURE_BRIDGE_TABLE_CHAN: AIPressureBridgeTableChan = {
  ...v0.ZERO_AI_PRESSURE_BRIDGE_TABLE_CHAN,
  ...ZERO_AI_CHAN_EXTENSION,
};

const aiPressureBridgeTwoPointLinChanZ =
  v0.aiPressureBridgeTwoPointLinChanZ.extend(aiChanExtensionShape);
interface AIPressureBridgeTwoPointLinChan
  extends z.infer<typeof aiPressureBridgeTwoPointLinChanZ> {}
const ZERO_AI_PRESSURE_BRIDGE_TWO_POINT_LIN_CHAN: AIPressureBridgeTwoPointLinChan = {
  ...v0.ZERO_AI_PRESSURE_BRIDGE_TWO_POINT_LIN_CHAN,
  ...ZERO_AI_CHAN_EXTENSION,
};

const aiResistanceChanZ = v0.aiResistanceChanZ.extend(aiChanExtensionShape);
interface AIResistanceChan extends z.infer<typeof aiResistanceChanZ> {}
const ZERO_AI_RESISTANCE_CHAN: AIResistanceChan = {
  ...v0.ZERO_AI_RESISTANCE_CHAN,
  ...ZERO_AI_CHAN_EXTENSION,
};

const aiRTDChanZ = v0.aiRTDChanZ.extend(aiChanExtensionShape);
interface AIRTDChan extends z.infer<typeof aiRTDChanZ> {}
const ZERO_AI_RTD_CHAN: AIRTDChan = {
  ...v0.ZERO_AI_RTD_CHAN,
  ...ZERO_AI_CHAN_EXTENSION,
};

const aiStrainGageChanZ = v0.aiStrainGageChanZ.extend(aiChanExtensionShape);
interface AIStrainGageChan extends z.infer<typeof aiStrainGageChanZ> {}
const ZERO_AI_STRAIN_GAGE_CHAN: AIStrainGageChan = {
  ...v0.ZERO_AI_STRAIN_GAGE_CHAN,
  ...ZERO_AI_CHAN_EXTENSION,
};

const aiTempBuiltInChanZ = v0.aiTempBuiltInChanZ.extend(aiChanExtensionShape);
interface AITempBuiltInChan extends z.infer<typeof aiTempBuiltInChanZ> {}
const ZERO_AI_TEMP_BUILT_IN_CHAN: AITempBuiltInChan = {
  ...v0.ZERO_AI_TEMP_BUILT_IN_CHAN,
  ...ZERO_AI_CHAN_EXTENSION,
};

const aiThrmcplChanZ = v0.aiThrmcplChanZ.and(z.object(aiChanExtensionShape));
type AIThrmcplChan = z.infer<typeof aiThrmcplChanZ>;
const ZERO_AI_THRMCPL_CHAN: AIThrmcplChan = {
  ...v0.ZERO_AI_THRMCPL_CHAN,
  ...ZERO_AI_CHAN_EXTENSION,
};

const aiTorqueBridgeTableChanZ =
  v0.aiTorqueBridgeTableChanZ.extend(aiChanExtensionShape);
interface AITorqueBridgeTableChan extends z.infer<typeof aiTorqueBridgeTableChanZ> {}
const ZERO_AI_TORQUE_BRIDGE_TABLE_CHAN: AITorqueBridgeTableChan = {
  ...v0.ZERO_AI_TORQUE_BRIDGE_TABLE_CHAN,
  ...ZERO_AI_CHAN_EXTENSION,
};

const aiTorqueBridgeTwoPointLinChanZ =
  v0.aiTorqueBridgeTwoPointLinChanZ.extend(aiChanExtensionShape);
interface AITorqueBridgeTwoPointLinChan
  extends z.infer<typeof aiTorqueBridgeTwoPointLinChanZ> {}
const ZERO_AI_TORQUE_BRIDGE_TWO_POINT_LIN_CHAN: AITorqueBridgeTwoPointLinChan = {
  ...v0.ZERO_AI_TORQUE_BRIDGE_TWO_POINT_LIN_CHAN,
  ...ZERO_AI_CHAN_EXTENSION,
};

const aiVelocityIEPEChanZ = v0.aiVelocityIEPEChanZ.extend(aiChanExtensionShape);
interface AIVelocityIEPEChan extends z.infer<typeof aiVelocityIEPEChanZ> {}
const ZERO_AI_VELOCITY_IEPE_CHAN: AIVelocityIEPEChan = {
  ...v0.ZERO_AI_VELOCITY_IEPE_CHAN,
  ...ZERO_AI_CHAN_EXTENSION,
};

const aiVoltageChanZ = v0.aiVoltageChanZ.extend(aiChanExtensionShape);
interface AIVoltageChan extends z.infer<typeof aiVoltageChanZ> {}
const ZERO_AI_VOLTAGE_CHAN: AIVoltageChan = {
  ...v0.ZERO_AI_VOLTAGE_CHAN,
  ...ZERO_AI_CHAN_EXTENSION,
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
export type AIChannel = z.infer<typeof aiChannelZ>;

export const AI_CHANNEL_SCHEMAS: Record<v0.AIChannelType, z.ZodType<AIChannel>> = {
  [v0.AI_ACCEL_CHAN_TYPE]: aiAccelChanZ,
  [v0.AI_BRIDGE_CHAN_TYPE]: aiBridgeChanZ,
  [v0.AI_CURRENT_CHAN_TYPE]: aiCurrentChanZ,
  [v0.AI_FORCE_BRIDGE_TABLE_CHAN_TYPE]: aiForceBridgeTableChanZ,
  [v0.AI_FORCE_BRIDGE_TWO_POINT_LIN_CHAN_TYPE]: aiForceBridgeTwoPointLinChanZ,
  [v0.AI_FORCE_IEPE_CHAN_TYPE]: aiForceIEPEChanZ,
  [v0.AI_MICROPHONE_CHAN_TYPE]: aiMicrophoneChanZ,
  [v0.AI_PRESSURE_BRIDGE_TABLE_CHAN_TYPE]: aiPressureBridgeTableChanZ,
  [v0.AI_PRESSURE_BRIDGE_TWO_POINT_LIN_CHAN_TYPE]: aiPressureBridgeTwoPointLinChanZ,
  [v0.AI_RESISTANCE_CHAN_TYPE]: aiResistanceChanZ,
  [v0.AI_RTD_CHAN_TYPE]: aiRTDChanZ,
  [v0.AI_STRAIN_GAGE_CHAN_TYPE]: aiStrainGageChanZ,
  [v0.AI_TEMP_BUILT_IN_CHAN_TYPE]: aiTempBuiltInChanZ,
  [v0.AI_THRMCPL_CHAN_TYPE]: aiThrmcplChanZ,
  [v0.AI_TORQUE_BRIDGE_TABLE_CHAN_TYPE]: aiTorqueBridgeTableChanZ,
  [v0.AI_TORQUE_BRIDGE_TWO_POINT_LIN_CHAN_TYPE]: aiTorqueBridgeTwoPointLinChanZ,
  [v0.AI_VELOCITY_IEPE_CHAN_TYPE]: aiVelocityIEPEChanZ,
  [v0.AI_VOLTAGE_CHAN_TYPE]: aiVoltageChanZ,
};

export const ZERO_AI_CHANNELS: Record<v0.AIChannelType, AIChannel> = {
  [v0.AI_ACCEL_CHAN_TYPE]: ZERO_AI_ACCEL_CHAN,
  [v0.AI_BRIDGE_CHAN_TYPE]: ZERO_AI_BRIDGE_CHAN,
  [v0.AI_CURRENT_CHAN_TYPE]: ZERO_AI_CURRENT_CHAN,
  [v0.AI_FORCE_BRIDGE_TABLE_CHAN_TYPE]: ZERO_AI_FORCE_BRIDGE_TABLE_CHAN,
  [v0.AI_FORCE_BRIDGE_TWO_POINT_LIN_CHAN_TYPE]: ZERO_AI_FORCE_BRIDGE_TWO_POINT_LIN_CHAN,
  [v0.AI_FORCE_IEPE_CHAN_TYPE]: ZERO_AI_FORCE_IEPE_CHAN,
  [v0.AI_MICROPHONE_CHAN_TYPE]: ZERO_AI_MICROPHONE_CHAN,
  [v0.AI_PRESSURE_BRIDGE_TABLE_CHAN_TYPE]: ZERO_AI_PRESSURE_BRIDGE_TABLE_CHAN,
  [v0.AI_PRESSURE_BRIDGE_TWO_POINT_LIN_CHAN_TYPE]:
    ZERO_AI_PRESSURE_BRIDGE_TWO_POINT_LIN_CHAN,
  [v0.AI_RESISTANCE_CHAN_TYPE]: ZERO_AI_RESISTANCE_CHAN,
  [v0.AI_RTD_CHAN_TYPE]: ZERO_AI_RTD_CHAN,
  [v0.AI_STRAIN_GAGE_CHAN_TYPE]: ZERO_AI_STRAIN_GAGE_CHAN,
  [v0.AI_TEMP_BUILT_IN_CHAN_TYPE]: ZERO_AI_TEMP_BUILT_IN_CHAN,
  [v0.AI_THRMCPL_CHAN_TYPE]: ZERO_AI_THRMCPL_CHAN,
  [v0.AI_TORQUE_BRIDGE_TABLE_CHAN_TYPE]: ZERO_AI_TORQUE_BRIDGE_TABLE_CHAN,
  [v0.AI_TORQUE_BRIDGE_TWO_POINT_LIN_CHAN_TYPE]:
    ZERO_AI_TORQUE_BRIDGE_TWO_POINT_LIN_CHAN,
  [v0.AI_VELOCITY_IEPE_CHAN_TYPE]: ZERO_AI_VELOCITY_IEPE_CHAN,
  [v0.AI_VOLTAGE_CHAN_TYPE]: ZERO_AI_VOLTAGE_CHAN,
};
export const ZERO_AI_CHANNEL: AIChannel = ZERO_AI_CHANNELS[v0.AI_VOLTAGE_CHAN_TYPE];

// ==================== Counter Input Channels ====================

const ciChanExtensionShape = { device: Common.Device.keyZ };
interface CIChanExtension extends z.infer<z.ZodObject<typeof ciChanExtensionShape>> {}
const ZERO_CI_CHAN_EXTENSION: CIChanExtension = { device: "" };

const ciFrequencyChanZ = v0.ciFrequencyChanZ.extend(ciChanExtensionShape);
interface CIFrequencyChan extends z.infer<typeof ciFrequencyChanZ> {}
const ZERO_CI_FREQUENCY_CHAN: CIFrequencyChan = {
  ...v0.ZERO_CI_FREQUENCY_CHAN,
  ...ZERO_CI_CHAN_EXTENSION,
};

const ciEdgeCountChanZ = v0.ciEdgeCountChanZ.extend(ciChanExtensionShape);
interface CIEdgeCountChan extends z.infer<typeof ciEdgeCountChanZ> {}
const ZERO_CI_EDGE_COUNT_CHAN: CIEdgeCountChan = {
  ...v0.ZERO_CI_EDGE_COUNT_CHAN,
  ...ZERO_CI_CHAN_EXTENSION,
};

const ciPeriodChanZ = v0.ciPeriodChanZ.extend(ciChanExtensionShape);
interface CIPeriodChan extends z.infer<typeof ciPeriodChanZ> {}
const ZERO_CI_PERIOD_CHAN: CIPeriodChan = {
  ...v0.ZERO_CI_PERIOD_CHAN,
  ...ZERO_CI_CHAN_EXTENSION,
};

const ciPulseWidthChanZ = v0.ciPulseWidthChanZ.extend(ciChanExtensionShape);
interface CIPulseWidthChan extends z.infer<typeof ciPulseWidthChanZ> {}
const ZERO_CI_PULSE_WIDTH_CHAN: CIPulseWidthChan = {
  ...v0.ZERO_CI_PULSE_WIDTH_CHAN,
  ...ZERO_CI_CHAN_EXTENSION,
};

const ciSemiPeriodChanZ = v0.ciSemiPeriodChanZ.extend(ciChanExtensionShape);
interface CISemiPeriodChan extends z.infer<typeof ciSemiPeriodChanZ> {}
const ZERO_CI_SEMI_PERIOD_CHAN: CISemiPeriodChan = {
  ...v0.ZERO_CI_SEMI_PERIOD_CHAN,
  ...ZERO_CI_CHAN_EXTENSION,
};

const ciTwoEdgeSepChanZ = v0.ciTwoEdgeSepChanZ.extend(ciChanExtensionShape);
interface CITwoEdgeSepChan extends z.infer<typeof ciTwoEdgeSepChanZ> {}
const ZERO_CI_TWO_EDGE_SEP_CHAN: CITwoEdgeSepChan = {
  ...v0.ZERO_CI_TWO_EDGE_SEP_CHAN,
  ...ZERO_CI_CHAN_EXTENSION,
};

const ciChannelZ = z.union([
  ciFrequencyChanZ,
  ciEdgeCountChanZ,
  ciPeriodChanZ,
  ciPulseWidthChanZ,
  ciSemiPeriodChanZ,
  ciTwoEdgeSepChanZ,
]);
export type CIChannel = z.infer<typeof ciChannelZ>;

export const CI_CHANNEL_SCHEMAS: Record<v0.CIChannelType, z.ZodType<CIChannel>> = {
  [v0.CI_FREQUENCY_CHAN_TYPE]: ciFrequencyChanZ,
  [v0.CI_EDGE_COUNT_CHAN_TYPE]: ciEdgeCountChanZ,
  [v0.CI_PERIOD_CHAN_TYPE]: ciPeriodChanZ,
  [v0.CI_PULSE_WIDTH_CHAN_TYPE]: ciPulseWidthChanZ,
  [v0.CI_SEMI_PERIOD_CHAN_TYPE]: ciSemiPeriodChanZ,
  [v0.CI_TWO_EDGE_SEP_CHAN_TYPE]: ciTwoEdgeSepChanZ,
};

export const ZERO_CI_CHANNELS: Record<v0.CIChannelType, CIChannel> = {
  [v0.CI_FREQUENCY_CHAN_TYPE]: ZERO_CI_FREQUENCY_CHAN,
  [v0.CI_EDGE_COUNT_CHAN_TYPE]: ZERO_CI_EDGE_COUNT_CHAN,
  [v0.CI_PERIOD_CHAN_TYPE]: ZERO_CI_PERIOD_CHAN,
  [v0.CI_PULSE_WIDTH_CHAN_TYPE]: ZERO_CI_PULSE_WIDTH_CHAN,
  [v0.CI_SEMI_PERIOD_CHAN_TYPE]: ZERO_CI_SEMI_PERIOD_CHAN,
  [v0.CI_TWO_EDGE_SEP_CHAN_TYPE]: ZERO_CI_TWO_EDGE_SEP_CHAN,
};
export const ZERO_CI_CHANNEL: CIChannel = ZERO_CI_CHANNELS[v0.CI_FREQUENCY_CHAN_TYPE];

// ==================== Counter Output Channels ====================

const coChanExtensionShape = { device: Common.Device.keyZ };
interface COChanExtension extends z.infer<z.ZodObject<typeof coChanExtensionShape>> {}
const ZERO_CO_CHAN_EXTENSION: COChanExtension = { device: "" };

const coPulseOutputChanZ = v0.coPulseOutputChanZ.extend(coChanExtensionShape);
interface COPulseOutputChan extends z.infer<typeof coPulseOutputChanZ> {}
const ZERO_CO_PULSE_OUTPUT_CHAN: COPulseOutputChan = {
  ...v0.ZERO_CO_PULSE_OUTPUT_CHAN,
  ...ZERO_CO_CHAN_EXTENSION,
};

const coChannelZ = z.union([coPulseOutputChanZ]);
export type COChannel = z.infer<typeof coChannelZ>;

export const CO_CHANNEL_SCHEMAS: Record<v0.COChannelType, z.ZodType<COChannel>> = {
  [v0.CO_PULSE_OUTPUT_CHAN_TYPE]: coPulseOutputChanZ,
};

export const ZERO_CO_CHANNELS: Record<v0.COChannelType, COChannel> = {
  [v0.CO_PULSE_OUTPUT_CHAN_TYPE]: ZERO_CO_PULSE_OUTPUT_CHAN,
};
export const ZERO_CO_CHANNEL: COChannel =
  ZERO_CO_CHANNELS[v0.CO_PULSE_OUTPUT_CHAN_TYPE];

export type AnalogChannel = AIChannel | v0.AOChannel;

export type Channel = AnalogChannel | v0.DigitalChannel | CIChannel | COChannel;

const baseAnalogReadConfigZ = v0.baseAnalogReadConfigZ
  .omit({ channels: true, device: true })
  .extend({
    channels: z
      .array(aiChannelZ)
      .check(Common.Task.validateReadChannels)
      .check(validateAnalogPorts),
  })
  .check(Common.Task.validateStreamRate);
export interface AnalogReadConfig extends z.infer<typeof baseAnalogReadConfigZ> {}
export const analogReadConfigZ = z.union([
  v0.analogReadConfigZ.transform<AnalogReadConfig>(({ channels, device, ...rest }) => ({
    ...rest,
    channels: channels.map((c) => ({ ...c, device })),
  })),
  baseAnalogReadConfigZ,
]);
const { device: _, ...rest } = v0.ZERO_ANALOG_READ_CONFIG;
const ZERO_ANALOG_READ_CONFIG: AnalogReadConfig = {
  ...rest,
  channels: [],
};

export interface AnalogReadPayload
  extends task.Payload<
    typeof v0.analogReadTypeZ,
    typeof analogReadConfigZ,
    typeof v0.analogReadStatusDataZ
  > {}
export const ZERO_ANALOG_READ_PAYLOAD: AnalogReadPayload = {
  ...v0.ZERO_ANALOG_READ_PAYLOAD,
  config: ZERO_ANALOG_READ_CONFIG,
};

export interface AnalogReadTask
  extends task.Task<
    typeof v0.analogReadTypeZ,
    typeof analogReadConfigZ,
    typeof v0.analogReadStatusDataZ
  > {}
export interface NewAnalogReadTask
  extends task.New<typeof v0.analogReadTypeZ, typeof analogReadConfigZ> {}

// ==================== Counter Read Task ====================

const validateCounterPorts = ({
  value: channels,
  issues,
}: z.core.ParsePayload<CIChannel[]>) => {
  const deviceToPortMap = new Map<device.Key, PortToIndexMap>();
  channels.forEach(({ device, port }, i) => {
    if (!deviceToPortMap.has(device)) deviceToPortMap.set(device, new Map());
    const portToIndexMap = deviceToPortMap.get(device) as PortToIndexMap;
    if (!portToIndexMap.has(port)) {
      portToIndexMap.set(port, i);
      return;
    }
    const index = portToIndexMap.get(port) as number;
    const code = "custom";
    const message = `Counter port ${port} has already been used on another channel on the same device`;
    issues.push({ path: [index, "port"], code, message, input: channels });
    issues.push({ path: [i, "port"], code, message, input: channels });
  });
};

const baseCounterReadConfigZ = v0.counterReadConfigZ
  .omit({ channels: true, device: true })
  .extend({
    channels: z
      .array(ciChannelZ)
      .check(Common.Task.validateReadChannels)
      .check(validateCounterPorts),
  })
  .check(Common.Task.validateStreamRate);
export interface CounterReadConfig extends z.infer<typeof baseCounterReadConfigZ> {}
export const counterReadConfigZ = z.union([
  v0.counterReadConfigZ.transform<CounterReadConfig>(
    ({ channels, device, ...rest }) => ({
      ...rest,
      channels: channels.map((c) => ({ ...c, device })),
    }),
  ),
  baseCounterReadConfigZ,
]);
const { device: _counterDevice, ...counterRest } = v0.ZERO_COUNTER_READ_CONFIG;
const ZERO_COUNTER_READ_CONFIG: CounterReadConfig = {
  ...counterRest,
  channels: [],
};

export interface CounterReadPayload
  extends task.Payload<
    typeof v0.counterReadTypeZ,
    typeof counterReadConfigZ,
    typeof v0.counterReadStatusDataZ
  > {}
export const ZERO_COUNTER_READ_PAYLOAD: CounterReadPayload = {
  ...v0.ZERO_COUNTER_READ_PAYLOAD,
  config: ZERO_COUNTER_READ_CONFIG,
};

export interface CounterReadTask
  extends task.Task<
    typeof v0.counterReadTypeZ,
    typeof counterReadConfigZ,
    typeof v0.counterReadStatusDataZ
  > {}
export interface NewCounterReadTask
  extends task.New<typeof v0.counterReadTypeZ, typeof counterReadConfigZ> {}

// ==================== Counter Write Config Migration ====================

const validateCounterWritePorts = ({
  value: channels,
  issues,
}: z.core.ParsePayload<COChannel[]>) => {
  const devicePortMap = new Map<string, Map<number, number>>();
  channels.forEach(({ port, device }, i) => {
    if (!devicePortMap.has(device)) {
      devicePortMap.set(device, new Map([[port, i]]));
      return;
    }
    const portToIndexMap = devicePortMap.get(device)!;
    if (!portToIndexMap.has(port)) {
      portToIndexMap.set(port, i);
      return;
    }
    const index = portToIndexMap.get(port) as number;
    const code = "custom";
    const message = `Counter port ${port} has already been used on another channel on the same device`;
    issues.push({ path: [index, "port"], code, message, input: channels });
    issues.push({ path: [i, "port"], code, message, input: channels });
  });
};

const baseCounterWriteConfigZ = v0.counterWriteConfigZ
  .omit({ channels: true, device: true })
  .extend({
    channels: z
      .array(coChannelZ)
      .check(Common.Task.validateWriteChannels)
      .check(validateCounterWritePorts),
  });
export interface CounterWriteConfig extends z.infer<typeof baseCounterWriteConfigZ> {}
export const counterWriteConfigZ = z.union([
  v0.counterWriteConfigZ.transform<CounterWriteConfig>(
    ({ channels, device, ...rest }) => ({
      ...rest,
      channels: channels.map((c) => ({ ...c, device })),
    }),
  ),
  baseCounterWriteConfigZ,
]);
const ZERO_COUNTER_WRITE_CONFIG: CounterWriteConfig = {
  ...Common.Task.ZERO_BASE_CONFIG,
  dataSaving: true,
  stateRate: 10,
  channels: [],
};

export interface CounterWritePayload
  extends task.Payload<
    typeof v0.counterWriteTypeZ,
    typeof counterWriteConfigZ,
    typeof v0.counterWriteStatusDataZ
  > {}
export const ZERO_COUNTER_WRITE_PAYLOAD: CounterWritePayload = {
  ...v0.ZERO_COUNTER_WRITE_PAYLOAD,
  config: ZERO_COUNTER_WRITE_CONFIG,
};

export interface CounterWriteTask
  extends task.Task<
    typeof v0.counterWriteTypeZ,
    typeof counterWriteConfigZ,
    typeof v0.counterWriteStatusDataZ
  > {}
export interface NewCounterWriteTask
  extends task.New<typeof v0.counterWriteTypeZ, typeof counterWriteConfigZ> {}
