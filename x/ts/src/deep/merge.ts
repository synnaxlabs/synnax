// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type z } from "zod";

import { type Partial } from "@/deep/partial";
import { narrow } from "@/narrow";
import { type record } from "@/record";

const merge = (base: unknown, source: unknown): unknown => {
  if (!narrow.isObject(base) || !narrow.isObject(source)) return base;
  const out: record.Unknown = { ...base };
  for (const key in source) {
    const value = source[key];
    out[key] = narrow.isObject(value)
      ? merge(key in base ? base[key] : {}, value)
      : value;
  }
  return out;
};

/**
 * Writes the fields of each override over the base, in order. Nested objects merge
 * field by field, and a nested object over a value that is not an object is dropped.
 *
 * @returns the merged value. The base and its nested objects stay unchanged.
 */
export const override = <T>(base: T, ...overrides: Array<Partial<T>>): T =>
  overrides.reduce<T>((merged, source) => merge(merged, source) as T, base);

const mergeValidFields = (base: any, override: any, schema: any): any => {
  if (schema.def?.type === "union")
    return schema.def.options.reduce(
      (merged: any, option: any) => mergeValidFields(merged, override, option),
      base,
    );
  if (schema.def?.type === "intersection") {
    const left = mergeValidFields(base, override, schema.def.left);
    return mergeValidFields(left, override, schema.def.right);
  }
  const out = { ...base };
  for (const key in override) {
    const value = override[key];
    let shape = schema?.shape;
    while (shape != null) {
      if (shape[key] != null) {
        const result = shape[key].safeParse(value);
        if (result.success) {
          out[key] = result.data;
          break;
        }
      }
      shape = shape.def?.shape;
    }
    if (narrow.isObject(value) && schema?.shape?.[key] != null)
      out[key] = mergeValidFields(out[key] || {}, value, schema.shape[key]);
  }
  return out;
};

/**
 * Writes each field of the override that the schema accepts over the base. Nested
 * objects merge field by field.
 *
 * @returns the merged value. The base and its nested objects stay unchanged.
 */
export const overrideValidItems = <A>(
  base: A,
  override: unknown,
  schema: z.ZodType<A>,
): A => mergeValidFields(base, override, schema);
