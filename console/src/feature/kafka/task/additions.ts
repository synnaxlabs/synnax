// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { icd, kafka } from "@synnaxlabs/client";
import { json, map, unique, uuid } from "@synnaxlabs/x";

import type { ReadRecord } from "@/feature/kafka/task/types";
import { ICD } from "@/platform/icd";

/** The pointers of the fields picked from each shape of a sample, by shape key. */
export type Picks = Map<string, Set<string>>;

/** The text of the Default input of each picked field, by shape key, then pointer. */
export type FieldDefaults = Map<string, Map<string, string>>;

/** @returns the text that defaults give the field at pointer of shape, or "". */
const textOf = (defaults: FieldDefaults, shape: json.Shape, pointer: string): string =>
  defaults.get(json.shapeKey(shape))?.get(pointer) ?? "";

/**
 * @returns a read field of f named name, with the default that defaults give f in
 * shape.
 */
const createField = (
  f: json.InferredField,
  name: string,
  shape: json.Shape,
  defaults: FieldDefaults,
): icd.JSONField =>
  ICD.parseDefault(ICD.createReadField(f, name), textOf(defaults, shape, f.pointer));

/** A message that new records read, and those records. */
export interface Addition {
  message: icd.Message;
  /** The key of the shape whose new records read the message. Absent for none. */
  shape?: string;
  /** Whether an ICD holds the message already. Absent, Add data puts it in one. */
  reused?: boolean;
  /** The fields that a held message gains. */
  appended?: icd.JSONField[];
  /** The Kafka keys of the new records that read the message. */
  recordKeys: string[];
  /** The key of the field that indexes the message. Absent when Kafka stamps it. */
  index?: icd.FieldKey;
  /** The keys of the fields of the message that the records read. */
  fields: icd.FieldKey[];
}

/** @returns the key of a record of topic in the read properties of a device. */
export const sourceKey = (topic: string, recordKey: string): string =>
  `${topic}#${recordKey}`;

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

/** @returns the shapes with a key that has no record: the shapes of new records. */
export const freeShapes = (shapes: json.Shape[], reads: Reads): json.Shape[] =>
  shapes.filter((shape) => freeKeys(shape, reads).length > 0);

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

const readOf = (
  message: icd.Message,
  index: icd.FieldKey | undefined,
  fields: icd.FieldKey[],
): icd.ReadMessage =>
  icd.readMessageZ.parse({
    message: message.key,
    index,
    bindings: fields.map((field) => ({ field })),
  });

/** The fields that messages gain in one add and the forms do not hold yet, by key. */
export type Appends = Map<icd.MessageKey, icd.JSONField[]>;

/**
 * @returns whether the new records of shape can read a message of fields. They cannot
 * when a field holds a field of the shape but cannot read its values, or holds the
 * timestamp of the shape in a field of another type.
 */
export const isReusable = (shape: json.Shape, fields: icd.JSONField[]): boolean => {
  const stamp = shape.index == null ? undefined : shape.fields[shape.index];
  return shape.fields.every((f) =>
    fields.every((h) => {
      if (!ICD.Reuse.covers(h, f.pointer)) return true;
      if (f === stamp && h.pointer === f.pointer && h.type !== "timestamp")
        return false;
      return ICD.Reuse.reads(h, f);
    }),
  );
};

/**
 * @returns an addition whose records read the message of fit for the free keys of
 * shape. The records read the picked and carried fields, indexed by the timestamp of
 * the shape. The message gains each of those fields that it lacks.
 * @param appends - What messages gain in this add. The fields that the message gains
 * join it.
 */
