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

const linearScaleZ = z.object({
  type: z.literal("linear"),
  slope: z.number(),
  offset: z.number(),
});

interface LinearScale extends z.infer<typeof linearScaleZ> {}

const ZERO_LINEAR_SCALE = {
  type: "linear",
  slope: 1,
  offset: 0,
} as const satisfies LinearScale;

const noScaleZ = z.object({ type: z.literal("none") });

interface NoScale extends z.infer<typeof noScaleZ> {}

const NO_SCALE = { type: "none" } as const satisfies NoScale;

const scaleZ = z.union([noScaleZ, linearScaleZ]);

export type Scale = z.infer<typeof scaleZ>;

export type ScaleType = Scale["type"];

export const SCALE_SCHEMAS: Record<ScaleType, z.ZodType<Scale>> = {
  none: noScaleZ,
  linear: linearScaleZ,
};

export const ZERO_SCALES: Record<ScaleType, Scale> = {
  none: NO_SCALE,
  linear: ZERO_LINEAR_SCALE,
};

const aiChannelZ = Common.Task.readChannelZ.extend({
  type: z.literal("AI"),
  range: z.number().positive().optional(),
  scale: scaleZ,
  port: portZ.regex(
    Device.AIN_PORT_REGEX,
    "Invalid port, ports must start with AIN and end with an integer",
  ),
});

interface AIChannel extends z.infer<typeof aiChannelZ> {}

const ZERO_AI_CHANNEL = {
  ...Common.Task.ZERO_READ_CHANNEL,
  type: "AI",
  port: "AIN0",
  range: 10,
  scale: ZERO_SCALES.none,
} as const satisfies AIChannel;

const diChannelZ = Common.Task.readChannelZ.extend({
  type: z.literal("DI"),
  port: digitalPortZ,
});

interface DIChannel extends z.infer<typeof diChannelZ> {}

const ZERO_DI_CHANNEL = {
  ...Common.Task.ZERO_READ_CHANNEL,
  port: "DIO4",
  type: "DI",
} as const satisfies DIChannel;

const temperatureUnitsZ = z.enum(["C", "F", "K"]);

export type TemperatureUnits = z.infer<typeof temperatureUnitsZ>;

const thermocoupleTypeZ = z.enum(["J", "K", "N", "R", "S", "T", "B", "E", "C"]);

export type ThermocoupleType = z.infer<typeof thermocoupleTypeZ>;

export const DEVICE_CJC_SOURCE = "TEMPERATURE_DEVICE_K";

export const AIR_CJC_SOURCE = "TEMPERATURE_AIR_K";

const tcChannelZ = aiChannelZ.omit({ type: true, range: true }).extend({
  type: z.literal("TC"),
  thermocoupleType: thermocoupleTypeZ,
  posChan: z.number().int(),
  negChan: z.number().int(),
  cjcSource: z.string().min(1, "CJC Source must be specified"),
  cjcSlope: z.number(),
  cjcOffset: z.number(),
  units: temperatureUnitsZ,
});

interface TCChannel extends z.infer<typeof tcChannelZ> {}

const ZERO_TC_CHANNEL = {
  ...ZERO_AI_CHANNEL,
  type: "TC",
  thermocoupleType: "K",
  posChan: 0,
  negChan: 199,
  units: "K",
  cjcSource: DEVICE_CJC_SOURCE,
  cjcSlope: 1,
  cjcOffset: 0,
  scale: NO_SCALE,
} as const satisfies TCChannel;

const inputChannelZ = z.union([aiChannelZ, diChannelZ, tcChannelZ]);

export type InputChannel = z.infer<typeof inputChannelZ>;

export type InputChannelType = InputChannel["type"];

export const INPUT_CHANNEL_SCHEMAS: Record<
  InputChannelType,
  z.ZodType<InputChannel>
> = {
  AI: aiChannelZ,
  DI: diChannelZ,
  TC: tcChannelZ,
};

export const ZERO_INPUT_CHANNELS = {
  AI: ZERO_AI_CHANNEL,
  DI: ZERO_DI_CHANNEL,
  TC: ZERO_TC_CHANNEL,
} as const satisfies Record<InputChannelType, InputChannel>;

export const ZERO_INPUT_CHANNEL = ZERO_INPUT_CHANNELS.AI;

const v0BaseOutputChannelZ = Common.Task.channelZ.extend({
  cmdKey: channel.keyZ,
  stateKey: channel.keyZ,
});

