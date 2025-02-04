// Copyright 2025 Synnax Labs, Inc.
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

import * as v0 from "@/hardware/ni/task/types/v0";

const VERSION = "1.0.0";

const aiChanExtensionZ = z.object({ device: device.deviceKeyZ });
interface AIChanExtension extends z.infer<typeof aiChanExtensionZ> {}
const ZERO_AI_CHAN_EXTENSION: AIChanExtension = { device: "" };

const aiAccelChanZ = v0.aiAccelChanZ.merge(aiChanExtensionZ);
interface AIAccelChan extends z.infer<typeof aiAccelChanZ> {}
const ZERO_AI_ACCEL_CHAN: AIAccelChan = {
  ...v0.ZERO_AI_ACCEL_CHAN,
  ...ZERO_AI_CHAN_EXTENSION,
};

const aiBridgeChanZ = v0.aiBridgeChanZ.merge(aiChanExtensionZ);
interface AIBridgeChan extends z.infer<typeof aiBridgeChanZ> {}
const ZERO_AI_BRIDGE_CHAN: AIBridgeChan = {
  ...v0.ZERO_AI_BRIDGE_CHAN,
  ...ZERO_AI_CHAN_EXTENSION,
};

const aiCurrentChanZ = v0.aiCurrentChanZ.merge(aiChanExtensionZ);
interface AICurrentChan extends z.infer<typeof aiCurrentChanZ> {}
const ZERO_AI_CURRENT_CHAN: AICurrentChan = {
  ...v0.ZERO_AI_CURRENT_CHAN,
  ...ZERO_AI_CHAN_EXTENSION,
};

const aiForceBridgeTableChanZ = v0.aiForceBridgeTableChanZ.merge(aiChanExtensionZ);
interface AIForceBridgeTableChan extends z.infer<typeof aiForceBridgeTableChanZ> {}
const ZERO_AI_FORCE_BRIDGE_TABLE_CHAN: AIForceBridgeTableChan = {
  ...v0.ZERO_AI_FORCE_BRIDGE_TABLE_CHAN,
  ...ZERO_AI_CHAN_EXTENSION,
};

const aiForceBridgeTwoPointLinChan =
  v0.aiForceBridgeTwoPointLinChan.merge(aiChanExtensionZ);
interface AIForceBridgeTwoPointLinChan
  extends z.infer<typeof aiForceBridgeTwoPointLinChan> {}
const ZERO_AI_FORCE_BRIDGE_TWO_POINT_LIN_CHAN: AIForceBridgeTwoPointLinChan = {
  ...v0.ZERO_AI_FORCE_BRIDGE_TWO_POINT_LIN_CHAN,
  ...ZERO_AI_CHAN_EXTENSION,
};

const aiForceIEPEChanZ = v0.aiForceIEPEChanZ.merge(aiChanExtensionZ);
interface AIForceIEPEChan extends z.infer<typeof aiForceIEPEChanZ> {}
const ZERO_AI_FORCE_IEPE_CHAN: AIForceIEPEChan = {
  ...v0.ZERO_AI_FORCE_IEPE_CHAN,
  ...ZERO_AI_CHAN_EXTENSION,
};

const aiMicrophoneChanZ = v0.aiMicrophoneChanZ.merge(aiChanExtensionZ);
interface AIMicrophoneChan extends z.infer<typeof aiMicrophoneChanZ> {}
const ZERO_AI_MICROPHONE_CHAN: AIMicrophoneChan = {
  ...v0.ZERO_AI_MICROPHONE_CHAN,
  ...ZERO_AI_CHAN_EXTENSION,
};

const aiPressureBridgeTableChanZ =
  v0.aiPressureBridgeTableChanZ.merge(aiChanExtensionZ);
interface AIPressureBridgeTableChan
  extends z.infer<typeof aiPressureBridgeTableChanZ> {}
const ZERO_AI_PRESSURE_BRIDGE_TABLE_CHAN: AIPressureBridgeTableChan = {
  ...v0.ZERO_AI_PRESSURE_BRIDGE_TABLE_CHAN,
  ...ZERO_AI_CHAN_EXTENSION,
};

const aiPressureBridgeTwoPointLinChanZ =
  v0.aiPressureBridgeTwoPointLinChanZ.merge(aiChanExtensionZ);
interface AIPressureBridgeTwoPointLinChan
  extends z.infer<typeof aiPressureBridgeTwoPointLinChanZ> {}
const ZERO_AI_PRESSURE_BRIDGE_TWO_POINT_LIN_CHAN: AIPressureBridgeTwoPointLinChan = {
  ...v0.ZERO_AI_PRESSURE_BRIDGE_TWO_POINT_LIN_CHAN,
  ...ZERO_AI_CHAN_EXTENSION,
};

