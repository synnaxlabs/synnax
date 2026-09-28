// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color, deep, type record } from "@synnaxlabs/x";
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

/**
 * @returns a copy of the config with each path set to its value. An undefined value
 * removes the path, so the field reads as absent.
 */
export const patch = <C>(config: C, entries: Array<[string, unknown]>): C => {
  const next = deep.copy(config);
  entries.forEach(([path, value]) => {
    if (value === undefined) deep.remove(next, path);
    else deep.set(next, path, value);
  });
  return next;
};

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

/** Groups color references by the color they hold, in first-seen order. */
export const groupByColor = (refs: ColorRef[]): Map<color.Hex, ColorRef[]> => {
  const groups = new Map<color.Hex, ColorRef[]>();
  refs.forEach((ref) => {
    const hex = color.hex(ref.value);
    const group = groups.get(hex);
    if (group == null) groups.set(hex, [ref]);
    else group.push(ref);
  });
  return groups;
};

/** Groups color references by the key of the element that holds them. */
export const groupByKey = (refs: ColorRef[]): Map<string, ColorRef[]> => {
  const groups = new Map<string, ColorRef[]>();
  refs.forEach((ref) => {
    const group = groups.get(ref.key);
    if (group == null) groups.set(ref.key, [ref]);
    else group.push(ref);
  });
  return groups;
};
