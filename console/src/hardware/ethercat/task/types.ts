// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, type task } from "@synnaxlabs/client";
import { id } from "@synnaxlabs/x";
import { z } from "zod/v4";

import { Common } from "@/hardware/common";

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

const readChannelZ = z.union([
  automaticChannelZ.extend({ channel: channel.keyZ }),
  manualChannelZ.extend({ channel: channel.keyZ }),
]);
export type ReadChannel = z.infer<typeof readChannelZ>;
export type ReadChannelType = ReadChannel["type"];

export const ZERO_AUTOMATIC_READ_CHANNEL = {
  type: AUTOMATIC_TYPE,
  device: "",
  pdo: "",
  channel: 0,
  enabled: true,
  key: "",
  name: "",
} as const satisfies ReadChannel;

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
} as const satisfies ReadChannel;

export const ZERO_READ_CHANNELS: Record<ReadChannelType, ReadChannel> = {
  [AUTOMATIC_TYPE]: ZERO_AUTOMATIC_READ_CHANNEL,
  [MANUAL_TYPE]: ZERO_MANUAL_READ_CHANNEL,
};

export const readConfigZ = Common.Task.baseReadConfigZ
  .extend({
    device: z.string(),
    sampleRate: z.number().positive(),
    streamRate: z.number().positive(),
    channels: z.array(readChannelZ),
  })
  .check(Common.Task.validateStreamRate);
interface ReadConfig extends z.infer<typeof readConfigZ> {}

const ZERO_READ_CONFIG: ReadConfig = {
  ...Common.Task.ZERO_BASE_READ_CONFIG,
  device: "",
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

const writeChannelZ = z.union([
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
export type WriteChannel = z.infer<typeof writeChannelZ>;
export type WriteChannelType = WriteChannel["type"];

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
} as const satisfies WriteChannel;

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
} as const satisfies WriteChannel;

export const ZERO_WRITE_CHANNELS: Record<WriteChannelType, WriteChannel> = {
  [AUTOMATIC_TYPE]: ZERO_AUTOMATIC_WRITE_CHANNEL,
  [MANUAL_TYPE]: ZERO_MANUAL_WRITE_CHANNEL,
};

export const writeConfigZ = Common.Task.baseConfigZ.extend({
  device: z.string(),
  stateRate: z.number().positive(),
  channels: z.array(writeChannelZ),
});
interface WriteConfig extends z.infer<typeof writeConfigZ> {}

const ZERO_WRITE_CONFIG: WriteConfig = {
  ...Common.Task.ZERO_BASE_CONFIG,
  device: "",
  stateRate: 25,
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

/** Generates a unique map key for a read channel configuration. */
export const readMapKey = (ch: ReadChannel): string => {
  if (ch.type === AUTOMATIC_TYPE) return `${ch.device}-auto-${ch.pdo}`;
  return `${ch.device}-manual-${ch.index}-${ch.subindex}`;
};

/** Generates a unique map key for a write channel configuration. */
export const writeMapKey = (ch: WriteChannel): string => {
  if (ch.type === AUTOMATIC_TYPE) return `${ch.device}-auto-${ch.pdo}`;
  return `${ch.device}-manual-${ch.index}-${ch.subindex}`;
};

/** Creates a new read channel, copying from the last channel if available. */
export const createReadChannel = (channels: ReadChannel[]): ReadChannel => {
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
export const createWriteChannel = (channels: WriteChannel[]): WriteChannel => {
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
