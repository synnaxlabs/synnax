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

const baseChannelZ = Common.Task.channelZ.extend({
  device: z.string(),
  name: Common.Task.nameZ,
});

const automaticChannelZ = baseChannelZ.extend({
  type: z.literal("automatic"),
  pdo: z.string(),
});

const manualChannelZ = baseChannelZ.extend({
  type: z.literal("manual"),
  index: z.number(),
  subindex: z.number(),
  bitLength: z.number(),
  dataType: z.string(),
});

export const READ_TYPE = `${PREFIX}_read`;

const inputChannelExtensionShape = { channel: channel.keyZ } as const;

const inputChannelZ = z.discriminatedUnion("type", [
  automaticChannelZ.extend(inputChannelExtensionShape),
  manualChannelZ.extend(inputChannelExtensionShape),
]);

export type InputChannel = z.infer<typeof inputChannelZ>;

const ZERO_AUTOMATIC_INPUT_CHANNEL = {
  type: "automatic",
  device: "",
  pdo: "",
  channel: 0,
  enabled: true,
  key: "",
  name: "",
} as const satisfies InputChannel;

const ZERO_MANUAL_INPUT_CHANNEL = {
  type: "manual",
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

export const ZERO_INPUT_CHANNELS = {
  automatic: ZERO_AUTOMATIC_INPUT_CHANNEL,
  manual: ZERO_MANUAL_INPUT_CHANNEL,
} as const satisfies Record<ChannelMode, InputChannel>;

const outputChannelExtensionShape = {
  cmdChannel: channel.keyZ,
  stateChannel: channel.keyZ,
  cmdChannelName: Common.Task.nameZ,
  stateChannelName: Common.Task.nameZ,
} as const;

const outputChannelZ = z.union([
  automaticChannelZ.extend(outputChannelExtensionShape),
  manualChannelZ.extend(outputChannelExtensionShape),
]);

export type OutputChannel = z.infer<typeof outputChannelZ>;

const ZERO_AUTOMATIC_OUTPUT_CHANNEL = {
  type: "automatic",
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

const ZERO_MANUAL_OUTPUT_CHANNEL = {
  type: "manual",
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

export const ZERO_OUTPUT_CHANNELS = {
  automatic: ZERO_AUTOMATIC_OUTPUT_CHANNEL,
  manual: ZERO_MANUAL_OUTPUT_CHANNEL,
} as const satisfies Record<ChannelMode, OutputChannel>;

export type Channel = InputChannel | OutputChannel;

export type ChannelMode = Channel["type"];

const readConfigZ = Common.Task.baseReadConfigZ
  .omit({ device: true })
  .extend({
    sampleRate: z.number().positive(),
    streamRate: z.number().positive(),
    channels: z.array(inputChannelZ),
  })
  .check(Common.Task.validateStreamRate);

interface ReadConfig extends z.infer<typeof readConfigZ> {}

const ZERO_READ_CONFIG = {
  autoStart: false,
  dataSaving: true,
  sampleRate: 1000,
  streamRate: 25,
  channels: [],
} as const satisfies ReadConfig;

const readStatusDataZ = z
  .object({
    running: z.boolean(),
    message: z.string(),
    errors: z.array(z.object({ message: z.string(), path: z.string() })).optional(),
  })
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
  name: "EtherCAT Read Task",
  config: ZERO_READ_CONFIG,
  type: "ethercat_read",
} as const satisfies ReadPayload;

export const WRITE_TYPE = `${PREFIX}_write`;

const writeConfigZ = Common.Task.baseConfigZ.omit({ device: true }).extend({
  stateRate: z.number().positive(),
  executionRate: z.number().positive(),
  channels: z.array(outputChannelZ),
});

interface WriteConfig extends z.infer<typeof writeConfigZ> {}

const ZERO_WRITE_CONFIG = {
  autoStart: false,
  stateRate: 25,
  executionRate: 1000,
  channels: [],
} as const satisfies WriteConfig;

const writeStatusDataZ = z
  .object({
    running: z.boolean(),
    message: z.string(),
    errors: z.array(z.object({ message: z.string(), path: z.string() })).optional(),
  })
  .or(z.null());

export const WRITE_SCHEMAS = {
  typeSchema: z.literal(WRITE_TYPE),
  configSchema: writeConfigZ,
  statusDataSchema: writeStatusDataZ,
} as const satisfies task.Schemas;

export type WriteSchemas = typeof WRITE_SCHEMAS;

export interface WritePayload extends task.Payload<WriteSchemas> {}

export const ZERO_WRITE_PAYLOAD = {
  key: "",
  name: "EtherCAT Write Task",
  config: ZERO_WRITE_CONFIG,
  type: "ethercat_write",
} as const satisfies WritePayload;

/** Generates a unique map key for a channel configuration within a slave. */
export const channelMapKey = (ch: Channel): string => {
  if (ch.type === "automatic") return `auto_${ch.pdo}`;
  return `manual_${ch.index}_${ch.subindex}`;
};

/** Creates a new input channel, copying from the last channel if available. */
export const createInputChannel = (channels: InputChannel[]): InputChannel => {
  if (channels.length === 0)
    return { ...ZERO_AUTOMATIC_INPUT_CHANNEL, key: id.create() };
  const last = channels[channels.length - 1];
  return { ...last, ...Common.Task.READ_CHANNEL_OVERRIDE, key: id.create() };
};

/** Creates a new output channel, copying from the last channel if available. */
export const createOutputChannel = (channels: OutputChannel[]): OutputChannel => {
  if (channels.length === 0)
    return { ...ZERO_AUTOMATIC_OUTPUT_CHANNEL, key: id.create() };
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
export const getPortLabel = (ch: Channel): string =>
  ch.type === "automatic"
    ? ch.pdo || "No PDO"
    : `0x${ch.index.toString(16).padStart(4, "0")}:${ch.subindex}`;

/** Generates a safe name for a PDO channel. */
export const getPDOName = (ch: Channel): string =>
  channel.escapeInvalidName(
    ch.type === "automatic" ? ch.pdo : `0x${ch.index.toString(16)}_${ch.subindex}`,
  );
