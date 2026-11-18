// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, type icd, type Synnax } from "@synnaxlabs/client";
import { ICD as PICD } from "@synnaxlabs/pluto";
import { DataType, deep, type payload, unique } from "@synnaxlabs/x";

import { converts } from "@/platform/icd/dataType";

/** Writes the value of one message field to a channel. */
export interface ReadBinding {
  field: icd.FieldKey;
  channel: channel.Key;
  /** The data type of the channel that deploying creates. Empty takes the default. */
  dataType: string;
}

/** Reads the values of one ICD message. */
export interface ReadMessage {
  message: icd.MessageKey;
  /** The field whose channel indexes the others. Absent when the source stamps it. */
  index?: icd.FieldKey;
  bindings: ReadBinding[];
}

/** A source of a read task, such as a topic or a record, that reads ICD messages. */
export interface ReadSource {
  icd: icd.Key;
  disabled: boolean;
  messages: ReadMessage[];
}

/** The messages of each ICD, by ICD key. */
export type Messages = Map<icd.Key, icd.Message[]>;

/** @returns the path of the fields of a JSON message in the form of its ICD. */
export const fieldsPath = (key: icd.MessageKey) =>
  deep.variant(PICD.formPath.messages[key].payload, "format", "json").fields;

export const jsonFields = (message?: icd.Message | null): icd.JSONField[] =>
  message?.payload.format === "json" ? message.payload.fields : [];

export const retrieveMessages = async (
  client: Synnax,
  keys: icd.Key[],
): Promise<Messages> => {
  const icds = await client.icds.retrieve(unique.unique(keys));
  return new Map(icds.map(({ key, messages }) => [key, messages]));
};

/** @returns the fields that a read gives channels: the bound ones and the index. */
export const readFields = (
  fields: icd.JSONField[],
  { index, bindings }: Pick<ReadMessage, "index" | "bindings">,
): icd.JSONField[] =>
  fields.filter(
    ({ key }) => key === index || bindings.some(({ field }) => field === key),
  );

/** @returns whether the Core can read the field: a bytes field needs a pointer. */
export const isReadable = ({ type, pointer }: icd.JSONField): boolean =>
  type !== "bytes" || pointer !== "";

export const isIndexable = (field?: icd.JSONField): boolean =>
  field?.type === "timestamp" && field.default == null;

/** @returns the data types that a channel of a field of the kind cannot take. */
export const hiddenDataTypes = (kind: payload.Kind): DataType[] =>
  DataType.ALL.filter((dt) => !converts(kind, dt.toString()));

/**
 * @returns read, changed to suit the fields of its message: no binding of a field that
 * is gone or that the Core cannot read, no data type that its field cannot convert to,
 * and no index that is not a timestamp field. An index field with a default stays, for
 * its error to show.
 */
export const suit = (read: ReadMessage, fields: icd.JSONField[]): ReadMessage => {
  const byKey = new Map(fields.map((f) => [f.key, f]));
  const indexField = read.index == null ? undefined : byKey.get(read.index);
  const index = indexField?.type === "timestamp" ? read.index : undefined;
  const bindings = read.bindings.flatMap((b) => {
    const field = byKey.get(b.field);
    // The binding of a former index field writes to an index channel.
    if (
      field == null ||
      !isReadable(field) ||
      (b.field === read.index && index == null)
    )
      return [];
    if (b.dataType === "" || converts(field.type, b.dataType)) return [b];
    return [{ ...b, dataType: "" }];
  });
  return { message: read.message, ...(index == null ? {} : { index }), bindings };
};

/**
 * @returns read, indexed on the field of next, or on nothing. The index field reads
 * into indexChannel.
 */
export const reindex = (
  read: ReadMessage,
  next: icd.FieldKey | undefined,
  indexChannel: channel.Key,
): ReadMessage => {
  const bindings = read.bindings.filter(
    ({ field }) => field !== read.index && field !== next,
  );
  if (next == null) return { message: read.message, bindings };
  const binding = { field: next, channel: indexChannel, dataType: "" };
  return { message: read.message, index: next, bindings: [...bindings, binding] };
};
