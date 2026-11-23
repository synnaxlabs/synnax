// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, icd, query } from "@synnaxlabs/client";
import { Channel } from "@synnaxlabs/pluto";
import { unique } from "@synnaxlabs/x";

import { converts } from "@/platform/icd/dataType";
import type { Task } from "@/platform/task";

/**
 * @returns the bindings of fields, each to the channel stored for its pointer. A field
 * gets no channel when its stored channel is in written, as the Core refuses a channel
 * that two bindings write, or when it does not convert to the data type of the channel
 * in dataTypes. Adds the channels it binds to written.
 */
export const bindingsOf = (
  stored: Record<string, channel.Key>,
  fields: icd.JSONField[],
  written: Set<channel.Key>,
  dataTypes: ReadonlyMap<channel.Key, string>,
): icd.ReadBinding[] =>
  fields.map((f) => {
    const ch = stored[f.pointer] ?? 0;
    const dataType = dataTypes.get(ch);
    if (written.has(ch) || dataType == null || !converts(f, dataType))
      return icd.readBindingZ.parse({ field: f.key });
    written.add(ch);
    return icd.readBindingZ.parse({ field: f.key, channel: ch });
  });

/** Where the read properties of a device store the channels of one source. */
export interface StoredSource {
  /** The key of the source. The key of each of its messages extends it. */
  key: string;
  /** The key where an older Console stored the channels of the source, if any. */
  legacy?: string;
  /** The messages that the source reads. */
  messages: icd.ReadMessage[];
}

/** @returns the key of the read properties that store the channels of a message. */
export const storedKey = (source: string, message: icd.MessageKey): string =>
  `${source}#${message}`;

/**
 * @returns the channels that stored holds for read r of source. The channels under the
 * legacy key belong to the message that binds one of them, or else to the first one.
 */
export const storedOf = (
  stored: Record<string, Task.ReadChannelProps>,
  { key, legacy, messages }: StoredSource,
  r: number,
): Task.ReadChannelProps | undefined => {
  const own = stored[storedKey(key, messages[r].message)];
  if (own != null || legacy == null) return own;
  const held = stored[legacy];
  if (held == null) return undefined;
  const channels = new Set([held.index, ...Object.values(held.channels)]);
  channels.delete(0);
  const owner = messages.findIndex(({ bindings }) =>
    bindings.some(({ channel }) => channels.has(channel)),
  );
  return Math.max(owner, 0) === r ? held : undefined;
};

/**
 * @returns the channel that stored holds for each pointer of source, for a message
 * that the source reads first. The channels of the message itself come first. The
 * channels of a later read of the source stay with that read.
 */
export const sourceChannels = (
  stored: Record<string, Task.ReadChannelProps>,
  source: StoredSource,
  message: icd.MessageKey,
): Record<string, channel.Key> => {
  const claimed = new Set(
    source.messages.slice(1).map((_, i) => storedOf(stored, source, i + 1)),
  );
  const channels: Record<string, channel.Key> = {};
  for (const [key, props] of Object.entries(stored))
    if (
      (key === source.legacy || key.startsWith(`${source.key}#`)) &&
      !claimed.has(props)
    )
      Object.assign(channels, props.channels);
  return { ...channels, ...stored[storedKey(source.key, message)]?.channels };
};

/**
 * Moves the channels under the legacy key of source to the key of read r, when they
 * belong to that read.
 */
export const moveLegacy = (
  stored: Record<string, Task.ReadChannelProps>,
  source: StoredSource,
  r: number,
): void => {
  const key = storedKey(source.key, source.messages[r].message);
  if (stored[key] != null || source.legacy == null) return;
  const held = storedOf(stored, source, r);
  if (held == null) return;
  stored[key] = held;
  delete stored[source.legacy];
};

export interface StoredDataTypes {
  /** The data type of each stored channel that exists, by key. */
  dataTypes: ReadonlyMap<channel.Key, string>;
  /**
   * True while the data types load. Each control that binds waits for them, as the
   * Core refuses a channel that a field does not convert to.
   */
  bindDisabled: boolean;
}

/** @returns the data types of the channels in read, the read properties of a device. */
export const useStoredDataTypes = (
  read: Record<string, Pick<Task.ReadChannelProps, "channels">>,
): StoredDataTypes => {
  const keys = unique
    .unique(Object.values(read).flatMap(({ channels }) => Object.values(channels)))
    .filter((key) => key !== 0);
  const { data, variant } = Channel.useResultMultiple(
    keys.length === 0 ? null : query.keysRequest(keys),
  );
  return {
    dataTypes: new Map(data?.map(({ key, dataType }) => [key, dataType.toString()])),
    bindDisabled: keys.length > 0 && variant !== "success",
  };
};
