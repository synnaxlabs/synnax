// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { icd } from "@synnaxlabs/client";
import { type json, map, uuid } from "@synnaxlabs/x";

import {
  type Addition,
  type Appends,
  createAdditions,
  createExtensions,
  type Extension,
  type FieldDefaults,
  type Inputs,
  type KindMessage,
  type MessagesOf,
  type Picks,
  readsOf,
  uniqueName,
} from "@/feature/kafka/task/additions";
import type { ReadRecord } from "@/feature/kafka/task/types";
import type { ICD } from "@/platform/icd";

/** The fields picked from the shapes of a sample of one topic. */
export interface TopicPick {
  topic: string;
  shapes: json.Shape[];
  picks: Picks;
  /** The text of each Default input. Absent, none has text. */
  defaults?: FieldDefaults;
  /** The ICD that the new records of the topic read. Absent for no fit or a tie. */
  reading?: ICD.Reuse.Reading;
}

const NO_DEFAULTS: FieldDefaults = new Map();

/** The additions of a topic, and the ICD that its new records read. */
export interface TopicAdditions {
  /** Absent, they read the ICD that the most records read, or a new one. */
  icd?: icd.Key;
  added: Addition[];
}

/** What one add puts in the ICDs and the records of a task. */
export interface Plan {
  extensions: Extension[];
  kinds: KindMessage[];
  /** The additions of each topic, by topic name. */
  additions: Map<string, TopicAdditions>;
  /** The Default inputs of the fields that the add creates, by topic name. */
  inputs: Map<string, Inputs>;
}

/** A Default input whose text spells no default of its field. */
export interface Problem {
  topic: string;
  shape: string;
  pointer: string;
  /** Says why the text spells no default. */
  message: string;
}

/** @returns the first Default input of plan whose text spells no default, if any. */
export const findProblem = ({ inputs }: Plan): Problem | undefined => {
  for (const [topic, shapes] of inputs)
    for (const [shape, pointers] of shapes)
      for (const [pointer, message] of pointers)
        if (message != null) return { topic, shape, pointer, message };
  return undefined;
};

export interface PlanParams {
  topics: TopicPick[];
  /** The topic that gets a record of any key with no fields, or null for none. */
  record: string | null;
  recordsOf: (topic: string) => ReadRecord[];
  messagesOf: MessagesOf;
  /** The ICD that the most records read. Absent, new messages go in a new ICD. */
  home?: icd.Key;
  /** @returns the name of a new message for the records of topic with recordKey. */
  nameOf: (topic: string, recordKey: string) => string;
}

const createMessage = (name: string): icd.Message => ({
  key: uuid.create(),
  name,
  payload: { format: "json", fields: [] },
});

/** @returns message with fields after its own fields. */
export const withFields = (
  message: icd.Message,
  fields: icd.JSONField[],
): icd.Message => {
  const { payload } = message;
  if (payload.format !== "json")
    throw new Error(`Message ${message.key} is not a JSON message`);
  return {
    ...message,
    payload: { ...payload, fields: [...payload.fields, ...fields] },
  };
};

/**
 * @returns what an add of the picks of topics puts in the ICDs and the records of a
 * task. Every topic first extends the messages that its records read. Then every topic
 * adds the records of its free keys, which see each extension. It changes no ICD and no
 * task.
 * @throws if a picked shape is not extendable.
 */
export const planAdd = ({
  topics,
  record,
  recordsOf,
  messagesOf,
  home,
  nameOf,
}: PlanParams): Plan => {
  // The messages of each ICD as the add leaves them so far.
  const drafts = new Map<icd.Key, icd.Message[]>();
  const draftOf: MessagesOf = (icdKey) => drafts.get(icdKey) ?? messagesOf(icdKey);
  // The new messages that additions put in each ICD, or in a new ICD for undefined.
  const making = new Map<icd.Key | undefined, icd.Message[]>();
  const takenIn = (icdKey?: icd.Key): icd.Message[] => [
    ...(icdKey == null ? [] : draftOf(icdKey)),
    ...(making.get(icdKey) ?? []),
  ];
  const plan: Plan = {
    extensions: [],
    kinds: [],
    additions: new Map(),
    inputs: new Map(),
  };
  const inputsOf = (topic: string): Inputs =>
    map.getOrInsert(plan.inputs, topic, () => new Map());
  const held = topics.map(({ topic }) => readsOf(recordsOf(topic), messagesOf));
  const nameIn = (topic: string) => (recordKey: string) => nameOf(topic, recordKey);
  for (const { topic, shapes, picks, defaults = NO_DEFAULTS } of topics) {
    const { extensions, kinds } = createExtensions(
      shapes,
      picks,
      defaults,
      inputsOf(topic),
      recordsOf(topic),
      draftOf,
      nameIn(topic),
    );
    for (const { icd: icdKey, message, fields } of extensions)
      drafts.set(
        icdKey,
        draftOf(icdKey).map((m) => (m.key === message ? withFields(m, fields) : m)),
      );
    for (const { icd: icdKey, message } of kinds)
      drafts.set(icdKey, [...draftOf(icdKey), message]);
    plan.extensions.push(...extensions);
    plan.kinds.push(...kinds);
  }
  const appends: Appends = new Map();
  for (const [i, pick] of topics.entries()) {
    const { topic, shapes, picks, defaults = NO_DEFAULTS, reading } = pick;
    const target = reading?.icd ?? home;
    const added = createAdditions(
      shapes,
      picks,
      defaults,
      inputsOf(topic),
      held[i],
      takenIn(target),
      nameIn(topic),
      reading,
      appends,
    );
    if (added.length === 0) continue;
    plan.additions.set(topic, { icd: reading?.icd, added });
    map
      .getOrInsert(making, target, () => [])
      .push(
        ...added.filter(({ reused }) => reused !== true).map(({ message }) => message),
      );
  }
  if (record != null) {
    const held = plan.additions.get(record);
    const name = uniqueName(nameOf(record, ""), takenIn(held?.icd ?? home));
    const empty: Addition = {
      message: createMessage(name),
      recordKeys: [""],
      fields: [],
    };
    plan.additions.set(record, {
      icd: held?.icd,
      added: [...(held?.added ?? []), empty],
    });
  }
  return plan;
};
