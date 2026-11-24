// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { icd, Synnax } from "@synnaxlabs/client";
import { ICD as PICD } from "@synnaxlabs/pluto";

import { encodes } from "@/platform/icd/dataType";
import { formatOf, type Formats } from "@/platform/icd/Formats";
import { createMessage } from "@/platform/icd/inferred";
import { jsonFields, mostRead } from "@/platform/icd/read";

const channelBinding = (field: icd.FieldKey): icd.ChannelWriteBinding => ({
  type: "channel",
  field,
  channel: 0,
});

/**
 * @returns bindings, changed to suit fields: no binding of a field that is gone, no
 * fixed value of another kind, and no data type that its field does not take. Each
 * field with no default and no binding takes a channel. When no field takes one, the
 * first field does, because the Core refuses a message that binds no channels.
 */
export const suit = (
  bindings: icd.WriteBinding[],
  fields: icd.JSONField[],
): icd.WriteBinding[] => {
  const byKey = new Map(fields.map((f) => [f.key, f]));
  const kept = bindings.flatMap((b): icd.WriteBinding[] => {
    const field = byKey.get(b.field);
    if (field == null) return [];
    if (b.type === "value") return b.value.type === field.type ? [b] : [];
    if (b.dataType == null || encodes(field, b.dataType)) return [b];
    return [{ type: "channel", field: b.field, channel: b.channel }];
  });
  const bound = new Set(kept.map(({ field }) => field));
  const suited = [
    ...kept,
    ...fields
      .filter((f) => f.default == null && !bound.has(f.key))
      .map((f) => channelBinding(f.key)),
  ];
  const first = fields.at(0);
  if (first == null || suited.some(({ type }) => type === "channel")) return suited;
  return [channelBinding(first.key), ...suited.filter((b) => b.field !== first.key)];
};

/** @returns the write of message, with the bindings that a new pick starts with. */
export const create = (message: icd.Message): icd.WriteMessage => ({
  message: message.key,
  bindings: suit([], jsonFields(message)),
});

export interface AddMessagesParams {
  client: Synnax;
  /** The open forms of the ICDs that the targets send messages of. */
  formats: Formats;
  /** The targets of the task that send ICD messages. */
  targets: { icd: icd.Key }[];
  /** The name of a new ICD. */
  name: string;
  /** The name and fields of each message to add. */
  messages: { name: string; fields: icd.JSONField[] }[];
}

export interface AddMessagesResult {
  icd: icd.Key;
  /** The messages that the ICD gains, in the order of the messages to add. */
  messages: icd.Message[];
}

/** @returns field, sending the write time when it is a timestamp with no default. */
const sendWriteTime = (field: icd.JSONField): icd.JSONField =>
  field.type === "timestamp" && field.default == null
    ? { ...field, default: { type: "now" } }
    : field;

const createMessages = (
  messages: AddMessagesParams["messages"],
  taken: icd.Message[],
): icd.Message[] => {
  const created: icd.Message[] = [];
  for (const { name, fields } of messages)
    created.push(
      createMessage(name, fields.map(sendWriteTime), [...taken, ...created]),
    );
  return created;
};

/**
 * Adds a JSON message for each of messages to the ICD of the most targets, through its
 * form, or to a new ICD named name when no target sends one. A message name that the
 * ICD holds gets a number. A timestamp field with no default sends the time of each
 * write, so it takes no channel.
 */
export const addMessages = async ({
  client,
  formats,
  targets,
  name,
  messages,
}: AddMessagesParams): Promise<AddMessagesResult> => {
  const home = mostRead(targets);
  if (home == null) {
    const created = createMessages(messages, []);
    const { key } = await client.icds.create({ name, messages: created });
    return { icd: key, messages: created };
  }
  const form = formatOf(formats, home);
  const taken = form.value().messages;
  const created = createMessages(messages, taken);
  form.batch(() => {
    let after = taken.at(-1)?.key;
    for (const m of created) {
      form.insert(PICD.formPath.messages, m, after);
      after = m.key;
    }
  });
  return { icd: home, messages: created };
};
