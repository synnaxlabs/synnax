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
import { DataType, deep, unique } from "@synnaxlabs/x";

import { converts } from "@/platform/icd/dataType";

/** A source of a read task, such as a topic or a record, that reads ICD messages. */
export interface ReadSource {
  icd: icd.Key;
  disabled: boolean;
  messages: icd.ReadMessage[];
}

/**
 * @returns the ICD that takes the messages of new sources: the one that the most
 * sources read, or the first of those. Undefined when no source reads one.
 */
export const mostRead = (sources: Pick<ReadSource, "icd">[]): icd.Key | undefined => {
  const counts = new Map<icd.Key, number>();
  for (const { icd: key } of sources) counts.set(key, (counts.get(key) ?? 0) + 1);
  let most: icd.Key | undefined;
  let count = 0;
  for (const [key, n] of counts)
    if (n > count) {
      most = key;
      count = n;
    }
  return most;
};

/** The messages of each ICD, by ICD key. */
export type Messages = Map<icd.Key, icd.Message[]>;

/** A field of a message of any format. */
export type Field = icd.JSONField | icd.TextField | icd.BinaryField;

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
export const readFields = <F extends Field>(
  fields: F[],
  { index, bindings }: Pick<icd.ReadMessage, "index" | "bindings">,
): F[] =>
  fields.filter(
    ({ key }) => key === index || bindings.some(({ field }) => field === key),
  );

/**
 * What a read tree shows of a field. It holds only JSON values, so a tree can compare
 * its structure as a string, and an edit to the rest of the field keeps the structure.
 */
export interface TreeField extends Pick<icd.BaseField, "key" | "name"> {
  readable: boolean;
}

/** @returns whether the Core can read the field: a JSON bytes field needs a pointer. */
export const isReadable = (field: Field): boolean =>
  !("type" in field) || field.type !== "bytes" || field.pointer !== "";

export const treeFieldOf = (field: Field): TreeField => ({
  key: field.key,
  name: field.name,
  readable: isReadable(field),
});

/** @returns whether a read can index on the field: a JSON timestamp with no default. */
export const isIndexable = (field?: Field): boolean =>
  field != null &&
  "type" in field &&
  field.type === "timestamp" &&
  field.default == null;

/** @returns the data types that a channel of field cannot take. */
export const hiddenDataTypes = (field: Field): DataType[] =>
  DataType.ALL.filter((dt) => !converts(field, dt.toString()));

/**
 * @returns read, changed to suit the fields of its message: no binding of a field that
 * is gone or that the Core cannot read, no data type that its field cannot convert to,
 * and no index that is not a timestamp field. An index field with a default stays, for
 * its error to show.
 */
export const suit = (read: icd.ReadMessage, fields: Field[]): icd.ReadMessage => {
  const byKey = new Map(fields.map((f) => [f.key, f]));
  const indexField = read.index == null ? undefined : byKey.get(read.index);
  const index =
    indexField != null && "type" in indexField && indexField.type === "timestamp"
      ? read.index
      : undefined;
  const bindings = read.bindings.flatMap((b) => {
    const field = byKey.get(b.field);
    // The binding of a former index field writes to an index channel.
    if (
      field == null ||
      !isReadable(field) ||
      (b.field === read.index && index == null)
    )
      return [];
    const { dataType, ...unTyped } = b;
    if (dataType == null || converts(field, dataType)) return [b];
    return [unTyped];
  });
  return { message: read.message, ...(index == null ? {} : { index }), bindings };
};

/**
 * @returns read, indexed on the field of next, or on nothing. The index field reads
 * into indexChannel.
 */
export const reindex = (
  read: icd.ReadMessage,
  next: icd.FieldKey | undefined,
  indexChannel: channel.Key,
): icd.ReadMessage => {
  const bindings = read.bindings.filter(
    ({ field }) => field !== read.index && field !== next,
  );
  if (next == null) return { message: read.message, bindings };
  const binding = { field: next, channel: indexChannel };
  return { message: read.message, index: next, bindings: [...bindings, binding] };
};
