// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { channel, icd, Synnax } from "@synnaxlabs/client";
import { ICD as PICD } from "@synnaxlabs/pluto";

import { bindingDataType, encodes } from "@/platform/icd/dataType";
import { formatOf, type Formats } from "@/platform/icd/Formats";
import { createMessage } from "@/platform/icd/inferred";
import { jsonFields, mostRead, retrieveMessages } from "@/platform/icd/read";
import { Task } from "@/platform/task";

const channelBinding = (field: icd.FieldKey): icd.ChannelWriteBinding => ({
  type: "channel",
  field,
  channel: 0,
});

/** Tells the user what to do when the ICD no longer has the message of a write. */
export const MISSING_DESCRIPTION =
  "The ICD no longer has this message. Pick another message format.";

/** @returns message with no command channels, so deploying creates new ones. */
export const unbind = (message: icd.WriteMessage): icd.WriteMessage => ({
  ...message,
  bindings: message.bindings.map((b) =>
    b.type === "channel" ? { ...b, channel: 0 } : b,
  ),
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

const createMessages = (
  messages: AddMessagesParams["messages"],
  taken: icd.Message[],
): icd.Message[] => {
  const created: icd.Message[] = [];
  for (const { name, fields } of messages)
    created.push(createMessage(name, fields, [...taken, ...created]));
  return created;
};

/**
 * Adds a JSON message for each of messages to the ICD of the most targets, through its
 * form, or to a new ICD named name when no target sends one. A message name that the
 * ICD holds gets a number.
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

/** Keys and names the command channel of each field of a write. */
export interface ChannelNaming {
  /** @returns the key of the field's channel in the write record of the device. */
  key: (field: icd.JSONField) => string;
  /** @returns the name of a new command channel for the field. */
  name: (field: icd.JSONField) => string;
}

/**
 * Gives each channel binding of write a command channel that its field takes: the one
 * it has, the one that record holds at its key, or a new one. A new channel takes the
 * data type of the binding, or the default of its field when the binding names none or
 * one that the field no longer takes. Changes the bindings of write and the channels of
 * record in place.
 */
export const configureChannels = async (
  client: Synnax,
  record: Record<string, channel.Key>,
  write: { icd: icd.Key; message: icd.WriteMessage },
  naming: ChannelNaming,
): Promise<void> => {
  const bindings = write.message.bindings.filter(
    (b): b is icd.ChannelWriteBinding => b.type === "channel",
  );
  if (bindings.length === 0) return;
  const messages = await retrieveMessages(client, [write.icd]);
  const fields = jsonFields(
    messages.get(write.icd)?.find(({ key }) => key === write.message.message),
  );
  for (const binding of bindings) {
    const field = fields.find(({ key }) => key === binding.field);
    // The Core rejects a binding to a field that the message lacks.
    if (field == null) continue;
    const { dataType = "" } = binding;
    const suited = dataType !== "" && encodes(field, dataType);
    binding.channel = await Task.configureCommandChannel(client, record, {
      propertiesKey: naming.key(field),
      channel: binding.channel,
      name: naming.name(field),
      dataType: bindingDataType(suited ? binding : {}, field),
      suits: (ch) => encodes(field, ch.dataType.toString()),
    });
  }
};
