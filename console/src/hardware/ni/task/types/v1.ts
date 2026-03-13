// Copyright 2026 Synnax Labs, Inc.
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
import * as v0 from "@/hardware/ni/task/types/v0";
import { createPortValidator } from "@/hardware/ni/task/types/validation";

const validateAnalogPorts = createPortValidator();

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

interface AIForceBridgeTwoPointLinChan extends z.infer<
  typeof aiForceBridgeTwoPointLinChanZ
> {}

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

interface AIPressureBridgeTableChan extends z.infer<
  typeof aiPressureBridgeTableChanZ
> {}

const ZERO_AI_PRESSURE_BRIDGE_TABLE_CHAN: AIPressureBridgeTableChan = {
  ...v0.ZERO_AI_PRESSURE_BRIDGE_TABLE_CHAN,
  ...ZERO_AI_CHAN_EXTENSION,
};

const aiPressureBridgeTwoPointLinChanZ =
  v0.aiPressureBridgeTwoPointLinChanZ.extend(aiChanExtensionShape);

interface AIPressureBridgeTwoPointLinChan extends z.infer<
  typeof aiPressureBridgeTwoPointLinChanZ
> {}

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

interface AITorqueBridgeTwoPointLinChan extends z.infer<
  typeof aiTorqueBridgeTwoPointLinChanZ
> {}

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
  ai_accel: aiAccelChanZ,
  ai_bridge: aiBridgeChanZ,
  ai_current: aiCurrentChanZ,
  ai_force_bridge_table: aiForceBridgeTableChanZ,
  ai_force_bridge_two_point_lin: aiForceBridgeTwoPointLinChanZ,
  ai_force_iepe: aiForceIEPEChanZ,
  ai_microphone: aiMicrophoneChanZ,
  ai_pressure_bridge_table: aiPressureBridgeTableChanZ,
  ai_pressure_bridge_two_point_lin: aiPressureBridgeTwoPointLinChanZ,
  ai_resistance: aiResistanceChanZ,
  ai_rtd: aiRTDChanZ,
  ai_strain_gauge: aiStrainGageChanZ,
  ai_temp_builtin: aiTempBuiltInChanZ,
  ai_thermocouple: aiThrmcplChanZ,
  ai_torque_bridge_table: aiTorqueBridgeTableChanZ,
  ai_torque_bridge_two_point_lin: aiTorqueBridgeTwoPointLinChanZ,
  ai_velocity_iepe: aiVelocityIEPEChanZ,
  ai_voltage: aiVoltageChanZ,
};

export const ZERO_AI_CHANNELS: Record<v0.AIChannelType, AIChannel> = {
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
  ai_temp_builtin: ZERO_AI_TEMP_BUILT_IN_CHAN,
  ai_thermocouple: ZERO_AI_THRMCPL_CHAN,
  ai_torque_bridge_table: ZERO_AI_TORQUE_BRIDGE_TABLE_CHAN,
  ai_torque_bridge_two_point_lin: ZERO_AI_TORQUE_BRIDGE_TWO_POINT_LIN_CHAN,
  ai_velocity_iepe: ZERO_AI_VELOCITY_IEPE_CHAN,
  ai_voltage: ZERO_AI_VOLTAGE_CHAN,
};

export const ZERO_AI_CHANNEL = ZERO_AI_CHANNELS.ai_voltage;

export type AnalogChannel = AIChannel | v0.AOChannel;

export type Channel = AnalogChannel | v0.DigitalChannel;

const v1AnalogReadConfigZ = v0.baseAnalogReadConfigZ
  .omit({ channels: true, device: true })
  .extend({
    channels: z
      .array(aiChannelZ)
      .check(Common.Task.validateReadChannels)
      .check(validateAnalogPorts),
  })
  .check(Common.Task.validateStreamRate);

export interface AnalogReadConfig extends z.infer<typeof v1AnalogReadConfigZ> {}

export const analogReadConfigZ = z.union([
  v0.analogReadConfigZ.transform<AnalogReadConfig>(({ channels, device, ...rest }) => ({
    ...rest,
    channels: channels.map((c) => ({ ...c, device })),
  })),
  v1AnalogReadConfigZ,
]);

const { device: _, ...rest } = v0.ZERO_ANALOG_READ_CONFIG;

const ZERO_ANALOG_READ_CONFIG: AnalogReadConfig = { ...rest, channels: [] };

export const ANALOG_READ_SCHEMAS = {
  ...v0.ANALOG_READ_SCHEMAS,
  config: analogReadConfigZ,
} as const satisfies task.Schemas;

export type AnalogReadSchemas = typeof ANALOG_READ_SCHEMAS;

export interface AnalogReadPayload extends task.Payload<AnalogReadSchemas> {}

export const ZERO_ANALOG_READ_PAYLOAD = {
  ...v0.ZERO_ANALOG_READ_PAYLOAD,
  config: ZERO_ANALOG_READ_CONFIG,
} as const satisfies AnalogReadPayload;
