// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

import { boolZ, containsZ, orderedZ, setZ, stringZ } from "@/filter/types.gen";

const shorthandZ = <O extends z.ZodType, T extends z.ZodType>(op: O, t: T) =>
  z.union([
    op,
    t.array().transform((values) => ({ in: values })),
    t.transform((value) => ({ eq: value })),
  ]);

/** Reads a set line written as a value, a list of values, or a set operator. */
export const setInputZ = <T extends z.ZodType>(t: T) => shorthandZ(setZ(t), t);

/** Reads an ordered line written as a value, a list of values, or an ordered operator. */
export const orderedInputZ = <T extends z.ZodType>(t: T) => shorthandZ(orderedZ(t), t);

/** Reads a string line written as a value, a list of values, or a string operator. */
export const stringInputZ = shorthandZ(stringZ, z.string());

/** Reads a bool line written as a value or a bool operator. */
export const boolInputZ = z.union([
  boolZ,
  z.boolean().transform((value) => ({ eq: value })),
]);

/**
 * Reads a line on a list field written as a value, a list of values, or a contains
 * operator. A value or a list matches the records that hold any of them.
 */
export const containsInputZ = <T extends z.ZodType>(t: T) =>
  z.union([
    containsZ(t),
    t.array().transform((values) => ({ any: values })),
    t.transform((value) => ({ any: [value] })),
  ]);

/** Reads a typed line that takes a list, written as a value or a list of values. */
export const listInputZ = <T extends z.ZodType>(t: T) =>
  z.union([t.array(), t.transform((value) => [value])]);

/** Input is the partial record that a shape of line schemas reads. */
export type Input<S extends Record<string, z.ZodType>> = {
  [K in keyof S]?: z.input<S[K]>;
};

/**
 * @returns input with the members named by fields moved into where. where is undefined
 * when input sets none of them, so the two requests hash alike.
 */
export const split = <T extends object, F extends keyof T>(
  input: T,
  fields: readonly F[],
): Omit<T, F> & { where?: Pick<T, F> } => {
  const rest: Partial<T> = { ...input };
  const where: Partial<T> = {};
  for (const f of fields) {
    if (input[f] === undefined) continue;
    where[f] = input[f];
    delete rest[f];
  }
  return {
    ...(rest as Omit<T, F>),
    where: Object.keys(where).length > 0 ? (where as Pick<T, F>) : undefined,
  };
};
