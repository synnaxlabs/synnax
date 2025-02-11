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

export const PREFIX = "labjack";

// Channels

const baseChannelZ = z.object({
  port: z.string().min(1, "Port must be specified"),
  enabled: z.boolean(),
  key: z.string().min(1, "Key must be specified"),
});
interface BaseChannel extends z.infer<typeof baseChannelZ> {}
const ZERO_BASE_CHANNEL: BaseChannel = { port: "", enabled: true, key: "" };

// Input Channels

const baseInputChannelZ = baseChannelZ.extend({ channel: channel.keyZ });
interface BaseInputChannel extends z.infer<typeof baseInputChannelZ> {}
const ZERO_BASE_INPUT_CHANNEL: BaseInputChannel = {
  ...ZERO_BASE_CHANNEL,
  channel: 0,
};

// Analog Input Channels

export const LINEAR_SCALE_TYPE = "linear";
export type LinearScaleType = typeof LINEAR_SCALE_TYPE;

const linearScaleZ = z.object({
  type: z.literal(LINEAR_SCALE_TYPE),
  slope: z.number().finite(),
  offset: z.number().finite(),
});
interface LinearScale extends z.infer<typeof linearScaleZ> {}
const ZERO_LINEAR_SCALE: LinearScale = { type: LINEAR_SCALE_TYPE, slope: 1, offset: 0 };

export const NO_SCALE_TYPE = "none";
export type NoScaleType = typeof NO_SCALE_TYPE;

const noScaleZ = z.object({ type: z.literal(NO_SCALE_TYPE) });
interface NoScale extends z.infer<typeof noScaleZ> {}
const NO_SCALE: NoScale = { type: NO_SCALE_TYPE };

const scaleZ = z.union([noScaleZ, linearScaleZ]);
export type Scale = z.infer<typeof scaleZ>;
export type ScaleType = Scale["type"];
export const SCALE_SCHEMAS: Record<ScaleType, z.ZodType<Scale>> = {
  [NO_SCALE_TYPE]: noScaleZ,
  [LINEAR_SCALE_TYPE]: linearScaleZ,
};
export const ZERO_SCALES: Record<ScaleType, Scale> = {
  [NO_SCALE_TYPE]: NO_SCALE,
  [LINEAR_SCALE_TYPE]: ZERO_LINEAR_SCALE,
};

export const AI_CHANNEL_TYPE = "AI";
export type AIChannelType = typeof AI_CHANNEL_TYPE;

const aiChannelZ = baseInputChannelZ.extend({
  type: z.literal(AI_CHANNEL_TYPE),
  range: z.number().finite().optional(),
  scale: scaleZ,
});
interface AIChannel extends z.infer<typeof aiChannelZ> {}
const ZERO_AI_CHANNEL: AIChannel = {
  ...ZERO_BASE_INPUT_CHANNEL,
  type: AI_CHANNEL_TYPE,
  port: "AIN0",
  range: 0,
  scale: ZERO_SCALES[NO_SCALE_TYPE],
};

// Digital Input Channels

export const DI_CHANNEL_TYPE = "DI";
export type DIChannelType = typeof DI_CHANNEL_TYPE;

const diChannelZ = baseInputChannelZ.extend({
  type: z.literal(DI_CHANNEL_TYPE),
});
interface DIChannel extends z.infer<typeof diChannelZ> {}
const ZERO_DI_CHANNEL: DIChannel = {
  ...ZERO_BASE_INPUT_CHANNEL,
  port: "DIO4",
  type: DI_CHANNEL_TYPE,
};

// Thermocouple Channels

export const TC_CHANNEL_TYPE = "TC";
export type TCChannelType = typeof TC_CHANNEL_TYPE;

export const CELSIUS_UNIT = "C";
export const FAHRENHEIT_UNIT = "F";
export const KELVIN_UNIT = "K";
const temperatureUnitsZ = z.enum([CELSIUS_UNIT, FAHRENHEIT_UNIT, KELVIN_UNIT]);
export type TemperatureUnits = z.infer<typeof temperatureUnitsZ>;

export const J_TC_TYPE = "J";
export const K_TC_TYPE = "K";
export const N_TC_TYPE = "N";
export const R_TC_TYPE = "R";
export const S_TC_TYPE = "S";
export const T_TC_TYPE = "T";
export const B_TC_TYPE = "B";
export const E_TC_TYPE = "E";
export const C_TC_TYPE = "C";
const thermocoupleTypeZ = z.enum([
  J_TC_TYPE,
  K_TC_TYPE,
  N_TC_TYPE,
  R_TC_TYPE,
  S_TC_TYPE,
  T_TC_TYPE,
  B_TC_TYPE,
  E_TC_TYPE,
  C_TC_TYPE,
]);
export type ThermocoupleType = z.infer<typeof thermocoupleTypeZ>;

