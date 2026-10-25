// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { primitive } from "@/primitive";

/** SetOperator is the shape of the Set, Bool, and String operators. */
export interface SetOperator<T> {
  eq?: T;
  in?: T[];
}

/** OrderedOperator is the shape of the Ordered operator. */
export interface OrderedOperator<T> extends SetOperator<T> {
  gt?: T;
  gte?: T;
  lt?: T;
  lte?: T;
}

/** ContainsOperator is the shape of the Contains operator. */
export interface ContainsOperator<T> {
  any?: T[];
}

const unwrap = (value: unknown): unknown =>
  value instanceof primitive.ValueExtension ? value.valueOf() : value;

const equal = (a: unknown, b: unknown): boolean => unwrap(a) === unwrap(b);

/**
 * @returns whether value matches every member of op that is set. An empty `in` matches
 * no value.
 */
export const matchesSet = <T>(op: SetOperator<T>, value: T): boolean =>
  (op.eq === undefined || equal(op.eq, value)) &&
  (op.in == null || op.in.some((v) => equal(v, value)));

/**
 * @returns whether value matches every member of op that is set.
 * @throws if op sets a regex, which only the Core can check.
 */
export const matchesString = (
  op: SetOperator<string> & { regex?: string },
  value: string,
): boolean => {
  if (op.regex != null) throw new Error("only the Core can check a regex");
  return matchesSet(op, value);
};

/** @returns whether value matches every member of op that is set. */
export const matchesOrdered = <T>(op: OrderedOperator<T>, value: T): boolean => {
  if (!matchesSet(op, value)) return false;
  const v = unwrap(value) as number | bigint;
  const bound = (b: T | undefined, ok: (b: number | bigint) => boolean): boolean =>
    b === undefined || ok(unwrap(b) as number | bigint);
  return (
    bound(op.gt, (b) => v > b) &&
    bound(op.gte, (b) => v >= b) &&
    bound(op.lt, (b) => v < b) &&
    bound(op.lte, (b) => v <= b)
  );
};

/**
 * @returns whether values holds an element of op.any. An empty `any` matches no
 * array.
 */
export const matchesContains = <T>(op: ContainsOperator<T>, values: T[]): boolean =>
  op.any == null || op.any.some((v) => values.some((e) => equal(e, v)));
