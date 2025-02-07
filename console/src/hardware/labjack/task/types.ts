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

import { type Device } from "@/hardware/labjack/device";

export const PREFIX = "labjack";

// Channel Types

export const DI_CHANNEL_TYPE = "DI";
type DIChannelType = typeof DI_CHANNEL_TYPE;

export const TC_CHANNEL_TYPE = "TC";
export type TCChannelType = typeof TC_CHANNEL_TYPE;

export const AO_CHANNEL_TYPE = "AO";
export type AOChannelType = typeof AO_CHANNEL_TYPE;

export const AI_CHANNEL_TYPE = "AI";
export type AIChannelType = typeof AI_CHANNEL_TYPE;

export const DO_CHANNEL_TYPE = "DO";
export type DOChannelType = typeof DO_CHANNEL_TYPE;

export type InputChannelType = DIChannelType | AIChannelType | TCChannelType;
export const outputChannelTypeZ = z.enum([AO_CHANNEL_TYPE, DO_CHANNEL_TYPE]);
export type OutputChannelType = z.infer<typeof outputChannelTypeZ>;
export type ChannelType = InputChannelType | OutputChannelType;

interface ConvertChannelTypeToPortType {
  [DI_CHANNEL_TYPE]: Device.DIPortType;
  [AI_CHANNEL_TYPE]: Device.AIPortType;
  [TC_CHANNEL_TYPE]: Device.AIPortType;
  [AO_CHANNEL_TYPE]: Device.AOPortType;
  [DO_CHANNEL_TYPE]: Device.DOPortType;
}

export const getPortTypeFromChannelType = <
  T extends keyof ConvertChannelTypeToPortType,
>(
  type: T,
): ConvertChannelTypeToPortType[T] => {
  if (type === DI_CHANNEL_TYPE) return "DI" as ConvertChannelTypeToPortType[T];
  if (type === AI_CHANNEL_TYPE) return "AI" as ConvertChannelTypeToPortType[T];
  if (type === TC_CHANNEL_TYPE) return "AI" as ConvertChannelTypeToPortType[T];
  if (type === AO_CHANNEL_TYPE) return "AO" as ConvertChannelTypeToPortType[T];
  if (type === DO_CHANNEL_TYPE) return "DO" as ConvertChannelTypeToPortType[T];
  throw new Error(`Unknown channel type: ${type}`);
};

const LINEAR_SCALE_TYPE = "linear";

const linearScaleZ = z.object({
  type: z.literal(LINEAR_SCALE_TYPE),
  slope: z.number(),
  offset: z.number(),
});

interface LinearScale extends z.infer<typeof linearScaleZ> {}

const ZERO_LINEAR_SCALE: LinearScale = { type: LINEAR_SCALE_TYPE, slope: 1, offset: 0 };

const NO_SCALE_TYPE = "none";

const noScaleZ = z.object({ type: z.literal(NO_SCALE_TYPE) });

interface NoScale extends z.infer<typeof noScaleZ> {}

const NO_SCALE: NoScale = { type: NO_SCALE_TYPE };

const scaleZ = z.union([noScaleZ, linearScaleZ]);
export type Scale = z.infer<typeof scaleZ>;
export type ScaleType = Scale["type"];

export const ZERO_SCALES: Record<ScaleType, Scale> = {
  [NO_SCALE_TYPE]: NO_SCALE,
  [LINEAR_SCALE_TYPE]: ZERO_LINEAR_SCALE,
};

export const SCALE_SCHEMAS: Record<ScaleType, z.ZodType<Scale>> = {
  [NO_SCALE_TYPE]: noScaleZ,
  [LINEAR_SCALE_TYPE]: linearScaleZ,
};

const baseInputChannelZ = z.object({
  port: z.string(),
  enabled: z.boolean(),
  key: z.string(),
  range: z.number().optional(),
  channel: channel.keyZ,
  scale: scaleZ,
});

export const inputChannelZ = baseInputChannelZ.extend({
  type: z.literal(AI_CHANNEL_TYPE).or(z.literal(DI_CHANNEL_TYPE)),
});

const CELSIUS_UNIT = "C";
const FAHRENHEIT_UNIT = "F";
const KELVIN_UNIT = "K";
const temperatureUnitsZ = z.enum([CELSIUS_UNIT, FAHRENHEIT_UNIT, KELVIN_UNIT]);
export type TemperatureUnits = z.infer<typeof temperatureUnitsZ>;

const thermocoupleTypeZ = z.enum(["J", "K", "N", "R", "S", "T", "B", "E", "C"]);

