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

/**
 * Overrides the properties of the base object with the existing properties of the provided
 * object(s)
 * @param base The base object to override
 * @param overrides The object(s) to override the base object with
 */
export const override = <T>(base: T, ...overrides: Array<Partial<T>>): T => {
  if (overrides.length === 0) return base;
  const source = overrides.shift();

  if (narrow.isObject(base) && narrow.isObject(source))
    for (const key in source)
      try {
        if (narrow.isObject(source[key])) {
          if (!(key in base)) Object.assign(base, { [key]: {} });
          override(base[key], source[key]);
        } else Object.assign(base, { [key]: source[key] });
      } catch (e) {
        if (e instanceof TypeError) throw new TypeError(`.${key}: ${e.message}`);
        throw e;
      }

  return override(base, ...overrides);
};

export const overrideValidItems = <A, B>(
  base: A,
  override: B,
  schema: z.ZodType<A>,
): A => {
  const mergeValidFields = (
    baseObj: any,
    overrideObj: any,
    currentSchema: any,
  ): any => {
    if (currentSchema.def?.type === "union")
      return currentSchema.def.options.reduce(
        (acc: any, option: any) => mergeValidFields(acc, overrideObj, option),
        baseObj,
      );
    if (currentSchema.def?.type === "intersection") {
      const out = mergeValidFields(baseObj, overrideObj, currentSchema.def.left);
      const right = mergeValidFields(out, overrideObj, currentSchema.def.right);
      return right;
    }

    // Iterate over each property in the override object
    for (const key in overrideObj) {
      const overrideValue = overrideObj[key];
      let shape = currentSchema?.shape;
      if (shape != null)
        while (shape != null) {
          if (shape[key] != null) {
            const result = shape[key].safeParse(overrideValue);
            // Check if parsing succeeded
            if (result.success) {
              baseObj[key] = result.data;
              break;
            }
          }
          shape = shape.def?.shape;
        }

      if (
        typeof overrideValue === "object" &&
        !Array.isArray(overrideValue) &&
        overrideValue !== null
      )
        if (currentSchema && currentSchema.shape && currentSchema.shape[key]) {
          // If it's a nested object, recurse into it only if a valid schema exists
          baseObj[key] ||= {};
          mergeValidFields(baseObj[key], overrideValue, currentSchema.shape[key]);
        }
    }
    return baseObj;
  };

  return mergeValidFields({ ...base }, override, schema);
};
