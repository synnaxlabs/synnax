// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, ethercat, type task, UnexpectedError } from "@synnaxlabs/client";
import { caseconv, id } from "@synnaxlabs/x";
import { z } from "zod";

import { type SlaveDevice } from "@/feature/ethercat/device/types";
import { Task } from "@/platform/task";

export const PREFIX = "ethercat";

export type InputChannel = ethercat.InputChannel;

export const ZERO_INPUT_CHANNELS = {
  automatic: ethercat.inputChannelAutomaticZ.parse({ type: "automatic" }),
  manual: ethercat.inputChannelManualZ.parse({ type: "manual" }),
} satisfies Record<ChannelMode, InputChannel>;

export type OutputChannel = ethercat.OutputChannel;

export const ZERO_OUTPUT_CHANNELS = {
  automatic: ethercat.outputChannelAutomaticZ.parse({ type: "automatic" }),
  manual: ethercat.outputChannelManualZ.parse({ type: "manual" }),
} satisfies Record<ChannelMode, OutputChannel>;

export type Channel = InputChannel | OutputChannel;

export type ChannelMode = Channel["type"];

export const READ_TYPE = `${PREFIX}_read`;

export const readConfigZ = ethercat.readConfigZ;

export const deployReadConfigZ = ethercat.readConfigZ
  .extend({
    sampleRate: z.number().positive(),
    streamRate: z.number().positive(),
  })
  .check(Task.validateStreamRate);

const ZERO_READ_CONFIG = ethercat.readConfigZ.parse({});

const readStatusDataZ = z
  .object({
    running: z.boolean(),
    message: z.string(),
    errors: z.array(z.object({ message: z.string(), path: z.string() })).optional(),
  })
  .nullish()
  .optional();

export const READ_SCHEMAS = {
  type: z.literal(READ_TYPE),
  config: readConfigZ,
  statusData: readStatusDataZ,
} as const satisfies task.Schemas;

export type ReadSchemas = typeof READ_SCHEMAS;

export interface ReadPayload extends task.Payload<ReadSchemas> {}

export const ZERO_READ_PAYLOAD: ReadPayload = {
  key: "",
  rack: 0,
  name: "EtherCAT Read Task",
  config: ZERO_READ_CONFIG,
  configHash: "",
  type: READ_TYPE,
  internal: false,
  snapshot: false,
};

export const WRITE_TYPE = `${PREFIX}_write`;

export const writeConfigZ = ethercat.writeConfigZ;

export const deployWriteConfigZ = ethercat.writeConfigZ.extend({
  stateRate: z.number().positive(),
  executionRate: z.number().positive(),
});

const ZERO_WRITE_CONFIG = ethercat.writeConfigZ.parse({});

const writeStatusDataZ = z
  .object({
    running: z.boolean(),
    message: z.string(),
    errors: z.array(z.object({ message: z.string(), path: z.string() })).optional(),
  })
  .nullish()
  .optional();

export const WRITE_SCHEMAS = {
  type: z.literal(WRITE_TYPE),
  config: writeConfigZ,
  statusData: writeStatusDataZ,
} as const satisfies task.Schemas;

export type WriteSchemas = typeof WRITE_SCHEMAS;

export interface WritePayload extends task.Payload<WriteSchemas> {}

export const ZERO_WRITE_PAYLOAD: WritePayload = {
  key: "",
  rack: 0,
  name: "EtherCAT Write Task",
  config: ZERO_WRITE_CONFIG,
  configHash: "",
  type: WRITE_TYPE,
  internal: false,
  snapshot: false,
};

/** Generates a unique map key for a channel configuration within a slave. */
export const channelMapKey = (ch: Channel): string => {
  if (ch.type === "automatic") return `auto_${ch.pdo}`;
  return `manual_${ch.index}_${ch.subIndex}`;
};

/** Creates a new input channel, copying from the last channel if available. */
export const createInputChannel = (channels: InputChannel[]): InputChannel => {
  if (channels.length === 0)
    return { ...ZERO_INPUT_CHANNELS.automatic, key: id.create() };
  const last = channels[channels.length - 1];
  return { ...last, ...Task.READ_CHANNEL_OVERRIDE, key: id.create() };
};

/** Creates a new output channel, copying from the last channel if available. */
export const createOutputChannel = (channels: OutputChannel[]): OutputChannel => {
  if (channels.length === 0)
    return { ...ZERO_OUTPUT_CHANNELS.automatic, key: id.create() };
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
    : `0x${ch.index.toString(16).padStart(4, "0")}:${ch.subIndex}`;

/** Generates a safe name for a PDO channel. */
export const getPDOName = (ch: Channel): string =>
  channel.escapeInvalidName(
    ch.type === "automatic" ? ch.pdo : `0x${ch.index.toString(16)}_${ch.subIndex}`,
  );