export const thermocoupleChannelZ = baseInputChannelZ.extend({
  range: z.number(),
  type: z.literal(TC_CHANNEL_TYPE),
  thermocoupleType: thermocoupleTypeZ,
  posChan: z.number(),
  negChan: z.number(),
  cjcSource: z.string(),
  cjcSlope: z.number(),
  cjcOffset: z.number(),
  units: temperatureUnitsZ,
});
interface ThermocoupleChannel extends z.infer<typeof thermocoupleChannelZ> {}
export const ZERO_THERMOCOUPLE_CHANNEL: ThermocoupleChannel = {
  port: "",
  enabled: true,
  key: "",
  channel: 0,
  range: 0,
  type: TC_CHANNEL_TYPE,
  thermocoupleType: KELVIN_UNIT,
  posChan: 0,
  negChan: 199,
  units: "K",
  cjcSource: "TEMPERATURE_DEVICE_K",
  cjcSlope: 1,
  cjcOffset: 0,
  scale: NO_SCALE,
};

const readChannelZ = z.union([inputChannelZ, thermocoupleChannelZ]);
export type ReadChannel = z.infer<typeof readChannelZ>;

export const ZERO_READ_CHANNEL: ReadChannel = {
  port: "AIN0",
  enabled: true,
  key: "",
  channel: 0,
  type: AI_CHANNEL_TYPE,
  range: 0,
  scale: { ...NO_SCALE },
};

const writeChannelZ = z.object({
  type: outputChannelTypeZ,
  port: z.string(),
  enabled: z.boolean(),
  cmdKey: channel.keyZ,
  stateKey: channel.keyZ,
  key: z.string(),
});

export interface WriteChannel extends z.infer<typeof writeChannelZ> {}
export const ZERO_WRITE_CHANNEL: WriteChannel = {
  port: "DIO4",
  enabled: true,
  key: "",
  cmdKey: 0,
  stateKey: 0,
  type: DO_CHANNEL_TYPE,
};

const deviceKeyZ = device.keyZ.min(1, "Must specify a device");

export const readConfigZ = z
  .object({
    device: deviceKeyZ,
    sampleRate: z.number().int().min(0).max(50000),
    streamRate: z.number().int().min(0).max(50000),
    channels: z.array(readChannelZ),
    dataSaving: z.boolean(),
  })
  .refine(
    (cfg) =>
      // Ensure that the stream Rate is lower than the sample rate
      cfg.sampleRate >= cfg.streamRate,
    {
      path: ["streamRate"],
      message: "Stream rate must be less than or equal to the sample rate",
    },
  );
export type ReadConfig = z.infer<typeof readConfigZ>;

export const writeConfigZ = z.object({
  device: deviceKeyZ,
  channels: z.array(writeChannelZ),
  dataSaving: z.boolean(),
  stateRate: z.number().int().min(1).max(50000),
});
export type WriteConfig = z.infer<typeof writeConfigZ>;

type BaseReadStateDetails = {
  running: boolean;
  message: string;
};

type ErrorReadStateDetails = BaseReadStateDetails & {
  errors: { message: string; path: string }[];
};

export type Channel = ReadChannel | WriteChannel;

export type ReadStateDetails = BaseReadStateDetails | ErrorReadStateDetails;

export type WriteStateDetails = {
  running: boolean;
};

export const READ_TYPE = `${PREFIX}_read`;
export type ReadType = typeof READ_TYPE;

const ZERO_READ_CONFIG: ReadConfig = {
  device: "",
  sampleRate: 10,
  streamRate: 5,
  channels: [],
  dataSaving: true,
};
export interface ReadTask extends task.Task<ReadConfig, ReadStateDetails, ReadType> {}
export type ReadPayload = task.Payload<ReadConfig, ReadStateDetails, ReadType>;
export const ZERO_READ_PAYLOAD: ReadPayload = {
  key: "",
  name: "LabJack Read Task",
  config: ZERO_READ_CONFIG,
  type: READ_TYPE,
};

export const WRITE_TYPE = `${PREFIX}_write`;
export type WriteType = typeof WRITE_TYPE;

const ZERO_WRITE_CONFIG: WriteConfig = {
  device: "",
  channels: [],
  dataSaving: true,
  stateRate: 10,
};
export type WriteTask2 = task.Task<WriteConfig, WriteStateDetails, WriteType>;
export interface WritePayload
  extends task.Payload<WriteConfig, WriteStateDetails, WriteType> {}
export const ZERO_WRITE_PAYLOAD: WritePayload = {
  key: "",
  name: "LabJack Write Task",
  config: ZERO_WRITE_CONFIG,
  type: WRITE_TYPE,
};
