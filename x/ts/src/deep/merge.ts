// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Partial } from "@/deep/partial";
import { isObject } from "@/identity";
import { z } from "zod";

export const merge = <T>(base: T, ...objects: Array<Partial<T>>): T => {
  if (objects.length === 0) return base;
  const source = objects.shift();

  if (isObject(base) && isObject(source)) {
    for (const key in source) {
      try {
        if (isObject(source[key])) {
          if (!(key in base)) Object.assign(base, { [key]: {} });
          merge(base[key], source[key]);
        } else {
          Object.assign(base, { [key]: source[key] });
        }
      } catch (e) {
        if (e instanceof TypeError) {
          throw new TypeError(`.${key}: ${e.message}`);
        }
        throw e;
      }
    }
  }

  return merge(base, ...objects);
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
      // Check if the current key exists in the schema and if schema for this key is defined
      if (currentSchema?.shape[key]) {
        const result = currentSchema.shape[key].safeParse(overrideValue);
        // Check if parsing succeeded
        if (result.success) baseObj[key] = result.data;
      } else if (
        typeof overrideValue === "object" &&
        !Array.isArray(overrideValue) &&
        overrideValue !== null
      ) {
        // If it's a nested object, recurse into it only if a valid schema exists
        if (currentSchema && currentSchema.shape && currentSchema.shape[key]) {
          if (!baseObj[key]) baseObj[key] = {};
          mergeValidFields(baseObj[key], overrideValue, currentSchema.shape[key]);
        }
      }
    }
    return baseObj;
  };

  return mergeValidFields({ ...base }, override, schema);
};
