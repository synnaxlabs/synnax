// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, type icd, type Synnax, ValidationError } from "@synnaxlabs/client";
import { DataType } from "@synnaxlabs/x";

import { bindingDataType, converts } from "@/platform/icd/dataType";
import {
  isIndexable,
  type Messages,
  mixMessage,
  type ReadBinding,
  readFields,
  type ReadSource,
  variableAmongFixed,
} from "@/platform/icd/read";
import { Task } from "@/platform/task";

/** @returns a message that the source reads, from the source's own ICD. */
export const messageOf = (
  messages: Messages,
  source: ReadSource,
  read: number,
): icd.Message | undefined =>
  messages
    .get(source.icd)
    ?.find(({ key }) => key === source.messages.at(read)?.message);

/** A message that a source reads, with the fields that the task reads of it. */
export interface BoundMessage<S extends ReadSource> {
  source: S;
  /** The index of the message in the source's reads. */
  read: number;
  message: icd.Message;
  fields: icd.JSONField[];
}

/**
 * @param name - Names the source in an error, such as its topic.
 * @returns a message that the source reads, with the fields that the task reads of it.
 * @throws {ValidationError} if the task cannot read the message.
 */
export const boundMessage = <S extends ReadSource>(
  messages: Messages,
  source: S,
  read: number,
  name: string,
): BoundMessage<S> => {
  const message = messageOf(messages, source, read);
  const where =
    message != null && source.messages.length > 1 ? `${message.name} of ${name}` : name;
  if (message?.payload.format !== "json")
    throw new ValidationError(`The message format of ${where} is missing or not JSON`);
  const readMessage = source.messages[read];
  const fields = readFields(message.payload.fields, readMessage);
  const index = fields.find(({ key }) => key === readMessage.index);
  if (index != null && !isIndexable(index))
    throw new ValidationError(
      `In ${where}, the index field ${index.name} must be a timestamp field with no default`,
    );
  if (fields.length === 0)
    throw new ValidationError(`The task reads no fields of ${where}`);
  const mixed = variableAmongFixed(fields);
  if (mixed.length > 0) throw new ValidationError(`In ${where}, ${mixMessage(mixed)}`);
  return { source, read, message, fields };
};

/**
 * Gives each field that a message reads a channel: the one it has, the one stored for
 * its pointer, or a new one of the binding's data type. A channel that no longer suits
 * its field is replaced, and leaves stored. Rewrites the read of the source in place.
 *
 * @param prefix - The prefix of the names of the channels it creates.
 * @returns the channels that the message binds.
 */
export const configureMessage = async (
  client: Synnax,
  { source, read: r, fields }: BoundMessage<ReadSource>,
  prefix: string,
  stored?: Task.StoredChannels,
): Promise<channel.Key[]> => {
  const read = source.messages[r];
  const index = fields.find(({ key }) => key === read.index)?.key;
  const held = stored?.read[stored.key]?.channels ?? {};
  const bindingOf = (f: icd.JSONField): ReadBinding =>
    read.bindings.find(({ field }) => field === f.key) ?? {
      field: f.key,
      channel: 0,
      dataType: "",
    };
  const reused = fields.map((f) => bindingOf(f).channel || (held[f.pointer] ?? 0));
  const live = new Map(
    (await client.channels.retrieve(reused)).map((ch) => [ch.key, ch]),
  );
  const channels = fields.map((f, i) => {
    const binding = bindingOf(f);
    const isIndex = f.key === index;
    const suited = binding.dataType !== "" && converts(f.type, binding.dataType);
    const dataType = suited ? binding.dataType : "";
    const ch = live.get(reused[i]);
    // An edit of the field's kind or index role since the last deploy needs a new
    // channel.
    const stale =
      ch != null &&
      (!converts(f.type, ch.dataType.toString()) || ch.isIndex !== isIndex);
    if (stale && held[f.pointer] === ch.key) delete held[f.pointer];
    return {
      key: f.key,
      pointer: f.pointer,
      name: `${prefix}_${channel.escapeInvalidName(f.name)}`,
      dataType: isIndex
        ? DataType.TIMESTAMP.toString()
        : bindingDataType({ dataType }, f.type),
      channel: stale ? 0 : binding.channel,
      binding: { ...binding, dataType },
    };
  });
  await Task.configureReadChannels({
    client,
    stored,
    fields: channels,
    namePrefix: prefix,
    indexKey: index ?? "",
  });
  source.messages[r] = {
    message: read.message,
    index,
    bindings: channels.map(({ channel, binding }) => ({ ...binding, channel })),
  };
  return channels.map(({ channel }) => channel);
};
