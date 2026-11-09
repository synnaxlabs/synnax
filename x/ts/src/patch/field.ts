// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

import { narrow } from "@/narrow";
import { mark } from "@/patch/kind";
import { listMoveZ } from "@/patch/types.gen";

/**
 * Builds the schema of a change to a scalar field of an update: a value sets the field,
 * and null clears it. A generated update makes the field optional, and an absent field
 * does not change.
 */
export const changeZ = <T extends z.ZodType>(value: T) =>
  mark(value.nullable(), { kind: "change", value });
export type Change<T extends z.ZodType> = z.infer<ReturnType<typeof changeZ<T>>>;

const holdsSet = (value: unknown): boolean =>
  narrow.isObject(value) && Object.hasOwn(value, "set");

/** The schemas of the field that {@link structZ} changes. */
export interface StructSchemas<U extends z.ZodType, T extends z.ZodType> {
  /** The schema of a partial of the field. */
  u: U;
  /** The schema of the whole field. */
  t: T;
}

/**
 * Builds the schema of a change to a struct field of an update: a partial merges into
 * the field, and an object with only a `set` member replaces the field. An optional
 * struct field takes `changeZ(structZ({ u, t }))`, whose null clears the field.
 */
export const structZ = <U extends z.ZodType, T extends z.ZodType>({
  u,
  t,
}: StructSchemas<U, T>) =>
  mark(
    z.union([
      z.strictObject({ set: t }),
      z.custom<unknown>((v) => !holdsSet(v)).pipe(u),
    ]),
    { kind: "struct", update: u, full: t },
  );
export type Struct<U extends z.ZodType, T extends z.ZodType> = z.infer<
  ReturnType<typeof structZ<U, T>>
>;

/** The schemas of the entries of a list that {@link listZ} changes. */
export interface ListSchemas<
  E extends z.ZodType,
  K extends z.ZodType,
  U extends z.ZodType,
> {
  /** The schema of an entry. */
  e: E;
  /** The schema of the key of an entry. */
  k: K;
  /** The schema of a partial entry, which holds its key. */
  u: U;
}

/**
 * Builds the schema of the operators that change a list of entries with a key. The
 * operators apply in the order remove, update, insert, then move, and set replaces the
 * list alone.
 */
export const listZ = <E extends z.ZodType, K extends z.ZodType, U extends z.ZodType>({
  e,
  k,
  u,
}: ListSchemas<E, K, U>) =>
  mark(
    z.strictObject({
      set: e.array().optional(),
      insert: e.array().optional(),
      remove: k.array().optional(),
      update: u.array().optional(),
      move: listMoveZ(k).array().optional(),
    }),
    { kind: "list", entry: e, update: u },
  );
export type List<
  E extends z.ZodType,
  K extends z.ZodType,
  U extends z.ZodType,
> = z.infer<ReturnType<typeof listZ<E, K, U>>>;

/**
 * Builds the schema of the operators that change a list of values that are their own
 * keys. The operators apply in the order remove, insert, then move, and set replaces
 * the list alone.
 */
export const valueListZ = <E extends z.ZodType>(e: E) =>
  mark(
    z.strictObject({
      set: e.array().optional(),
      insert: e.array().optional(),
      remove: e.array().optional(),
      move: listMoveZ(e).array().optional(),
    }),
    { kind: "valueList", entry: e },
  );
export type ValueList<E extends z.ZodType> = z.infer<ReturnType<typeof valueListZ<E>>>;
