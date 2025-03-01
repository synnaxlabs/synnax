// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, type task } from "@synnaxlabs/client";
import { z } from "zod";

import { Common } from "@/hardware/common";
import { Device } from "@/hardware/labjack/device";

export const PREFIX = "labjack";

// Base Channels

const portZ = z.string().min(1, "Port must be specified");

// Digital Channels

const digitalPortZ = portZ.regex(
  Device.DIO_PORT_REGEX,
  "Invalid port, port must start with DIO and end with an integer",
);

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

const aiChannelZ = Common.Task.readChannelZ.extend({
  type: z.literal(AI_CHANNEL_TYPE),
  range: z.number().finite().optional(),
  scale: scaleZ,
  port: digitalPortZ,
});
interface AIChannel extends z.infer<typeof aiChannelZ> {}
const ZERO_AI_CHANNEL: AIChannel = {
  ...Common.Task.ZERO_READ_CHANNEL,
  type: AI_CHANNEL_TYPE,
  port: "AIN0",
  range: 0,
  scale: ZERO_SCALES[NO_SCALE_TYPE],
};

// Digital Input Channels

export const DI_CHANNEL_TYPE = "DI";
export type DIChannelType = typeof DI_CHANNEL_TYPE;

const diChannelZ = Common.Task.readChannelZ.extend({
  type: z.literal(DI_CHANNEL_TYPE),
  port: digitalPortZ,
});
interface DIChannel extends z.infer<typeof diChannelZ> {}
const ZERO_DI_CHANNEL: DIChannel = {
  ...Common.Task.ZERO_READ_CHANNEL,
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
  thermocoupleType: K_TC_TYPE,
  posChan: 0,
  negChan: 199,
  units: KELVIN_UNIT,
  cjcSource: DEVICE_CJC_SOURCE,
  cjcSlope: 1,
  cjcOffset: 0,
  scale: NO_SCALE,
};

// Input Channels

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

// Base Output Channels

const baseOutputChannelZ = Common.Task.channelZ.extend({
  cmdKey: channel.keyZ,
  stateKey: channel.keyZ,
});
interface BaseOutputChannel extends z.infer<typeof baseOutputChannelZ> {}
const ZERO_BASE_OUTPUT_CHANNEL: BaseOutputChannel = {
  ...Common.Task.ZERO_CHANNEL,
  cmdKey: 0,
  stateKey: 0,
};

// Analog Output Channels

export const AO_CHANNEL_TYPE = "AO";
export type AOChannelType = typeof AO_CHANNEL_TYPE;

const aoChannelZ = baseOutputChannelZ.extend({
  type: z.literal(AO_CHANNEL_TYPE),
  port: portZ.regex(
    Device.DAC_PORT_REGEX,
    "Invalid port, ports must start with DAC and end with an integer",
  ),
});
interface AOChannel extends z.infer<typeof aoChannelZ> {}
const ZERO_AO_CHANNEL: AOChannel = {
  ...ZERO_BASE_OUTPUT_CHANNEL,
  type: AO_CHANNEL_TYPE,
  port: "DAC0",
};

// Digital Output Channels

export const DO_CHANNEL_TYPE = "DO";
export type DOChannelType = typeof DO_CHANNEL_TYPE;

const doChannelZ = baseOutputChannelZ.extend({
  type: z.literal(DO_CHANNEL_TYPE),
  port: portZ.regex(
    Device.DIO_PORT_REGEX,
    "Invalid port, ports must start with DIO and end with an integer",
  ),
});
interface DOChannel extends z.infer<typeof doChannelZ> {}
const ZERO_DO_CHANNEL: DOChannel = {
  ...ZERO_BASE_OUTPUT_CHANNEL,
  type: DO_CHANNEL_TYPE,
  port: "DIO4",
};

// Output Channels

const outputChannelZ = z.union([aoChannelZ, doChannelZ]);
export type OutputChannel = z.infer<typeof outputChannelZ>;
export type OutputChannelType = OutputChannel["type"];

export const ZERO_OUTPUT_CHANNELS: Record<OutputChannelType, OutputChannel> = {
  [AO_CHANNEL_TYPE]: ZERO_AO_CHANNEL,
  [DO_CHANNEL_TYPE]: ZERO_DO_CHANNEL,
};
export const ZERO_OUTPUT_CHANNEL: OutputChannel = ZERO_OUTPUT_CHANNELS[DO_CHANNEL_TYPE];