export const DEVICE_CJC_SOURCE = "TEMPERATURE_DEVICE_K";
export const AIR_CJC_SOURCE = "TEMPERATURE_AIR_K";

const tcChannelZ = aiChannelZ.omit({ type: true }).extend({
  type: z.literal(TC_CHANNEL_TYPE),
  thermocoupleType: thermocoupleTypeZ,
  posChan: z.number().int(),
  negChan: z.number().int(),
  cjcSource: z.string().min(1, "CJC Source must be specified"),
  cjcSlope: z.number().finite(),
  cjcOffset: z.number().finite(),
  units: temperatureUnitsZ,
});
interface TCChannel extends z.infer<typeof tcChannelZ> {}
const ZERO_TC_CHANNEL: TCChannel = {
  ...ZERO_AI_CHANNEL,
  type: TC_CHANNEL_TYPE,
  thermocoupleType: KELVIN_UNIT,
  posChan: 0,
  negChan: 199,
  units: "K",
  cjcSource: DEVICE_CJC_SOURCE,
  cjcSlope: 1,
  cjcOffset: 0,
  scale: NO_SCALE,
};

const inputChannelZ = z.union([aiChannelZ, diChannelZ, tcChannelZ]);
export type InputChannel = z.infer<typeof inputChannelZ>;
export type InputChannelType = InputChannel["type"];
export const INPUT_CHANNEL_SCHEMAS: Record<
  InputChannelType,
  z.ZodType<InputChannel>
> = {
  [AI_CHANNEL_TYPE]: aiChannelZ,
  [DI_CHANNEL_TYPE]: diChannelZ,
  [TC_CHANNEL_TYPE]: tcChannelZ,
};
export const ZERO_INPUT_CHANNELS: Record<InputChannelType, InputChannel> = {
  [AI_CHANNEL_TYPE]: ZERO_AI_CHANNEL,
  [DI_CHANNEL_TYPE]: ZERO_DI_CHANNEL,
  [TC_CHANNEL_TYPE]: ZERO_TC_CHANNEL,
};
export const ZERO_INPUT_CHANNEL: InputChannel = ZERO_INPUT_CHANNELS[AI_CHANNEL_TYPE];

// Output Channels

export const AO_CHANNEL_TYPE = "AO";
export type AOChannelType = typeof AO_CHANNEL_TYPE;

export const DO_CHANNEL_TYPE = "DO";
export type DOChannelType = typeof DO_CHANNEL_TYPE;

export const outputChannelTypeZ = z.enum([AO_CHANNEL_TYPE, DO_CHANNEL_TYPE]);
export type OutputChannelType = z.infer<typeof outputChannelTypeZ>;

const outputChannelZ = baseChannelZ.extend({
  type: outputChannelTypeZ,
  cmdKey: channel.keyZ,
  stateKey: channel.keyZ,
});
export interface OutputChannel extends z.infer<typeof outputChannelZ> {}
export const ZERO_OUTPUT_CHANNEL: OutputChannel = {
  ...ZERO_BASE_CHANNEL,
  port: "DIO4",
  cmdKey: 0,
  stateKey: 0,
  type: DO_CHANNEL_TYPE,
};

export type Channel = InputChannel | OutputChannel;

// Tasks

const deviceKeyZ = device.keyZ.min(1, "Must specify a device");

export interface BaseStateDetails {
  running: boolean;
}

// Read Tasks

