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

const isVariable = ({ dataType }: json.InferredField): boolean =>
  new DataType(dataType).isVariable;

/**
 * @returns whether a pick of a shape can take the field. The fields of a message share
 * one index, which a variable-length field cannot have, so the two kinds never mix.
 */
export const isPickable = (
  shape: json.Shape,
  picked: Set<string>,
  field: json.InferredField,
): boolean =>
  shape.fields
    .filter((f, i) => i === shape.index || picked.has(f.pointer))
    .every((f) => isVariable(f) === isVariable(field));

/** @returns the messages of an ICD, by its key. */
export type MessagesOf = (icdKey: icd.Key) => icd.Message[];

/** What the records of the task read, by Kafka key. */
export interface Reads {
  /** The pointers of the fields that the records of each key read. */
  pointers: Map<string, Set<string>>;
  /** The keys with a record that reads no JSON message. */
  jsonless: Set<string>;
}

/** @returns the index of the first JSON message that a record reads, or -1 for none. */
const jsonRead = (record: ReadRecord, held: icd.Message[]): number =>
  record.messages.findIndex(
    ({ message }) => held.find(({ key }) => key === message)?.payload.format === "json",
  );

/** @returns the pointers of the fields that a record binds, in any of its messages. */
const pointersOf = (record: ReadRecord, messagesOf: MessagesOf): Set<string> => {
  const held = messagesOf(record.icd);
  return new Set(
    record.messages.flatMap(({ message, bindings }) => {
      const fields = ICD.jsonFields(held.find(({ key }) => key === message));
      return fields
        .filter(({ key }) => bindings.some(({ field }) => field === key))
        .map(({ pointer }) => pointer);
    }),
  );
};

/** @returns what the records of each Kafka key read. */
export const readsOf = (records: ReadRecord[], messagesOf: MessagesOf): Reads => {
  const reads: Reads = { pointers: new Map(), jsonless: new Set() };
  for (const record of records) {
    const { recordKey } = record;
    if (jsonRead(record, messagesOf(record.icd)) < 0) reads.jsonless.add(recordKey);
    const pointers = reads.pointers.get(recordKey) ?? new Set<string>();
    pointersOf(record, messagesOf).forEach((p) => pointers.add(p));
    reads.pointers.set(recordKey, pointers);
  }
  return reads;
};

/** @returns whether a pick of a shape can extend each record of its keys. */
export const isExtendable = (shape: json.Shape, { jsonless }: Reads): boolean =>
  !shape.sources.some((key) => jsonless.has(key));

/** @returns the pointers that every key of a shape reads: its fields in the task. */
export const takenPointers = (shape: json.Shape, { pointers }: Reads): Set<string> => {
  const [first, ...rest] = shape.sources.map(
    (key) => pointers.get(key) ?? new Set<string>(),
  );
  return new Set([...(first ?? [])].filter((p) => rest.every((r) => r.has(p))));
};

/**
 * @returns the pointers that any key of a shape reads. Their messages gain the picks of
 * the shape, so the picks must not mix variable-length fields among them.
 */
export const heldPointers = (shape: json.Shape, { pointers }: Reads): Set<string> =>
  new Set(shape.sources.flatMap((key) => [...(pointers.get(key) ?? [])]));

/**
 * @returns the fields that a click on a shape picks: each one that a key of the shape
 * does not read and that the shape can take, without mixing variable-length fields
 * among fixed ones. Fixed fields go first. A shape that is not extendable picks none.
 */
export const defaultPicks = (shape: json.Shape, reads: Reads): Set<string> => {
  if (!isExtendable(shape, reads)) return new Set();
  const taken = takenPointers(shape, reads);
  const held = heldPointers(shape, reads);
  const free = shape.fields.filter(
    (f, i) => i !== shape.index && !taken.has(f.pointer) && isPickable(shape, held, f),
  );
  const fixed = free.filter((f) => !isVariable(f));
  return new Set((fixed.length > 0 ? fixed : free).map(({ pointer }) => pointer));
};

/** @returns the Kafka keys of a shape that no record of the task reads yet. */
const freeKeys = (shape: json.Shape, records: ReadRecord[]): string[] =>
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

/** Picks that a message gains, because records of the task read the message already. */
export interface Extension {
  icd: icd.Key;
  message: icd.MessageKey;
  /** The fields that the message gains. */
  fields: icd.JSONField[];
  /** The field keys that each record starts to bind, by record and read index. */
  bindings: { record: string; read: number; fields: icd.FieldKey[] }[];
}

/**
 * @returns the picks that the records of the task gain, in the first JSON message that
 * each record reads. A message gains each picked field that it does not hold, once for
 * all of its records. A record starts to read each picked field that it does not.
 * @throws if a picked shape is not extendable.
 */
export const createExtensions = (
  shapes: json.Shape[],
  picks: Picks,
  records: ReadRecord[],
  messagesOf: MessagesOf,
): Extension[] => {
  const extensions = new Map<icd.MessageKey, Extension>();
  for (const shape of shapes) {
    const pointers = picks.get(shapeKey(shape));
    if (pointers == null) continue;
    for (const record of records) {
      if (!shape.sources.includes(record.recordKey)) continue;
      const held = messagesOf(record.icd);
      const read = jsonRead(record, held);
      if (read < 0) throw new Error(`Record ${record.key} reads no JSON message`);
      const { message } = record.messages[read];
      const bound = pointersOf(record, messagesOf);
      const picked = shape.fields.filter(
        (f) => pointers.has(f.pointer) && !bound.has(f.pointer),
      );
      if (picked.length === 0) continue;
      const extension = extensions.get(message) ?? {
        icd: record.icd,
        message,
        fields: [],
        bindings: [],
      };
      const holds = [
        ...ICD.jsonFields(held.find(({ key }) => key === message)),
        ...extension.fields,
      ];
      const missing = picked.filter((f) => !holds.some((h) => h.pointer === f.pointer));
      const names = ICD.fieldNames(
        missing.map(({ pointer }) => pointer),
        holds.map(({ name }) => name),
      );
      const added = missing.map((f, i) => toJSONField(f, names[i]));
      extension.fields.push(...added);
      const existing = holds.filter((h) => picked.some((f) => f.pointer === h.pointer));
      extension.bindings.push({
        record: record.key,
        read,
        fields: [...existing, ...added].map(({ key }) => key),
      });
      extensions.set(message, extension);
    }
  }
  return [...extensions.values()];
};
