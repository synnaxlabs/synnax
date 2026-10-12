// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color, type record } from "@synnaxlabs/x";
import { z } from "zod";

/**
 * Maps each variant to the top-level fields its schema declares, so an element "has" a
 * field even while the field is absent from its stored config.
 *
 * @throws if a schema is not a Zod object.
 */
export const fieldsByVariant = (
  schemas: Record<string, z.ZodType>,
): Map<string, Set<string>> =>
  new Map(
    Object.entries(schemas).map(([variant, schema]) => {
      if (!(schema instanceof z.ZodObject))
        throw new Error(`[multiedit] - schema for ${variant} is not an object`);
      return [variant, new Set(Object.keys(schema.shape))];
    }),
  );

/** A color stored at a path in one element's config. */
export interface ColorRef {
  key: string;
  path: string;
  value: color.Color;
}

/** @returns the colors stored at the given top-level fields of the config. */
export const colorRefs = (
  key: string,
  config: record.Unknown,
  fields: readonly string[],
): ColorRef[] =>
  fields.flatMap((path) => {
    const value = config[path];
    if (value == null) return [];
    return [{ key, path, value: color.construct(value as color.Crude) }];
  });
