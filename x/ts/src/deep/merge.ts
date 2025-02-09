// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type z } from "zod";

import { type Partial } from "@/deep/partial";
import { isObject } from "@/identity";

/**
 * Overrides the properties of the base object with the existing properties of the provided
 * object(s)
 * @param base The base object to override
 * @param overrides The object(s) to override the base object with
 */
export const override = <T>(base: T, ...overrides: Array<Partial<T>>): T => {
  if (overrides.length === 0) return base;
  const source = overrides.shift();

  if (isObject(base) && isObject(source))
    for (const key in source)
      try {
        if (isObject(source[key])) {
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
  schema: z.ZodType<A, any, any>,
): A => {
  const mergeValidFields = (
    baseObj: any,
    overrideObj: any,
    currentSchema: any,
  ): any => {
    // Iterate over each property in the override object
    for (const key in overrideObj) {
      const overrideValue = overrideObj[key];
      const shape = getSchemaShape(currentSchema);
      if (shape?.[key]) {
        const result = shape[key].safeParse(overrideValue);
        // Check if parsing succeeded
        if (result.success) baseObj[key] = result.data;
      } else if (
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

const getSchemaShape = (schema: z.ZodType<any, any, any>) => {
  if (schema._def?.typeName === "ZodEffects") return getSchemaShape(schema._def.schema);
  /// @ts-expect-error - shape is not defined on zod effects
  return schema?.shape;
};