const aiResistanceChanZ = v0.aiResistanceChanZ.merge(aiChanExtensionZ);
interface AIResistanceChan extends z.infer<typeof aiResistanceChanZ> {}
const ZERO_AI_RESISTANCE_CHAN: AIResistanceChan = {
  ...v0.ZERO_AI_RESISTANCE_CHAN,
  ...ZERO_AI_CHAN_EXTENSION,
};

const aiRTDChanZ = v0.aiRTDChanZ.merge(aiChanExtensionZ);
interface AIRTDChan extends z.infer<typeof aiRTDChanZ> {}
const ZERO_AI_RTD_CHAN: AIRTDChan = {
  ...v0.ZERO_AI_RTD_CHAN,
  ...ZERO_AI_CHAN_EXTENSION,
};

const aiStrainGageChan = v0.aiStrainGageChanZ.merge(aiChanExtensionZ);
interface AIStrainGageChan extends z.infer<typeof aiStrainGageChan> {}
const ZERO_AI_STRAIN_GAGE_CHAN: AIStrainGageChan = {
  ...v0.ZERO_AI_STRAIN_GAGE_CHAN,
  ...ZERO_AI_CHAN_EXTENSION,
};

const aiTempBuiltInChanZ = v0.aiTempBuiltInChanZ.merge(aiChanExtensionZ);
interface AITempBuiltInChan extends z.infer<typeof aiTempBuiltInChanZ> {}
const ZERO_AI_TEMP_BUILT_IN_CHAN: AITempBuiltInChan = {
  ...v0.ZERO_AI_TEMP_BUILT_IN_CHAN,
  ...ZERO_AI_CHAN_EXTENSION,
};

const aiThrmcplChanZ = v0.aiThrmcplChanZ
  .merge(aiChanExtensionZ)
  .refine(
    (v) => {
      if (v.cjcSource === v0.CONST_VAL) return v.cjcVal !== undefined;
      return true;
    },
    {
      path: ["cjcVal"],
      message: `CJC Value must be defined when CJC Source is ${v0.CONST_VAL}`,
    },
  )
  .refine(
    (v) => {
      if (v.cjcSource === v0.CHAN) return v.cjcPort !== undefined;
      return true;
    },
    {
      path: ["cjcPort"],
      message: `CJC Port must be defined when CJC Source is ${v0.CHAN}`,
    },
  );
interface AIThrmcplChan extends z.infer<typeof aiThrmcplChanZ> {}
const ZERO_AI_THRMCPL_CHAN: AIThrmcplChan = {
  ...v0.ZERO_AI_THRMCPL_CHAN,
  ...ZERO_AI_CHAN_EXTENSION,
};

const aiTorqueBridgeTableChanZ = v0.aiTorqueBridgeTableChanZ.merge(aiChanExtensionZ);
interface AITorqueBridgeTableChan extends z.infer<typeof aiTorqueBridgeTableChanZ> {}
const ZERO_AI_TORQUE_BRIDGE_TABLE_CHAN: AITorqueBridgeTableChan = {
  ...v0.ZERO_AI_TORQUE_BRIDGE_TABLE_CHAN,
  ...ZERO_AI_CHAN_EXTENSION,
};

const aiTorqueBridgeTwoPointLinChanZ =
  v0.aiTorqueBridgeTwoPointLinChanZ.merge(aiChanExtensionZ);
interface AITorqueBridgeTwoPointLinChan
  extends z.infer<typeof aiTorqueBridgeTwoPointLinChanZ> {}
const ZERO_AI_TORQUE_BRIDGE_TWO_POINT_LIN_CHAN: AITorqueBridgeTwoPointLinChan = {
  ...v0.ZERO_AI_TORQUE_BRIDGE_TWO_POINT_LIN_CHAN,
  ...ZERO_AI_CHAN_EXTENSION,
};

const aiVelocityIEPEChanZ = v0.aiVelocityIEPEChanZ.merge(aiChanExtensionZ);
interface AIVelocityIEPEChan extends z.infer<typeof aiVelocityIEPEChanZ> {}
const ZERO_AI_VELOCITY_IEPE_CHAN: AIVelocityIEPEChan = {
  ...v0.ZERO_AI_VELOCITY_IEPE_CHAN,
  ...ZERO_AI_CHAN_EXTENSION,
};

