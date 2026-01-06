// Copyright 2026 Synnax Labs, Inc.
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

const portZ = z.string().min(1, "Port must be specified");

const digitalPortZ = portZ.regex(
  Device.DIO_PORT_REGEX,
  "Invalid port, port must start with DIO and end with an integer",
);

export const LINEAR_SCALE_TYPE = "linear";
export type LinearScaleType = typeof LINEAR_SCALE_TYPE;

const linearScaleZ = z.object({
  type: z.literal(LINEAR_SCALE_TYPE),
  slope: z.number(),
  offset: z.number(),
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
  range: z.number().positive().optional(),
  scale: scaleZ,
  port: portZ.regex(
    Device.AIN_PORT_REGEX,
    "Invalid port, ports must start with AIN and end with an integer",
  ),
});
interface AIChannel extends z.infer<typeof aiChannelZ> {}
const ZERO_AI_CHANNEL: AIChannel = {
  ...Common.Task.ZERO_READ_CHANNEL,
  type: AI_CHANNEL_TYPE,
  port: "AIN0",
  range: 10,
  scale: ZERO_SCALES[NO_SCALE_TYPE],
};

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

const tcChannelZ = aiChannelZ.omit({ type: true, range: true }).extend({
  type: z.literal(TC_CHANNEL_TYPE),
  thermocoupleType: thermocoupleTypeZ,
  posChan: z.number().int(),
  negChan: z.number().int(),
  cjcSource: z.string().min(1, "CJC Source must be specified"),
  cjcSlope: z.number(),
  cjcOffset: z.number(),
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

const v0BaseOutputChannelZ = Common.Task.channelZ.extend({
  cmdKey: channel.keyZ,
  stateKey: channel.keyZ,
});

export const AO_CHANNEL_TYPE = "AO";
export type AOChannelType = typeof AO_CHANNEL_TYPE;

const aoChannelExtension = {
  type: z.literal(AO_CHANNEL_TYPE),
  port: portZ.regex(
    Device.DAC_PORT_REGEX,
    "Invalid port, ports must start with DAC and end with an integer",
  ),
};

const v0AOChannelZ = v0BaseOutputChannelZ.extend(aoChannelExtension);
const aoChannelZ = Common.Task.writeChannelZ.extend(aoChannelExtension);
interface AOChannel extends z.infer<typeof aoChannelZ> {}
const ZERO_AO_CHANNEL: AOChannel = {
  ...Common.Task.ZERO_WRITE_CHANNEL,
  type: AO_CHANNEL_TYPE,
  port: "DAC0",
};

export const DO_CHANNEL_TYPE = "DO";
export type DOChannelType = typeof DO_CHANNEL_TYPE;

const doChannelExtension = {
  type: z.literal(DO_CHANNEL_TYPE),
  port: portZ.regex(
    Device.DIO_PORT_REGEX,
    "Invalid port, ports must start with DIO and end with an integer",
  ),
};

const v0DOChannelZ = v0BaseOutputChannelZ.extend(doChannelExtension);
const doChannelZ = Common.Task.writeChannelZ.extend(doChannelExtension);
interface DOChannel extends z.infer<typeof doChannelZ> {}
const ZERO_DO_CHANNEL: DOChannel = {
  ...Common.Task.ZERO_WRITE_CHANNEL,
  type: DO_CHANNEL_TYPE,
  port: "DIO4",
};

const v0OutputChannelZ = z.union([v0AOChannelZ, v0DOChannelZ]);
type V0OutputChannel = z.infer<typeof v0OutputChannelZ>;
export type OutputChannelType = V0OutputChannel["type"];

export const outputChannelZ = z.union([aoChannelZ, doChannelZ]);
export type OutputChannel = z.infer<typeof outputChannelZ>;

export const ZERO_OUTPUT_CHANNELS: Record<OutputChannelType, OutputChannel> = {
  [AO_CHANNEL_TYPE]: ZERO_AO_CHANNEL,
  [DO_CHANNEL_TYPE]: ZERO_DO_CHANNEL,
};
export const ZERO_OUTPUT_CHANNEL: OutputChannel = ZERO_OUTPUT_CHANNELS[DO_CHANNEL_TYPE];

export type Channel = InputChannel | OutputChannel;
export type ChannelType = Channel["type"];

const validateUniquePorts: z.core.CheckFn<Channel[]> = ({
  value: channels,
  issues,
}) => {
  const portToIndexMap = new Map<string, number>();
  channels.forEach(({ port }, i) => {
    if (!portToIndexMap.has(port)) {
      portToIndexMap.set(port, i);
      return;
    }
    const index = portToIndexMap.get(port) as number;
    const code = "custom";
    const message = `Port ${port} has already been used on another channel`;
    issues.push({ code, message, path: [index, "port"], input: channels });
    issues.push({ code, message, path: [i, "port"], input: channels });
  });
};

export interface BaseStateDetails {
  running: boolean;
}

export const readConfigZ = Common.Task.baseReadConfigZ
  .extend({
    channels: z
      .array(inputChannelZ)
      .check(Common.Task.validateReadChannels)
      .check(validateUniquePorts),
    sampleRate: z.number().positive().max(50000),
    streamRate: z.number().positive().max(50000),
  })
  .check(Common.Task.validateStreamRate);
export interface ReadConfig extends z.infer<typeof readConfigZ> {}
const ZERO_READ_CONFIG: ReadConfig = {
  ...Common.Task.ZERO_BASE_READ_CONFIG,
  channels: [],
  sampleRate: 10,
  streamRate: 5,
};

export const readStatusDataZ = z
  .object({
    errors: z.array(z.object({ message: z.string(), path: z.string() })),
  })
  .or(z.null());

export type ReadStatus = task.Status<typeof readStatusDataZ>;

export const READ_TYPE = `${PREFIX}_read`;
export const readTypeZ = z.literal(READ_TYPE);
export type ReadType = typeof READ_TYPE;

export interface ReadPayload extends task.Payload<
  typeof readTypeZ,
  typeof readConfigZ,
  typeof readStatusDataZ
> {}
export const ZERO_READ_PAYLOAD: ReadPayload = {
  key: "",
  name: "LabJack Read Task",
  config: ZERO_READ_CONFIG,
  type: READ_TYPE,
};

export interface ReadTask extends task.Task<
  typeof readTypeZ,
  typeof readConfigZ,
  typeof readStatusDataZ
> {}
export interface NewReadTask extends task.New<typeof readTypeZ, typeof readConfigZ> {}

export const READ_SCHEMAS: task.PayloadSchemas<
  typeof readTypeZ,
  typeof readConfigZ,
  typeof readStatusDataZ
> = {
  type: readTypeZ,
  config: readConfigZ,
  statusData: readStatusDataZ,
};

export const writeConfigZ = Common.Task.baseConfigZ.extend({
  channels: z
    .array(v0OutputChannelZ)
    .transform((channels) =>
      channels.map<OutputChannel>(({ cmdKey, stateKey, ...rest }) => ({
        cmdChannel: cmdKey,
        stateChannel: stateKey,
        cmdChannelName: "",
        stateChannelName: "",
        ...rest,
      })),
    )
    .or(z.array(outputChannelZ))
    .check(Common.Task.validateWriteChannels)
    .check(validateUniquePorts),
  stateRate: z.number().positive().max(50000),
  dataSaving: z.boolean().default(true),
});
export interface WriteConfig extends z.infer<typeof writeConfigZ> {}
const ZERO_WRITE_CONFIG: WriteConfig = {
  ...Common.Task.ZERO_BASE_CONFIG,
  channels: [],
  dataSaving: true,
  stateRate: 10,
};

export const writeStatusDataZ = z.unknown();
export type WriteStatus = task.Status<typeof writeStatusDataZ>;

export const WRITE_TYPE = `${PREFIX}_write`;
export const writeTypeZ = z.literal(WRITE_TYPE);
export type WriteType = typeof WRITE_TYPE;

export interface WritePayload extends task.Payload<
  typeof writeTypeZ,
  typeof writeConfigZ,
  typeof writeStatusDataZ
> {}
export const ZERO_WRITE_PAYLOAD: WritePayload = {
  key: "",
  name: "LabJack Write Task",
  config: ZERO_WRITE_CONFIG,
  type: WRITE_TYPE,
};

export interface WriteTask extends task.Task<
  typeof writeTypeZ,
  typeof writeConfigZ,
  typeof writeStatusDataZ
> {}
export interface NewWriteTask extends task.New<
  typeof writeTypeZ,
  typeof writeConfigZ
> {}

export const WRITE_SCHEMAS: task.PayloadSchemas<
  typeof writeTypeZ,
  typeof writeConfigZ,
  typeof writeStatusDataZ
> = {
  type: writeTypeZ,
  config: writeConfigZ,
  statusData: writeStatusDataZ,
};
