// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { icd, kafka } from "@synnaxlabs/client";
import { json, map, uuid } from "@synnaxlabs/x";

import type { ReadRecord } from "@/feature/kafka/task/types";
import { ICD } from "@/platform/icd";

/** The pointers of the fields picked from each shape of a sample, by shape key. */
export type Picks = Map<string, Set<string>>;

/** A message that Add data puts in the task's ICD, and the records that read it. */
export interface Addition {
  message: icd.Message;
  /** The Kafka keys of the new records that read the message. */
  recordKeys: string[];
  /** The key of the field that indexes the message. Absent when Kafka stamps it. */
  index?: icd.FieldKey;
}

/** @returns the messages of an ICD, by its key. */
export type MessagesOf = (icdKey: icd.Key) => icd.Message[];

/** What a record reads in one of its JSON messages. */
interface MessageRead {
  /** The position of the message in the reads of the record. */
  read: number;
  match?: icd.JSONMatch;
  /** The fields that the record reads in the message, by pointer. */
  fields: Map<string, icd.JSONField>;
}

/** What the records of the task read, by Kafka key. */
export interface Reads {
  /** What the records of each key read, in each of their JSON messages. */
  messages: Map<string, MessageRead[]>;
  /** The keys with a record that reads no JSON message. */
  jsonless: Set<string>;
}

/** @returns what a record reads in each of its JSON messages. */
const messageReads = (record: ReadRecord, held: icd.Message[]): MessageRead[] =>
  record.messages.flatMap(({ message, bindings }, read): MessageRead[] => {
    const { payload } = held.find(({ key }) => key === message) ?? {};
    if (payload?.format !== "json") return [];
    const bound = new Set(bindings.map(({ field }) => field));
    const fields = payload.fields.filter(({ key }) => bound.has(key));
    return [
      {
        read,
        match: payload.match,
        fields: new Map(fields.map((f) => [f.pointer, f])),
      },
    ];
  });

/** @returns the message of messages that the Core gives the documents of shape. */
const routeOf = (shape: json.Shape, messages: MessageRead[]) =>
  ICD.route(shape.fields, messages, ({ fields }) => fields.keys());

/** @returns the messages that the records of a Kafka key read. */
const messagesOfKey = ({ messages }: Reads, key: string): MessageRead[] =>
  messages.get(key) ?? messages.get("") ?? [];

/** @returns what the records of each Kafka key read. */
export const readsOf = (records: ReadRecord[], messagesOf: MessagesOf): Reads => {
  const reads: Reads = { messages: new Map(), jsonless: new Set() };
  for (const record of records) {
    const own = messageReads(record, messagesOf(record.icd));
    if (own.length === 0) reads.jsonless.add(record.recordKey);
    map.getOrInsert(reads.messages, record.recordKey, () => []).push(...own);
  }
  return reads;
};

/**
 * @returns the fields of a shape that the record of the empty key reads for the keys of
 * the shape with no record of their own, by pointer. That is in the message that the
 * Core gives the documents of the shape. A new record of those keys reads them too, so
 * the keys keep their fields when they leave that record.
 */
export const carriedFields = (
  shape: json.Shape,
  { messages }: Reads,
): Map<string, icd.JSONField> => {
  const carried = new Map<string, icd.JSONField>();
  if (shape.sources.every((key) => messages.has(key))) return carried;
  const routed = routeOf(shape, messages.get("") ?? []).message?.fields;
  shape.fields.forEach(({ pointer }, i) => {
    const field = routed?.get(pointer);
    if (i !== shape.index && field != null) carried.set(pointer, field);
  });
  return carried;
};

/**
 * @returns the pointer of the value that tells the documents of a shape apart, when the
 * fields of the shape do not tell which message of a key gets them.
 */
export const splitOf = (shape: json.Shape, reads: Reads): string | undefined =>
  shape.sources
    .map((key) => routeOf(shape, messagesOfKey(reads, key)).split)
    .find((split) => split != null);

/** @returns whether a pick of a shape can extend each record of its keys. */
export const isExtendable = (shape: json.Shape, reads: Reads): boolean =>
  !shape.sources.some((key) => reads.jsonless.has(key)) &&
  splitOf(shape, reads) == null;

/**
 * @returns the pointers that every key of a shape reads in the message that the Core
 * gives the documents of the shape: its fields in the task. A key with no record of its
 * own reads what the record of the empty key reads.
 */