const reuse = (
  shape: json.Shape,
  { message }: ICD.Reuse.Fit,
  pointers: Set<string>,
  carried: Map<string, icd.JSONField>,
  recordKeys: string[],
  appends: Appends,
  defaults: FieldDefaults,
): Addition => {
  const gained = map.getOrInsert(appends, message.key, () => []);
  const stamp = shape.index == null ? undefined : shape.fields[shape.index];
  const holderOf = (f: json.InferredField) =>
    [...ICD.jsonFields(message), ...gained].find((h) =>
      f === stamp
        ? h.pointer === f.pointer && h.type === "timestamp"
        : ICD.Reuse.covers(h, f.pointer),
    );
  const read = shape.fields.filter(
    (f) => f === stamp || pointers.has(f.pointer) || carried.has(f.pointer),
  );
  const lacking = read.filter((f) => holderOf(f) == null);
  const names = ICD.fieldNames(
    lacking.map(({ pointer }) => pointer),
    [...ICD.jsonFields(message), ...gained].map(({ name }) => name),
  );
  const appended = lacking.map((f, i) => {
    const field = carried.get(f.pointer);
    return field == null
      ? createField(f, names[i], shape, defaults)
      : { ...field, key: uuid.create() };
  });
  gained.push(...appended);
  const index = stamp == null ? undefined : holderOf(stamp)?.key;
  return {
    message,
    shape: json.shapeKey(shape),
    reused: true,
    appended,
    recordKeys,
    ...(index == null ? {} : { index }),
    fields: unique.unique(read.flatMap((f) => holderOf(f)?.key ?? [])),
  };
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
 * @returns the messages that new records read for each shape with picks. Each record
 * reads a key of the shape with no record of its own. A shape reads the message that
 * reading fits to it. Else it gets a new message of the picked fields, the fields that
 * its keys carry, and its timestamp. A new message takes the name that nameOf gives its
 * first key. A shape that is one of several kinds adds the name of its present field,
 * and its message matches the documents that hold that field.
 * @param held - The messages of the ICD that the new records read, as the forms hold
 * them now. A new message takes none of their names.
 * @param reading - The ICD that the new records read, with a fit for each free shape
 * that reuses a message of it. A shape gets a new message instead when its message
 * gains a field in this add that cannot read the shape.
 * @param appends - What messages gain in this add. A held message gains the fields
 * that its new records read and it lacks.
 */
export const createAdditions = (
  shapes: json.Shape[],
  picks: Picks,
  defaults: FieldDefaults,
  reads: Reads,
  held: icd.Message[],
  nameOf: (recordKey: string) => string,
  reading: ICD.Reuse.Reading | undefined,
  appends: Appends,
): Addition[] => {
  const taken = [...held];
  return freeShapes(shapes, reads).flatMap((shape, i): Addition[] => {
    const pointers = picks.get(json.shapeKey(shape));
    const recordKeys = freeKeys(shape, reads);
    if (pointers == null || pointers.size === 0) return [];
    const carried = carriedFields(shape, reads);
    const fit = reading?.fits[i];
    if (fit != null) {
      const message = held.find(({ key }) => key === fit.message.key);
      if (message == null)
        throw new Error(`No held message has the key ${fit.message.key}`);
      const gained = appends.get(message.key) ?? [];
      if (isReusable(shape, [...ICD.jsonFields(message), ...gained]))
        return [
          reuse(
            shape,
            { ...fit, message },
            pointers,
            carried,
            recordKeys,
            appends,
            defaults,
          ),
        ];
    }
    const indexField = shape.index == null ? undefined : shape.fields[shape.index];
    const picked = shape.fields.filter(
      (f) => f === indexField || (pointers.has(f.pointer) && !carried.has(f.pointer)),
    );
    const names = ICD.fieldNames(
      picked.map(({ pointer }) => pointer),
      [...carried.values()].map(({ name }) => name),
    );
    const named = new Map(
      picked.map((f, i) => [f, createField(f, names[i], shape, defaults)]),
    );
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
        ICD.kindName(nameOf(recordKeys[0]), presentOf(shape)?.pointer),
        taken,
      ),
      payload: { format: "json", fields, ...matchOf(shape) },
    };
    taken.push(message);
    return [
      {
        message,
        shape: json.shapeKey(shape),
        recordKeys,
        ...(index == null ? {} : { index }),
        fields: fields.map(({ key }) => key),
      },
    ];
  });
};

