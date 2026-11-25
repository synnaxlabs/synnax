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
  /**
   * The key where an older Console stored the channels of the source, if any. split
   * moves them to the keys of its messages.
   */
  legacy?: string;
  /** The messages that the source reads. */
  messages: icd.ReadMessage[];
  /** Whether the source is disabled. */
  disabled?: boolean;
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
 * @returns stored with the channels under each legacy key of sources moved to the keys
 * of the messages that read it. One message, the owner, gets them all with their index:
 * the first enabled message that binds one of them, else the first disabled one, else
 * the first enabled message, else the first message. Each other message gets the
 * channels that it binds without the index, or all of them if it binds none. Only the
 * owner gets the index, because two holders would both keep it, and the Core refuses
 * two writers on one index. A message with a key of its own keeps it. The legacy key
 * stays while a source that reads it has no messages.
 */
export const split = (
  stored: Record<string, Task.ReadChannelProps>,
  sources: Source[],
): Record<string, Task.ReadChannelProps> => {
  const out = { ...stored };
  for (const legacy of new Set(sources.flatMap((s) => s.legacy ?? []))) {
    const held = stored[legacy];
    if (held == null) continue;
    const readers = sources.filter((s) => s.legacy === legacy);
    const heldChannels = new Set([held.index, ...Object.values(held.channels)]);
    heldChannels.delete(0);
    const messages = readers.flatMap(({ key: sourceKey, messages, disabled }) =>
      messages.map(({ message, bindings }) => {
        const bound = new Set(bindings.map(({ channel }) => channel));
        return {
          at: key(sourceKey, message),
          disabled: disabled === true,
          bound,
          binds: [...bound].some((ch) => heldChannels.has(ch)),
        };
      }),
    );
    if (messages.length === 0) continue;
    const owner =
      messages.find(({ disabled, binds }) => !disabled && binds) ??
      messages.find(({ binds }) => binds) ??
      messages.find(({ disabled }) => !disabled) ??
      messages[0];
    out[owner.at] ??= held;
    for (const { at, bound, binds } of messages) {
      const channels = binds
        ? Object.fromEntries(
            Object.entries(held.channels).filter(([, ch]) => bound.has(ch)),
          )
        : held.channels;
      if (Object.keys(channels).length > 0) out[at] ??= { index: 0, channels };
    }
    if (readers.every(({ messages }) => messages.length > 0)) delete out[legacy];
  }
  return out;
};

/** @returns the channels that stored holds for read r of source. */
export const of = (
  stored: Record<string, Task.ReadChannelProps>,
  { key: sourceKey, messages }: Source,
  r: number,
): Task.ReadChannelProps | undefined => stored[key(sourceKey, messages[r].message)];

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
    if (inSource(at, source.key) && !claimed.has(props))
      Object.assign(merged, props.channels);
  return { ...merged, ...stored[key(source.key, message)]?.channels };
};
