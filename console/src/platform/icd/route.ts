// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { icd } from "@synnaxlabs/client";
import type { json } from "@synnaxlabs/x";

/** A field of a shape, with the values that its documents hold. */
type Field = Pick<json.InferredField, "pointer" | "sample" | "enumLabels">;

/** The message that the Core gives the documents of a shape. */
export interface Route<M> {
  /** Absent when no message gets the documents, or when split or matches is set. */
  message?: M;
  /**
   * The pointer of the value that tells the documents apart, when their values there go
   * to more than one message or the form cannot read them.
   */
  split?: string;
  /**
   * The messages whose rules the documents meet, when they meet more than one present
   * rule, or a present rule and a value rule. The Core refuses those documents.
   */
  matches?: M[];
}

/**
 * @returns the values that the documents of a shape hold at pointer: the enum labels of
 * the field there, or the value of its sample. [undefined] when no field is at pointer,
 * and undefined when the sample is cut short.
 */
const valuesAt = (fields: Field[], pointer: string): unknown[] | undefined => {
  const field = fields.find((f) => f.pointer === pointer);
  if (field == null) return [undefined];
  if (field.enumLabels.length > 0) return field.enumLabels;
  try {
    return [JSON.parse(field.sample)];
  } catch {
    return undefined;
  }
};

/** @returns whether value meets the value rule match, as the Core compares them. */
const meets = (match: icd.JSONMatch, value: unknown): boolean => {
  switch (match.type) {
    case "string":
    case "boolean":
      return value === match.value;
    case "integer":
    case "unsigned":
    case "float":
      return typeof value === "number" && value === Number(match.value);
    case "present":
      return false;
  }
};

/**
 * @returns the message of messages that the Core gives the documents of a shape with
 * fields: the message whose rule they meet, else the message with no rule. Matches
 * holds the messages of documents that meet two rules, and split is set when the values
 * of the shape go to more than one message. When the sample at the pointer of a value
 * rule is cut short, the message that reads the most of the fields wins, and a tie is a
 * split.
 * @param pointersOf - The pointers of the fields that a message reads.
 */
export const route = <M extends { match?: icd.JSONMatch }>(
  fields: Field[],
  messages: M[],
  pointersOf: (message: M) => Iterable<string>,
): Route<M> => {
  const holds = (at: string): boolean =>
    fields.some(({ pointer }) => pointer === at || pointer.startsWith(`${at}/`));
  const met = messages.filter(
    ({ match }) => match?.type === "present" && holds(match.pointer),
  );
  if (met.length > 1) return { matches: met };
  const fallback = met[0] ?? messages.find(({ match }) => match == null);
  const valued = messages.filter(
    ({ match }) => match != null && match.type !== "present",
  );
  const at = valued[0]?.match?.pointer;
  if (at == null) return { message: fallback };
  const values = valuesAt(fields, at);
  if (values == null) {
    if (met.length === 1) return { message: fallback };
    const candidates = fallback == null ? valued : [...valued, fallback];
    const own = new Set(fields.map(({ pointer }) => pointer));
    const overlaps = candidates.map(
      (m) => [...pointersOf(m)].filter((p) => own.has(p)).length,
    );
    const most = Math.max(...overlaps);
    const winners = candidates.filter((_, i) => overlaps[i] === most);
    return winners.length === 1 ? { message: winners[0] } : { split: at };
  }
  const ruled = values.map((v) =>
    valued.find(({ match }) => match != null && meets(match, v)),
  );
  if (met.length === 1 && ruled.some((m) => m != null))
    return { matches: messages.filter((m) => m === met[0] || ruled.includes(m)) };
  const targets = new Set(ruled.map((m) => m ?? fallback));
  return targets.size === 1 ? { message: [...targets][0] } : { split: at };
};