export const takenPointers = (shape: json.Shape, reads: Reads): Set<string> => {
  const [first, ...rest] = shape.sources.map(
    (key) => new Set(routeOf(shape, messagesOfKey(reads, key)).message?.fields.keys()),
  );
  return new Set([...(first ?? [])].filter((p) => rest.every((r) => r.has(p))));
};

/**
 * @returns the fields that a click on a shape picks: each one but its index that a key
 * of the shape does not read. A shape that is not extendable picks none.
 */
export const defaultPicks = (shape: json.Shape, reads: Reads): Set<string> => {
  if (!isExtendable(shape, reads)) return new Set();
  const taken = takenPointers(shape, reads);
  return new Set(
    shape.fields
      .filter((f, i) => i !== shape.index && !taken.has(f.pointer))
      .map(({ pointer }) => pointer),
  );
};

/** @returns the Kafka keys of a shape with no record of their own. */
const freeKeys = (shape: json.Shape, { messages }: Reads): string[] =>
  shape.sources.filter((key) => !messages.has(key));

/** @returns the field that tells the documents of shape apart from other kinds. */
const presentOf = (shape: json.Shape): json.InferredField | undefined =>
  shape.present == null ? undefined : shape.fields[shape.present];

/** @returns a match for the documents of shape, or nothing when it is the only kind. */
const matchOf = (shape: json.Shape): Pick<icd.JSONPayload, "match"> => {
  const present = presentOf(shape);
  return present == null
    ? {}
    : { match: { type: "present", pointer: present.pointer } };
};