export const readConfigZ = z
  .object({
    device: deviceKeyZ,
    sampleRate: z.number().min(0).max(50000),
    streamRate: z.number().min(0).max(50000),
    channels: z
      .array(inputChannelZ)
      .refine((channels) => channels.some(({ enabled }) => enabled), {
        message: "At least one channel must be enabled",
      })
      .superRefine((channels, { addIssue }) => {
        const portSet = new Set<string>();
        channels.forEach(({ port }, i) => {
          if (portSet.has(port))
            addIssue({
              code: z.ZodIssueCode.custom,
              path: [i, "port"],
              message: `Port ${port} has already been used on another channel`,
            });
          else portSet.add(port);
        });
      })
      .superRefine((channels, { addIssue }) => {
        const channelSet = new Set<number>();
        channels.forEach(({ channel }, i) => {
          if (channel === 0) return;
          if (channelSet.has(channel))
            addIssue({
              code: z.ZodIssueCode.custom,
              path: [i, "channel"],
              message: `Synnax channel with key ${channel} is being used elsewhere in the configuration`,
            });
          else channelSet.add(channel);
        });
      })
      .superRefine((channels, { addIssue }) => {
        const keySet = new Set<string>();
        channels.forEach(({ key }, i) => {
          if (keySet.has(key))
            addIssue({
              code: z.ZodIssueCode.custom,
              path: [i, "key"],
              message: `Key ${key} has already been used on another channel`,
            });
          else keySet.add(key);
        });
      }),
    dataSaving: z.boolean(),
  })
  .refine(({ sampleRate, streamRate }) => sampleRate >= streamRate, {
    path: ["streamRate"],
    message: "Stream rate must be less than or equal to the sample rate",
  });
export interface ReadConfig extends z.infer<typeof readConfigZ> {}
const ZERO_READ_CONFIG: ReadConfig = {
  device: "",
  sampleRate: 10,
  streamRate: 5,
  channels: [],
  dataSaving: true,
};

export interface ReadStateDetails extends BaseStateDetails {
  message: string;
  errors?: { message: string; path: string }[];
}

export const READ_TYPE = `${PREFIX}_read`;
export type ReadType = typeof READ_TYPE;

export interface ReadPayload
  extends task.Payload<ReadConfig, ReadStateDetails, ReadType> {}
export const ZERO_READ_PAYLOAD: ReadPayload = {
  key: "",
  name: "LabJack Read Task",
  config: ZERO_READ_CONFIG,
  type: READ_TYPE,
};

export interface ReadTask extends task.Task<ReadConfig, ReadStateDetails, ReadType> {}

// Write Tasks

export const writeConfigZ = z.object({
  device: deviceKeyZ,
  channels: z
    .array(outputChannelZ)
    .refine((channels) => channels.some(({ enabled }) => enabled), {
      message: "At least one channel must be enabled",
    })
    .superRefine((channels, { addIssue }) => {
      const portSet = new Set<string>();
      channels.forEach(({ port }, i) => {
        if (portSet.has(port))
          addIssue({
            code: z.ZodIssueCode.custom,
            path: [i, "port"],
            message: `Port ${port} has already been used on another channel`,
          });
        else portSet.add(port);
      });
    })
    .superRefine((channels, { addIssue }) => {
      const cmdChannelSet = new Set<number>();
      const stateChannelSet = new Set<number>();
      channels.forEach(({ cmdKey, stateKey }, i) => {
        if (cmdKey !== 0)
          if (cmdChannelSet.has(cmdKey))
            addIssue({
              code: z.ZodIssueCode.custom,
              path: [i, "cmdKey"],
              message: `Command channel with key ${cmdKey} is being used elsewhere in the configuration`,
            });
          else cmdChannelSet.add(cmdKey);
        if (stateKey === 0) return;
        if (stateChannelSet.has(stateKey))
          addIssue({
            code: z.ZodIssueCode.custom,
            path: [i, "stateKey"],
            message: `State channel with key ${stateKey} is being used elsewhere in the configuration`,
          });
        else stateChannelSet.add(stateKey);
      });
    })
    .superRefine((channels, { addIssue }) => {
      const keySet = new Set<string>();
      channels.forEach(({ key }, i) => {
        if (keySet.has(key))
          addIssue({
            code: z.ZodIssueCode.custom,
            path: [i, "key"],
            message: `Key ${key} has already been used on another channel`,
          });
        else keySet.add(key);
      });
    }),
  dataSaving: z.boolean(),
  stateRate: z.number().min(1).max(50000),
});
export interface WriteConfig extends z.infer<typeof writeConfigZ> {}
const ZERO_WRITE_CONFIG: WriteConfig = {
  device: "",
  channels: [],
  dataSaving: true,
  stateRate: 10,
};

export interface WriteStateDetails extends BaseStateDetails {}

export const WRITE_TYPE = `${PREFIX}_write`;
export type WriteType = typeof WRITE_TYPE;

export interface WritePayload
  extends task.Payload<WriteConfig, WriteStateDetails, WriteType> {}
export const ZERO_WRITE_PAYLOAD: WritePayload = {
  key: "",
  name: "LabJack Write Task",
  config: ZERO_WRITE_CONFIG,
  type: WRITE_TYPE,
};

export interface WriteTask
  extends task.Task<WriteConfig, WriteStateDetails, WriteType> {}
