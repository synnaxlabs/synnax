// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { icd, kafka } from "@synnaxlabs/client";
import { DataType, type json, uuid } from "@synnaxlabs/x";

import { type ReadRecord } from "@/feature/kafka/task/types";
import { ICD } from "@/platform/icd";

/** The pointers of the fields picked from each shape of a sample, by shape key. */
export type Picks = Map<string, Set<string>>;

/**
 * @returns a key for a shape that stays the same across samples while its fields do.
 * The Core groups the keys of a topic into shapes by their set of fields.
 */
export const shapeKey = ({ fields }: json.Shape): string =>
  fields
    .map(({ pointer }) => pointer)
    .sort()
    .join("\n");

/** A message that adding picks puts in the task's ICD, and the records that read it. */
export interface Addition {
  message: icd.Message;
  /** The Kafka keys of the new records that read the message. */
  recordKeys: string[];
  /** The key of the field that indexes the message. Absent when Kafka stamps it. */
  index?: icd.FieldKey;
}

/** @returns the fields that a click on a shape picks: each one but its index. */
export const defaultPicks = (shape: json.Shape): Set<string> =>
  new Set(
    shape.fields.filter((_, i) => i !== shape.index).map(({ pointer }) => pointer),
  );

/** @returns the Kafka keys of a shape that no record of the task reads yet. */
export const freeKeys = (shape: json.Shape, records: ReadRecord[]): string[] =>
  shape.sources.filter((key) => !records.some(({ recordKey }) => recordKey === key));

const toJSONField = (
  { pointer, type, dataType, timeFormat }: json.InferredField,
  name: string,
): icd.JSONField => {
  const base = { key: uuid.create(), name, pointer };
  if (timeFormat != null)
    return icd.jsonFieldZ.parse({ ...base, type: "timestamp", timeFormat });
  if (type === "boolean" || type === "string")
    return icd.jsonFieldZ.parse({ ...base, type });
  const dt = new DataType(dataType);
  let number: "integer" | "unsigned" | "float" = "float";
  if (dt.equals(DataType.INT64)) number = "integer";
  else if (dt.equals(DataType.UINT64)) number = "unsigned";
  return icd.jsonFieldZ.parse({ ...base, type: number });
};

/** @returns base, numbered from 2 until no message has the name. */
export const uniqueName = (base: string, taken: icd.Message[]): string => {
  const names = new Set(taken.map(({ name }) => name));
  if (!names.has(base)) return base;
  let n = 2;
  while (names.has(`${base} ${n}`)) n++;
  return `${base} ${n}`;
};

/**
 * @returns a message for each shape with picks, holding the picked fields and the
 * shape's timestamp. Its records read the keys of the shape that the task does not
 * read yet. A message takes the name of its first key, or label for the empty key.
 */
export const createAdditions = (
  shapes: json.Shape[],
  picks: Picks,
  records: ReadRecord[],
  held: icd.Message[],
  label: (recordKey: string) => string,
): Addition[] => {
  const taken = [...held];
  return shapes.flatMap((shape): Addition[] => {
    const pointers = picks.get(shapeKey(shape));
    const recordKeys = freeKeys(shape, records);
    if (pointers == null || pointers.size === 0 || recordKeys.length === 0) return [];
    const indexField = shape.index == null ? undefined : shape.fields[shape.index];
    const picked = shape.fields.filter(
      (f) => f === indexField || pointers.has(f.pointer),
    );
    const names = ICD.fieldNames(picked.map(({ pointer }) => pointer));
    const fields = picked.map((f, i) => toJSONField(f, names[i]));
    const index =
      indexField == null
        ? undefined
        : fields.find(({ pointer }) => pointer === indexField.pointer)?.key;
    const message: icd.Message = {
      key: uuid.create(),
      name: uniqueName(label(recordKeys[0]), taken),
      payload: { format: "json", fields },
    };
    taken.push(message);
    return [{ message, recordKeys, ...(index == null ? {} : { index }) }];
  });
};

/** @returns a record for each Kafka key of each addition, reading all of its fields. */
export const createRecords = (icdKey: icd.Key, additions: Addition[]): ReadRecord[] =>
  additions.flatMap(({ message, recordKeys, index }) => {
    const fields = message.payload.format === "json" ? message.payload.fields : [];
    const read = kafka.readMessageZ.parse({
      message: message.key,
      index,
      bindings: fields.map(({ key }) => ({ field: key })),
    });
    return recordKeys.map((recordKey) =>
      kafka.readRecordZ.parse({ icd: icdKey, recordKey, messages: [read] }),
    );
  });
