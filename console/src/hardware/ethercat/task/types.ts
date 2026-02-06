// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, type task, UnexpectedError } from "@synnaxlabs/client";
import { caseconv, id } from "@synnaxlabs/x";
import { z } from "zod/v4";

import { Common } from "@/hardware/common";
import { type SlaveDevice } from "@/hardware/ethercat/device/types";

export const PREFIX = "ethercat";
export const READ_TYPE = `${PREFIX}_read`;
export const WRITE_TYPE = `${PREFIX}_write`;
export const SCAN_TYPE = `${PREFIX}_scan`;

export const AUTOMATIC_TYPE = "automatic";
export const MANUAL_TYPE = "manual";
export type ChannelMode = typeof AUTOMATIC_TYPE | typeof MANUAL_TYPE;

const baseChannelZ = Common.Task.channelZ.extend({
  device: z.string(),
  name: Common.Task.nameZ,
});

const automaticChannelZ = baseChannelZ.extend({
  type: z.literal(AUTOMATIC_TYPE),
  pdo: z.string(),
});

const manualChannelZ = baseChannelZ.extend({
  type: z.literal(MANUAL_TYPE),
  index: z.number(),
  subindex: z.number(),
  bitLength: z.number(),
  dataType: z.string(),
});

const inputChannelZ = z.union([
  automaticChannelZ.extend({ channel: channel.keyZ }),
  manualChannelZ.extend({ channel: channel.keyZ }),
]);
export type InputChannel = z.infer<typeof inputChannelZ>;
export type InputChannelType = InputChannel["type"];

export const ZERO_AUTOMATIC_READ_CHANNEL = {
  type: AUTOMATIC_TYPE,
  device: "",
  pdo: "",
  channel: 0,
  enabled: true,
  key: "",
  name: "",
} as const satisfies InputChannel;

export const ZERO_MANUAL_READ_CHANNEL = {
  type: MANUAL_TYPE,
  device: "",
  index: 0,
  subindex: 0,
  bitLength: 16,
  dataType: "uint16",
  channel: 0,
  enabled: true,
  key: "",
  name: "",
} as const satisfies InputChannel;

export const ZERO_READ_CHANNELS: Record<InputChannelType, InputChannel> = {
  [AUTOMATIC_TYPE]: ZERO_AUTOMATIC_READ_CHANNEL,
  [MANUAL_TYPE]: ZERO_MANUAL_READ_CHANNEL,
};

export const readConfigZ = Common.Task.baseReadConfigZ
  .omit({ device: true })
  .extend({
    sampleRate: z.number().positive(),
    streamRate: z.number().positive(),
    channels: z.array(inputChannelZ),
  })
  .check(Common.Task.validateStreamRate);
interface ReadConfig extends z.infer<typeof readConfigZ> {}

const ZERO_READ_CONFIG: ReadConfig = {
  autoStart: false,
  dataSaving: true,
  sampleRate: 1000,
  streamRate: 25,
  channels: [],
};

export const readStatusDataZ = z
  .object({
    running: z.boolean(),
    message: z.string(),
    errors: z.array(z.object({ message: z.string(), path: z.string() })).optional(),
  })
  .or(z.null());

export const readTypeZ = z.literal(READ_TYPE);

interface ReadPayload extends task.Payload<
  typeof readTypeZ,
  typeof readConfigZ,
  typeof readStatusDataZ
> {}

export const ZERO_READ_PAYLOAD: ReadPayload = {
  key: "",
  name: "EtherCAT Read Task",
  config: ZERO_READ_CONFIG,
  type: READ_TYPE,
};

export const READ_SCHEMAS: task.Schemas<
  typeof readTypeZ,
  typeof readConfigZ,
  typeof readStatusDataZ
> = {
  typeSchema: readTypeZ,
  configSchema: readConfigZ,
  statusDataSchema: readStatusDataZ,
};

const outputChannelZ = z.union([
  automaticChannelZ.extend({
    cmdChannel: channel.keyZ,
    stateChannel: channel.keyZ,
    cmdChannelName: Common.Task.nameZ,
    stateChannelName: Common.Task.nameZ,
  }),
  manualChannelZ.extend({
    cmdChannel: channel.keyZ,
    stateChannel: channel.keyZ,
    cmdChannelName: Common.Task.nameZ,
    stateChannelName: Common.Task.nameZ,
  }),
]);
export type OutputChannel = z.infer<typeof outputChannelZ>;
export type OutputChannelType = OutputChannel["type"];

export const ZERO_AUTOMATIC_WRITE_CHANNEL = {
  type: AUTOMATIC_TYPE,
  device: "",
  pdo: "",
  cmdChannel: 0,
  stateChannel: 0,
  cmdChannelName: "",
  stateChannelName: "",
  enabled: true,
  key: "",
  name: "",
} as const satisfies OutputChannel;