// Channels

export type Channel = InputChannel | OutputChannel;
export type ChannelType = Channel["type"];

// Tasks

const baseConfigZ = z.object({ dataSaving: z.boolean(), device: Common.Device.keyZ });
interface BaseConfig extends z.infer<typeof baseConfigZ> {}
const ZERO_BASE_CONFIG: BaseConfig = {
  dataSaving: true,
  device: "",
};

const validateUniquePorts = (channels: Channel[], { addIssue }: z.RefinementCtx) => {
  const portToIndexMap = new Map<string, number>();
  channels.forEach(({ port }, i) => {
    if (!portToIndexMap.has(port)) {
      portToIndexMap.set(port, i);
      return;
    }
    const index = portToIndexMap.get(port) as number;
    const baseIssue = {
      code: z.ZodIssueCode.custom,
      message: `Port ${port} has already been used on another channel`,
    };
    addIssue({ ...baseIssue, path: [index, "port"] });
    addIssue({ ...baseIssue, path: [i, "port"] });
  });
};

export interface BaseStateDetails {
  running: boolean;
}

// Read Task

export const readConfigZ = baseConfigZ
  .extend({
    channels: z
      .array(inputChannelZ)
      .superRefine(Common.Task.validateReadChannels)
      .superRefine(validateUniquePorts),
    sampleRate: z.number().positive().max(50000),
    streamRate: z.number().positive().max(50000),
  })
  .refine(Common.Task.validateStreamRate);
export interface ReadConfig extends z.infer<typeof readConfigZ> {}
const ZERO_READ_CONFIG: ReadConfig = {
  ...ZERO_BASE_CONFIG,
  channels: [],
  sampleRate: 10,
  streamRate: 5,
};

export interface ReadStateDetails extends BaseStateDetails {
  message: string;
  errors?: { message: string; path: string }[];
}
export interface ReadState extends task.State<ReadStateDetails> {}

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
export interface NewReadTask extends task.New<ReadConfig, ReadType> {}

// Write Task

interface IndexAndType {
  index: number;
  type: "cmd" | "state";
}

export const writeConfigZ = baseConfigZ.extend({
  channels: z
    .array(outputChannelZ)
    .superRefine(Common.Task.validateChannels)
    .superRefine(validateUniquePorts)
    .superRefine((channels, { addIssue }) => {
      // This has to be separate from the common write channel validation because
      // LabJack write channels stupidly use 'cmdKey' and 'stateKey' instead of
      // 'cmdChannel' and 'stateChannel' like everything else.
      const channelsToIndexMap = new Map<channel.Key, IndexAndType>();
      channels.forEach(({ cmdKey, stateKey }, i) => {
        if (cmdKey !== 0)
          if (channelsToIndexMap.has(cmdKey)) {
            const { index, type } = channelsToIndexMap.get(cmdKey) as IndexAndType;
            const baseIssue = {
              code: z.ZodIssueCode.custom,
              message: `Synnax channel with key ${cmdKey} is used on multiple channels`,
            };
            addIssue({ ...baseIssue, path: [index, `${type}Key`] });
            addIssue({ ...baseIssue, path: [i, "cmdKey"] });
          } else channelsToIndexMap.set(cmdKey, { index: i, type: "cmd" });
        if (stateKey === 0) return;
        if (channelsToIndexMap.has(stateKey)) {
          const { index, type } = channelsToIndexMap.get(stateKey) as IndexAndType;
          const baseIssue = {
            code: z.ZodIssueCode.custom,
            message: `Synnax channel with key ${stateKey} is used on multiple channels`,
          };
          addIssue({ ...baseIssue, path: [index, `${type}Key`] });
          addIssue({ ...baseIssue, path: [i, "stateKey"] });
        } else channelsToIndexMap.set(stateKey, { index: i, type: "state" });
      });
    }),
  stateRate: z.number().positive().max(50000),
});
export interface WriteConfig extends z.infer<typeof writeConfigZ> {}
const ZERO_WRITE_CONFIG: WriteConfig = {
  ...ZERO_BASE_CONFIG,
  channels: [],
  stateRate: 10,
};

export interface WriteStateDetails extends BaseStateDetails {}
export interface WriteState extends task.State<WriteStateDetails> {}

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
export interface NewWriteTask extends task.New<WriteConfig, WriteType> {}