const aoChannelExtension = {
  type: z.literal("AO"),
  port: portZ.regex(
    Device.DAC_PORT_REGEX,
    "Invalid port, ports must start with DAC and end with an integer",
  ),
};

const v0AOChannelZ = v0BaseOutputChannelZ.extend(aoChannelExtension);

const aoChannelZ = Common.Task.writeChannelZ.extend(aoChannelExtension);

interface AOChannel extends z.infer<typeof aoChannelZ> {}

const ZERO_AO_CHANNEL = {
  ...Common.Task.ZERO_WRITE_CHANNEL,
  type: "AO",
  port: "DAC0",
} as const satisfies AOChannel;

const doChannelExtension = {
  type: z.literal("DO"),
  port: portZ.regex(
    Device.DIO_PORT_REGEX,
    "Invalid port, ports must start with DIO and end with an integer",
  ),
};

const v0DOChannelZ = v0BaseOutputChannelZ.extend(doChannelExtension);

const doChannelZ = Common.Task.writeChannelZ.extend(doChannelExtension);

interface DOChannel extends z.infer<typeof doChannelZ> {}

const ZERO_DO_CHANNEL = {
  ...Common.Task.ZERO_WRITE_CHANNEL,
  type: "DO",
  port: "DIO4",
} as const satisfies DOChannel;

const v0OutputChannelZ = z.union([v0AOChannelZ, v0DOChannelZ]);

export const outputChannelZ = z.union([aoChannelZ, doChannelZ]);

export type OutputChannel = z.infer<typeof outputChannelZ>;

export type OutputChannelType = OutputChannel["type"];

export const ZERO_OUTPUT_CHANNELS: Record<OutputChannelType, OutputChannel> = {
  AO: ZERO_AO_CHANNEL,
  DO: ZERO_DO_CHANNEL,
};
export const ZERO_OUTPUT_CHANNEL: OutputChannel = ZERO_OUTPUT_CHANNELS.DO;

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

export const READ_TYPE = `${PREFIX}_read`;

const readConfigZ = Common.Task.baseReadConfigZ
  .extend({
    channels: z
      .array(inputChannelZ)
      .check(Common.Task.validateReadChannels)
      .check(validateUniquePorts),
    sampleRate: z.number().positive().max(50000),
    streamRate: z.number().positive().max(50000),
  })
  .check(Common.Task.validateStreamRate);

interface ReadConfig extends z.infer<typeof readConfigZ> {}

const ZERO_READ_CONFIG = {
  ...Common.Task.ZERO_BASE_READ_CONFIG,
  channels: [],
  sampleRate: 10,
  streamRate: 5,
} as const satisfies ReadConfig;

const readStatusDataZ = z
  .object({ errors: z.array(z.object({ message: z.string(), path: z.string() })) })
  .or(z.null());

export const READ_SCHEMAS = {
  typeSchema: z.literal(READ_TYPE),
  configSchema: readConfigZ,
  statusDataSchema: readStatusDataZ,
} as const satisfies task.Schemas;

export type ReadSchemas = typeof READ_SCHEMAS;

export interface ReadPayload extends task.Payload<ReadSchemas> {}

export const ZERO_READ_PAYLOAD = {
  key: "",
  name: "LabJack Read Task",
  config: ZERO_READ_CONFIG,
  type: "labjack_read",
} as const satisfies ReadPayload;

export const WRITE_TYPE = `${PREFIX}_write`;

const writeConfigZ = Common.Task.baseConfigZ.extend({
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

interface WriteConfig extends z.infer<typeof writeConfigZ> {}

const ZERO_WRITE_CONFIG = {
  ...Common.Task.ZERO_BASE_CONFIG,
  channels: [],
  dataSaving: true,
  stateRate: 10,
} as const satisfies WriteConfig;

export const WRITE_SCHEMAS = {
  typeSchema: z.literal(WRITE_TYPE),
  configSchema: writeConfigZ,
  statusDataSchema: z.unknown(),
} as const satisfies task.Schemas;

export type WriteSchemas = typeof WRITE_SCHEMAS;

export interface WritePayload extends task.Payload<WriteSchemas> {}

export const ZERO_WRITE_PAYLOAD = {
  key: "",
  name: "LabJack Write Task",
  config: ZERO_WRITE_CONFIG,
  type: "labjack_write",
} as const satisfies WritePayload;
