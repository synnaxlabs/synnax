// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, icd } from "@synnaxlabs/client";

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
export interface Source {
  /** The key of the source. The key of each of its messages extends it. */
  key: string;
  /** The key where an older Console stored the channels of the source, if any. */
  legacy?: string;
  /** The messages that the source reads. */
  messages: icd.ReadMessage[];
}

/** @returns the key of the read properties that store the channels of a message. */
export const key = (source: string, message: icd.MessageKey): string =>
  `${source}#${message}`;

/**
 * @returns whether at is the key of a message of the source with key source. A source
 * key may hold "#", and a message key never does.
 */
const inSource = (at: string, source: string): boolean => {
  const message = at.slice(source.length + 1);
  return at.startsWith(`${source}#`) && message !== "" && !message.includes("#");
};

/**
 * @returns the channels that stored holds for read r of source. The channels under the
 * legacy key belong to the message that binds one of them, or else to the first one.
 */
export const of = (
  stored: Record<string, Task.ReadChannelProps>,
  { key: sourceKey, legacy, messages }: Source,
  r: number,
): Task.ReadChannelProps | undefined => {
  const own = stored[key(sourceKey, messages[r].message)];
  if (own != null || legacy == null) return own;
  const held = stored[legacy];
  if (held == null) return undefined;
  const heldChannels = new Set([held.index, ...Object.values(held.channels)]);
  heldChannels.delete(0);
  const owner = messages.findIndex(({ bindings }) =>
    bindings.some(({ channel }) => heldChannels.has(channel)),
  );
  return Math.max(owner, 0) === r ? held : undefined;
};

/**
 * @returns the channel that stored holds for each pointer of source, for a message
 * that the source reads first. The channels of the message itself come first. The
 * channels of a later read of the source stay with that read.
 */
export const channels = (
  stored: Record<string, Task.ReadChannelProps>,
  source: Source,
  message: icd.MessageKey,
): Record<string, channel.Key> => {
  const claimed = new Set(
    source.messages.slice(1).map((_, i) => of(stored, source, i + 1)),
  );
  const merged: Record<string, channel.Key> = {};
  for (const [at, props] of Object.entries(stored))
    if ((at === source.legacy || inSource(at, source.key)) && !claimed.has(props))
      Object.assign(merged, props.channels);
  return { ...merged, ...stored[key(source.key, message)]?.channels };
};

/**
 * Moves the channels under the legacy key of source to the key of read r, when they
 * belong to that read.
 */
export const moveLegacy = (
  stored: Record<string, Task.ReadChannelProps>,
  source: Source,
  r: number,
): void => {
  const at = key(source.key, source.messages[r].message);
  if (stored[at] != null || source.legacy == null) return;
  const held = of(stored, source, r);
  if (held == null) return;
  stored[at] = held;
  delete stored[source.legacy];
};
