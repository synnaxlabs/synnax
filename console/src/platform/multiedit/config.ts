// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type color } from "@synnaxlabs/client";
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

/** A color reference stored at a path in one element's config. */
export interface ColorRef {
  key: string;
  path: string;
  value: color.Ref;
}
