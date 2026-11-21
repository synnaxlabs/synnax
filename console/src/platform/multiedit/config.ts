// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color } from "@synnaxlabs/client";
import { color as xcolor, type record } from "@synnaxlabs/x";
import { z } from "zod";

import { fixedRef } from "@/platform/multiedit/fixed";

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

/** A color reference stored at a path in one element's config. */
export interface ColorRef {
  key: string;
  path: string;
  value: color.Ref;
  /** True if the path stores the color as a value, so it cannot hold a named color. */
  valueOnly?: boolean;
}

/**
 * @returns the color references stored at the given top-level fields of the config.
 * Each field in valueFields stores a color value, which the result holds as a fixed
 * reference.
 * @throws if a field holds a value of the wrong kind.
 */
export const colorRefs = (
  key: string,
  config: record.Unknown,
  fields: readonly string[],
  valueFields: Readonly<Record<string, true>> = {},
): ColorRef[] =>
  fields.flatMap((path): ColorRef[] => {
    const value = config[path];
    if (value == null) return [];
    if (valueFields[path] === true)
      return [
        { key, path, value: fixedRef(xcolor.colorZ.parse(value)), valueOnly: true },
      ];
    return [{ key, path, value: color.refZ.parse(value) }];
  });