export const ZERO_MANUAL_WRITE_CHANNEL = {
  type: MANUAL_TYPE,
  device: "",
  index: 0,
  subindex: 0,
  bitLength: 16,
  dataType: "uint16",
  cmdChannel: 0,
  stateChannel: 0,
  cmdChannelName: "",
  stateChannelName: "",
  enabled: true,
  key: "",
  name: "",
} as const satisfies OutputChannel;

export const ZERO_WRITE_CHANNELS: Record<OutputChannelType, OutputChannel> = {
  [AUTOMATIC_TYPE]: ZERO_AUTOMATIC_WRITE_CHANNEL,
  [MANUAL_TYPE]: ZERO_MANUAL_WRITE_CHANNEL,
};

export const writeConfigZ = Common.Task.baseConfigZ.omit({ device: true }).extend({
  stateRate: z.number().positive(),
  executionRate: z.number().positive(),
  channels: z.array(outputChannelZ),
});
interface WriteConfig extends z.infer<typeof writeConfigZ> {}

const ZERO_WRITE_CONFIG: WriteConfig = {
  autoStart: false,
  stateRate: 25,
  executionRate: 1000,
  channels: [],
};

export const writeStatusDataZ = z
  .object({
    running: z.boolean(),
    message: z.string(),
    errors: z.array(z.object({ message: z.string(), path: z.string() })).optional(),
  })
  .or(z.null());

export const writeTypeZ = z.literal(WRITE_TYPE);

interface WritePayload extends task.Payload<
  typeof writeTypeZ,
  typeof writeConfigZ,
  typeof writeStatusDataZ
> {}

export const ZERO_WRITE_PAYLOAD: WritePayload = {
  key: "",
  name: "EtherCAT Write Task",
  config: ZERO_WRITE_CONFIG,
  type: WRITE_TYPE,
};

export const WRITE_SCHEMAS: task.Schemas<
  typeof writeTypeZ,
  typeof writeConfigZ,
  typeof writeStatusDataZ
> = {
  typeSchema: writeTypeZ,
  configSchema: writeConfigZ,
  statusDataSchema: writeStatusDataZ,
};

const scanTypeZ = z.literal(SCAN_TYPE);
const scanConfigZ = z.object({});
const scanStatusDataZ = z.object({}).or(z.null());

export const SCAN_SCHEMAS: task.Schemas<
  typeof scanTypeZ,
  typeof scanConfigZ,
  typeof scanStatusDataZ
> = {
  typeSchema: scanTypeZ,
  configSchema: scanConfigZ,
  statusDataSchema: scanStatusDataZ,
};

/** Generates a unique map key for a channel configuration within a slave. */
export const channelMapKey = (ch: InputChannel | OutputChannel): string => {
  if (ch.type === AUTOMATIC_TYPE) return `auto_${ch.pdo}`;
  return `manual_${ch.index}_${ch.subindex}`;
};

/** Creates a new read channel, copying from the last channel if available. */
export const createReadChannel = (channels: InputChannel[]): InputChannel => {
  if (channels.length === 0)
    return { ...ZERO_AUTOMATIC_READ_CHANNEL, key: id.create() };
  const last = channels[channels.length - 1];
  return {
    ...last,
    ...Common.Task.READ_CHANNEL_OVERRIDE,
    key: id.create(),
  };
};

/** Creates a new write channel, copying from the last channel if available. */
export const createWriteChannel = (channels: OutputChannel[]): OutputChannel => {
  if (channels.length === 0)
    return { ...ZERO_AUTOMATIC_WRITE_CHANNEL, key: id.create() };
  const last = channels[channels.length - 1];
  return {
    ...last,
    key: id.create(),
    name: "",
    cmdChannel: 0,
    stateChannel: 0,
    cmdChannelName: "",
    stateChannelName: "",
  };
};

/** Gets a channel key from a map, trying both the original key and camelCase. */
export const getChannelByMapKey = (
  channels: Record<string, number>,
  mapKey: string,
): number => channels[mapKey] ?? channels[caseconv.snakeToCamel(mapKey)] ?? 0;

/** Resolves the data type for a PDO entry on a slave device. */
export const resolvePDODataType = (
  slave: SlaveDevice,
  pdo: string,
  pdoType: "inputs" | "outputs",
): string => {
  const pdoEntry = slave.properties.pdos[pdoType].find((p) => p.name === pdo);
  if (pdoEntry == null)
    throw new UnexpectedError(
      `PDO "${pdo}" not found in ${pdoType} on slave "${slave.properties.name}"`,
    );
  return pdoEntry.dataType;
};

/** Generates a display label for a channel's port/address. */
export const getPortLabel = (ch: InputChannel | OutputChannel): string =>
  ch.type === AUTOMATIC_TYPE
    ? ch.pdo || "No PDO"
    : `0x${ch.index.toString(16).padStart(4, "0")}:${ch.subindex}`;

/** Generates a safe name for a PDO channel. */
export const getPDOName = (ch: InputChannel | OutputChannel): string =>
  channel.escapeInvalidName(
    ch.type === AUTOMATIC_TYPE ? ch.pdo : `0x${ch.index.toString(16)}_${ch.subindex}`,
  );

export type Channel = InputChannel | OutputChannel;