const aiVoltageChanZ = v0.aiVoltageChanZ.merge(aiChanExtensionZ);
interface AIVoltageChan extends z.infer<typeof aiVoltageChanZ> {}
const ZERO_AI_VOLTAGE_CHAN: AIVoltageChan = {
  ...v0.ZERO_AI_VOLTAGE_CHAN,
  ...ZERO_AI_CHAN_EXTENSION,
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
  aiStrainGageChan,
  aiTempBuiltInChanZ,
  aiThrmcplChanZ,
  aiTorqueBridgeTableChanZ,
  aiTorqueBridgeTwoPointLinChanZ,
  aiVelocityIEPEChanZ,
  aiVoltageChanZ,
]);

export type AIChan = z.infer<typeof aiChanZ>;

export const AI_CHANNEL_SCHEMAS: Record<v0.AIChanType, z.ZodType<AIChan>> = {
  [v0.AI_ACCEL_CHAN_TYPE]: aiAccelChanZ,
  [v0.AI_BRIDGE_CHAN_TYPE]: aiBridgeChanZ,
  [v0.AI_CURRENT_CHAN_TYPE]: aiCurrentChanZ,
  [v0.AI_FORCE_BRIDGE_TABLE_CHAN_TYPE]: aiForceBridgeTableChanZ,
  [v0.AI_FORCE_BRIDGE_TWO_POINT_LIN_CHAN_TYPE]: aiForceBridgeTwoPointLinChan,
  [v0.AI_FORCE_IEPE_CHAN_TYPE]: aiForceIEPEChanZ,
  [v0.AI_MICROPHONE_CHAN_TYPE]: aiMicrophoneChanZ,
  [v0.AI_PRESSURE_BRIDGE_TABLE_CHAN_TYPE]: aiPressureBridgeTableChanZ,
  [v0.AI_PRESSURE_BRIDGE_TWO_POINT_LIN_CHAN_TYPE]: aiPressureBridgeTwoPointLinChanZ,
  [v0.AI_RESISTANCE_CHAN_TYPE]: aiResistanceChanZ,
  [v0.AI_RTD_CHAN_TYPE]: aiRTDChanZ,
  [v0.AI_STRAIN_GAGE_CHAN_TYPE]: aiStrainGageChan,
  [v0.AI_TEMP_BUILT_IN_CHAN_TYPE]: aiTempBuiltInChanZ,
  [v0.AI_THRMCPL_CHAN_TYPE]: aiThrmcplChanZ,
  [v0.AI_TORQUE_BRIDGE_TABLE_CHAN_TYPE]: aiTorqueBridgeTableChanZ,
  [v0.AI_TORQUE_BRIDGE_TWO_POINT_LIN_CHAN_TYPE]: aiTorqueBridgeTwoPointLinChanZ,
  [v0.AI_VELOCITY_IEPE_CHAN_TYPE]: aiVelocityIEPEChanZ,
  [v0.AI_VOLTAGE_CHAN_TYPE]: aiVoltageChanZ,
};

export const ZERO_AI_CHANNELS: Record<v0.AIChanType, AIChan> = {
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

export type Chan = v0.DIChan | AIChan | v0.DOChan;

export const analogReadConfigZ = v0.analogReadConfigZ
  .omit({ version: true, device: true, channels: true })
  .extend({ version: z.literal(VERSION), channels: z.array(aiChanZ) })
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
export interface AnalogReadConfig extends z.infer<typeof analogReadConfigZ> {}
const { device: _, ...rest } = v0.ZERO_ANALOG_READ_CONFIG;
export const ZERO_ANALOG_READ_CONFIG: AnalogReadConfig = {
  ...rest,
  version: VERSION,
  channels: [],
};

export interface AnalogRead
  extends task.Task<AnalogReadConfig, v0.AnalogReadDetails, v0.AnalogReadType> {}
export interface AnalogReadPayload
  extends task.Payload<AnalogReadConfig, v0.AnalogReadDetails, v0.AnalogReadType> {}
export const ZERO_ANALOG_READ_PAYLOAD: AnalogReadPayload = {
  ...v0.ZERO_ANALOG_READ_PAYLOAD,
  config: ZERO_ANALOG_READ_CONFIG,
};

export const ANALOG_READ_CONFIG_MIGRATION_NAME = "hardware.ni.task.analogRead.config";

export const analogReadConfigMigration = migrate.createMigration<
  v0.AnalogReadConfig,
  AnalogReadConfig
>({
  name: ANALOG_READ_CONFIG_MIGRATION_NAME,
  migrate: (s) => {
    const { device, ...rest } = s;
    return {
      ...rest,
      version: VERSION,
      channels: rest.channels.map((c) => ({ device, ...c })),
    };
  },
});
