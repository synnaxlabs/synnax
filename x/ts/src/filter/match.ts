// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type z } from "zod";

import { type Contains, type Ordered, type Set, type String } from "@/filter/types.gen";
import { primitive } from "@/primitive";

type Ordinal = number | bigint;

type Orderable = Ordinal | primitive.ValueExtension<Ordinal>;

const unwrap = (value: unknown): unknown =>
  value instanceof primitive.ValueExtension ? value.valueOf() : value;

const ordinal = (value: Orderable): Ordinal =>
  value instanceof primitive.ValueExtension ? value.valueOf() : value;

const equal = (a: unknown, b: unknown): boolean => unwrap(a) === unwrap(b);

/**
 * @returns whether value matches every member of op that is set. An empty `in` matches
 * no value.
 */
export const matchesSet = <T>(op: Set<z.ZodType<T>>, value: T): boolean =>
  (op.eq === undefined || equal(op.eq, value)) &&
  (op.in == null || op.in.some((v) => equal(v, value)));

const escape = (s: string): string => s.replace(/[\\^$.*+?()[\]{}|]/g, "\\$&");

/**
 * @returns whether value matches every member of op that is set. With caseInsensitive,
 * it compares by Unicode simple case folding, the rule the Core uses.
 * @throws {Error} if op sets a regex, since regex dialects differ between languages.
 */
export const matchesString = (op: String, value: string): boolean => {
  if (op.regex != null) throw new Error("cannot match a regex locally");
  if (op.caseInsensitive !== true)
    return (
      (op.eq == null || op.eq === value) &&
      (op.in == null || op.in.includes(value)) &&
      (op.contains == null || value.includes(op.contains)) &&
      (op.startsWith == null || value.startsWith(op.startsWith))
    );
  const test = (pattern: string): boolean => new RegExp(pattern, "iu").test(value);
  return (
    (op.eq == null || test(`^${escape(op.eq)}$`)) &&
    (op.in == null || op.in.some((e) => test(`^${escape(e)}$`))) &&
    (op.contains == null || test(escape(op.contains))) &&
    (op.startsWith == null || test(`^${escape(op.startsWith)}`))
  );
};

/** @returns whether value matches every member of op that is set. */
export const matchesOrdered = <T extends Orderable>(
  op: Ordered<z.ZodType<T>>,
  value: T,
): boolean => {
  if (!matchesSet(op, value)) return false;
  const v = ordinal(value);
  const bound = (b: T | undefined, ok: (b: Ordinal) => boolean): boolean =>
    b === undefined || ok(ordinal(b));
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
export const matchesContains = <T>(op: Contains<z.ZodType<T>>, values: T[]): boolean =>
  op.any == null || op.any.some((v) => values.some((e) => equal(e, v)));