/**
 * @returns a record for each Kafka key of the additions, reading each message that adds
 * the key.
 */
export const createRecords = (icdKey: icd.Key, additions: Addition[]): ReadRecord[] => {
  const reads = new Map<string, icd.ReadMessage[]>();
  for (const { message, recordKeys, index, fields } of additions) {
    const read = readOf(message, index, fields);
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

/** An extension while it collects the picks of each shape. */
interface Draft extends Omit<Extension, "fields"> {
  /**
   * Each field that the message gains, with the inferred fields that pick it and the
   * first text that their Default inputs give it.
   */
  added: { field: icd.JSONField; picks: json.InferredField[]; text: string }[];
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
  defaults: FieldDefaults,
  records: ReadRecord[],
  messagesOf: MessagesOf,
  nameOf: (recordKey: string) => string,
): { extensions: Extension[]; kinds: KindMessage[] } => {
  const drafts = new Map<icd.MessageKey, Draft>();
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
            return createKindMessage(shape, picked, record, taken, nameOf, defaults);
          },
        );
        kind.records.push(record.key);
        continue;
      }
      const { read } = routed;
      const { message } = record.messages[read];
      const draft = map.getOrInsert(drafts, message, () => ({
        icd: record.icd,
        message,
        added: [],
        bindings: [],
      }));
      for (const a of draft.added) {
        const more = picked.filter(({ pointer }) => pointer === a.field.pointer);
        a.picks.push(...more);
        if (a.text === "" && more.length > 0)
          a.text = textOf(defaults, shape, a.field.pointer);
      }
      const holds = [
        ...ICD.jsonFields(held.find(({ key }) => key === message)),
        ...draft.added.map(({ field }) => field),
      ];
      const missing = picked.filter((f) => !holds.some((h) => h.pointer === f.pointer));
      const names = ICD.fieldNames(
        missing.map(({ pointer }) => pointer),
        holds.map(({ name }) => name),
      );
      const added = missing.map((f, i) => createField(f, names[i], shape, defaults));
      draft.added.push(
        ...added.map((field, i) => ({
          field,
          picks: [missing[i]],
          text: textOf(defaults, shape, missing[i].pointer),
        })),
      );
      const existing = holds.filter((h) => picked.some((f) => f.pointer === h.pointer));
      draft.bindings.push({
        record: record.key,
        read,
        fields: [...existing, ...added].map(({ key }) => key),
      });
    }
  }
  const extensions = [...drafts.values()].map(({ added, ...rest }) => ({
    ...rest,
    fields: added.map(({ field, picks, text }) => {
      const [merged] = ICD.mergeFields(picks);
      return ICD.parseDefault(
        { ...ICD.createReadField(merged, field.name), key: field.key },
        text,
      );
    }),
  }));
  return { extensions, kinds: [...kinds.values()] };
};

/** @returns a message of the picked fields of shape and its timestamp, unread. */
const createKindMessage = (
  shape: json.Shape,
  picked: json.InferredField[],
  record: ReadRecord,
  taken: icd.Message[],
  nameOf: (recordKey: string) => string,
  defaults: FieldDefaults,
): KindMessage => {
  const indexField = shape.index == null ? undefined : shape.fields[shape.index];
  const kept = shape.fields.filter((f) => f === indexField || picked.includes(f));
  const names = ICD.fieldNames(kept.map(({ pointer }) => pointer));
  const fields = kept.map((f, i) => createField(f, names[i], shape, defaults));
  const message: icd.Message = {
    key: uuid.create(),
    name: uniqueName(
      ICD.kindName(nameOf(record.recordKey), presentOf(shape)?.pointer),
      taken,
    ),
    payload: { format: "json", fields, ...matchOf(shape) },
  };
  const index = indexField == null ? undefined : fields[kept.indexOf(indexField)].key;
  const read = readOf(
    message,
    index,
    fields.map(({ key }) => key),
  );
  return { icd: record.icd, message, read, records: [] };
};