/** @returns a read of message with all of its fields. */
const readOf = (message: icd.Message, index?: icd.FieldKey): icd.ReadMessage => {
  const fields = message.payload.format === "json" ? message.payload.fields : [];
  return icd.readMessageZ.parse({
    message: message.key,
    index,
    bindings: fields.map(({ key }) => ({ field: key })),
  });
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
 * @returns a message for each shape with picks, holding the picked fields, the fields
 * that its keys carry, and the shape's timestamp. Its records read the keys of the
 * shape with no record of their own. A message takes the name of its first key, or
 * label for the empty key. A shape that is one of several kinds adds the name of its
 * present field, and its message matches the documents that hold that field.
 */
export const createAdditions = (
  shapes: json.Shape[],
  picks: Picks,
  reads: Reads,
  held: icd.Message[],
  label: (recordKey: string) => string,
): Addition[] => {
  const taken = [...held];
  return shapes.flatMap((shape): Addition[] => {
    const pointers = picks.get(json.shapeKey(shape));
    const recordKeys = freeKeys(shape, reads);
    if (pointers == null || pointers.size === 0 || recordKeys.length === 0) return [];
    const carried = carriedFields(shape, reads);
    const indexField = shape.index == null ? undefined : shape.fields[shape.index];
    const picked = shape.fields.filter(
      (f) => f === indexField || (pointers.has(f.pointer) && !carried.has(f.pointer)),
    );
    const names = ICD.fieldNames(
      picked.map(({ pointer }) => pointer),
      [...carried.values()].map(({ name }) => name),
    );
    const named = new Map(picked.map((f, i) => [f, ICD.createField(f, names[i])]));
    const fields = shape.fields.flatMap((f): icd.JSONField[] => {
      const field = carried.get(f.pointer);
      if (field != null) return [{ ...field, key: uuid.create() }];
      const own = named.get(f);
      return own == null ? [] : [own];
    });
    const index =
      indexField == null
        ? undefined
        : fields.find(({ pointer }) => pointer === indexField.pointer)?.key;
    const message: icd.Message = {
      key: uuid.create(),
      name: uniqueName(
        ICD.kindName(label(recordKeys[0]), presentOf(shape)?.pointer),
        taken,
      ),
      payload: { format: "json", fields, ...matchOf(shape) },
    };
    taken.push(message);
    return [{ message, recordKeys, ...(index == null ? {} : { index }) }];
  });
};

/**
 * @returns a record for each Kafka key of the additions, reading each message that adds
 * the key, with all of its fields.
 */
export const createRecords = (icdKey: icd.Key, additions: Addition[]): ReadRecord[] => {
  const reads = new Map<string, icd.ReadMessage[]>();
  for (const { message, recordKeys, index } of additions) {
    const read = readOf(message, index);
    for (const recordKey of recordKeys)
      map.getOrInsert(reads, recordKey, () => []).push(read);
  }
  return [...reads].map(([recordKey, messages]) =>
    kafka.readRecordZ.parse({ icd: icdKey, recordKey, messages }),
  );
};

/** Picks that a message gains, because records of the task read the message already. */
export interface Extension {
  icd: icd.Key;
  message: icd.MessageKey;
  /** The fields that the message gains. */
  fields: icd.JSONField[];
  /** The field keys that each record starts to bind, by record and read index. */
  bindings: { record: string; read: number; fields: icd.FieldKey[] }[];
}

/** A message for a kind of document that records of the task read no message of. */
export interface KindMessage {
  icd: icd.Key;
  message: icd.Message;
  /** The read of the message that each record adds. */
  read: icd.ReadMessage;
  /** The keys of the records that start to read the message. */
  records: string[];
}

/**
 * @returns the picks that the records of the task gain, in the message that the Core
 * gives the documents of each picked shape. A message gains each picked field that it
 * does not hold, once for all of its records. A record starts to read each picked
 * field that it does not read in that message. When no message of a record takes a
 * shape, the records of its ICD start to read a new message for the kind.
 * @throws if a picked shape is not extendable.
 */
export const createExtensions = (
  shapes: json.Shape[],
  picks: Picks,
  records: ReadRecord[],
  messagesOf: MessagesOf,
  label: (recordKey: string) => string,
): { extensions: Extension[]; kinds: KindMessage[] } => {
  const extensions = new Map<icd.MessageKey, Extension>();
  const kinds = new Map<string, KindMessage>();
  for (const shape of shapes) {
    const pointers = picks.get(json.shapeKey(shape));
    if (pointers == null) continue;
    for (const record of records) {
      if (!shape.sources.includes(record.recordKey)) continue;
      const held = messagesOf(record.icd);
      const own = messageReads(record, held);
      if (own.length === 0)
        throw new Error(`Record ${record.key} reads no JSON message`);
      const { message: routed, split } = routeOf(shape, own);
      if (split != null)
        throw new Error(
          `Record ${record.key} has no one message for shape ${json.shapeKey(shape)}`,
        );
      const picked = shape.fields.filter(
        (f) => pointers.has(f.pointer) && routed?.fields.has(f.pointer) !== true,
      );
      if (picked.length === 0) continue;
      if (routed == null) {
        const kind = map.getOrInsert(
          kinds,
          `${record.icd}\n${json.shapeKey(shape)}`,
          () => {
            const made = [...kinds.values()].filter(({ icd }) => icd === record.icd);
            const taken = [...held, ...made.map(({ message }) => message)];
            return createKindMessage(shape, picked, record, taken, label);
          },
        );
        kind.records.push(record.key);
        continue;
      }
      const { read } = routed;
      const { message } = record.messages[read];
      const extension = map.getOrInsert(extensions, message, () => ({
        icd: record.icd,
        message,
        fields: [],
        bindings: [],
      }));
      const holds = [
        ...ICD.jsonFields(held.find(({ key }) => key === message)),
        ...extension.fields,
      ];
      const missing = picked.filter((f) => !holds.some((h) => h.pointer === f.pointer));
      const names = ICD.fieldNames(
        missing.map(({ pointer }) => pointer),
        holds.map(({ name }) => name),
      );
      const added = missing.map((f, i) => ICD.createField(f, names[i]));
      extension.fields.push(...added);
      const existing = holds.filter((h) => picked.some((f) => f.pointer === h.pointer));
      extension.bindings.push({
        record: record.key,
        read,
        fields: [...existing, ...added].map(({ key }) => key),
      });
    }
  }
  return { extensions: [...extensions.values()], kinds: [...kinds.values()] };
};

/** @returns a message of the picked fields of shape and its timestamp, unread. */
const createKindMessage = (
  shape: json.Shape,
  picked: json.InferredField[],
  record: ReadRecord,
  taken: icd.Message[],
  label: (recordKey: string) => string,
): KindMessage => {
  const indexField = shape.index == null ? undefined : shape.fields[shape.index];
  const kept = shape.fields.filter((f) => f === indexField || picked.includes(f));
  const names = ICD.fieldNames(kept.map(({ pointer }) => pointer));
  const fields = kept.map((f, i) => ICD.createField(f, names[i]));
  const message: icd.Message = {
    key: uuid.create(),
    name: uniqueName(
      ICD.kindName(label(record.recordKey), presentOf(shape)?.pointer),
      taken,
    ),
    payload: { format: "json", fields, ...matchOf(shape) },
  };
  const index = indexField == null ? undefined : fields[kept.indexOf(indexField)].key;
  return { icd: record.icd, message, read: readOf(message, index), records: [] };
};
