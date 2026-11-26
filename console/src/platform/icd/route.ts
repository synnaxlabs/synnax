// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { icd } from "@synnaxlabs/client";

/** The message that the Core gives the documents of a shape. */
export interface Route<M> {
  /** Absent when no message gets the documents, or when split is set. */
  message?: M;
  /**
   * The pointer of the value that tells the documents apart, when the fields of the
   * shape do not tell which message gets them.
   */
  split?: string;
}

/**
 * @returns the message of messages that the Core gives a document with fields. When a
 * field is at the pointer of a value rule, the value picks, and a shape holds none: the
 * message that reads the most of the fields wins, and a tie is a split. Else the
 * message whose present rule is at or above a field wins, then the one with no rule.
 * @param pointersOf - The pointers of the fields that a message reads.
 */
export const route = <M extends { match?: icd.JSONMatch }>(
  fields: { pointer: string }[],
  messages: M[],
  pointersOf: (message: M) => Iterable<string>,
): Route<M> => {
  const holds = (at: string): boolean =>
    fields.some(({ pointer }) => pointer === at || pointer.startsWith(`${at}/`));
  const other =
    messages.find(({ match }) => match?.type === "present" && holds(match.pointer)) ??
    messages.find(({ match }) => match == null);
  const valued = messages.filter(
    ({ match }) => match != null && match.type !== "present",
  );
  const at = valued[0]?.match?.pointer;
  if (at == null || !holds(at)) return { message: other };
  // A value that no rule meets goes to the other message.
  const candidates = other == null ? valued : [...valued, other];
  const own = new Set(fields.map(({ pointer }) => pointer));
  const overlaps = candidates.map(
    (m) => [...pointersOf(m)].filter((p) => own.has(p)).length,
  );
  const most = Math.max(...overlaps);
  const winners = candidates.filter((_, i) => overlaps[i] === most);
  return winners.length === 1 ? { message: winners[0] } : { split: at };
};
