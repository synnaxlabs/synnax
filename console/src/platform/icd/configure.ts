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

import { bindingDataType, converts, DEFAULT_DATA_TYPES } from "@/platform/icd/dataType";
import {
  isIndexable,
  type Messages,
  type ReadBinding,
  readFields,
  type ReadSource,
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
  return { source, read, message, fields };
};

/** A field that a message reads, with the binding that deploying writes for it. */
export interface MessageField extends Task.ReadChannelField {
  binding: ReadBinding;
}

/** The channels that deploying gives the fields that a message reads. */
export interface MessagePlan {
  /** The key of the index field, if the message has one. */
  index?: string;
  fields: MessageField[];
  channels: Task.ReadChannelPlan;
}

/**
 * Plans the channel of each field that a message reads: the one it has, the one stored
 * for its pointer, or a new one of the binding's data type. A channel that no longer
 * suits its field is replaced, and leaves the stored channels. Reads from the Core, and
 * changes neither bound nor props.
 *
 * @param namer - Names the new channels of every source that the task plans.
 * @param prefix - The prefix of the names of the channels it creates.
 * @param props - What the device stores for the message.
 */
export const planMessage = async (
  client: Synnax,
  namer: Task.Namer,
  { source, read: r, fields }: BoundMessage<ReadSource>,
  prefix: string,
  props?: Task.ReadChannelProps,
): Promise<MessagePlan> => {
  const read = source.messages[r];
  const index = fields.find(({ key }) => key === read.index)?.key;
  const held = { ...props?.channels };
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
  // A message stores every field on one index when it has an index field, a stored
  // channel, or a fixed-density field that needs a new channel. Else it streams them.
  const indexed =
    index != null ||
    fields.some((f, i) => {
      const ch = live.get(reused[i]);
      if (ch != null && !ch.virtual) return true;
      if (ch != null && converts(f.type, ch.dataType.toString())) return false;
      return !new DataType(DEFAULT_DATA_TYPES[f.type]).isVariable;
    });
  const planned = fields.map((f, i): MessageField => {
    const binding = bindingOf(f);
    const isIndex = f.key === index;
    const suited = binding.dataType !== "" && converts(f.type, binding.dataType);
    const dataType = suited ? binding.dataType : "";
    const ch = live.get(reused[i]);
    // An edit of the field's kind or index role since the last deploy, or a streamed
    // field that the message now stores, needs a new channel.
    const stale =
      ch != null &&
      (!converts(f.type, ch.dataType.toString()) ||
        ch.isIndex !== isIndex ||
        (indexed && ch.virtual));
    if (stale && held[f.pointer] === ch.key) delete held[f.pointer];
    return {
      key: f.key,
      pointer: f.pointer,
      name: `${prefix}_${channel.escapeInvalidName(f.name)}`,
      dataType: isIndex
        ? DataType.TIMESTAMP.toString()
        : bindingDataType({ dataType }, f.type),
      channel: stale ? 0 : binding.channel,
      replaces: stale ? ch.key : 0,
      binding: { ...binding, dataType },
    };
  });
  const channels = await Task.planReadChannels({
    client,
    namer,
    props: props == null ? undefined : { ...props, channels: held },
    fields: planned,
    namePrefix: prefix,
    indexKey: index ?? "",
    indexed: () => indexed,
  });
  return { index, fields: planned, channels };
};

/**
 * @returns the channels that a message binds once deployed, other than the ones that
 * deploying creates.
 */
export const keptChannels = ({ channels: plan }: MessagePlan): channel.Key[] => {
  const kept = Object.values(plan.channels);
  if (plan.indexField != null && plan.newIndex == null && plan.props != null)
    kept.push(plan.props.index);
  return kept;
};

/**
 * Gives each field that a message reads its channel, as plan plans it. Rewrites the
 * read of the source in place.
 *
 * @param plan - The plan of planMessage for bound.
 */
export const applyMessage = async (
  client: Synnax,
  { source, read: r }: BoundMessage<ReadSource>,
  { index, fields, channels }: MessagePlan,
  stored?: Task.StoredChannels,
): Promise<void> => {
  await Task.applyReadChannels(client, channels, fields, stored);
  source.messages[r] = {
    message: source.messages[r].message,
    index,
    bindings: fields.map(({ channel, binding }) => ({ ...binding, channel })),
  };
};
