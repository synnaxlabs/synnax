// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { z } from "zod";

/**
 * @returns a check for an array schema that adds an issue at [index, key] for each item
 * whose key value an earlier item already holds. Items for which skip returns true are
 * not checked and do not claim their value.
 */
export const uniqueBy =
  <T, K extends keyof T & string>(
    key: K,
    message: (value: T[K]) => string,
    skip?: (item: T) => boolean,
  ) =>
  (ctx: z.core.ParsePayload<T[]>): void => {
    const seen = new Set<T[K]>();
    ctx.value.forEach((item, i) => {
      if (skip?.(item) === true) return;
      const value = item[key];
      if (seen.has(value))
        ctx.issues.push({
          continue: true,
          code: "custom",
          input: ctx.value,
          message: message(value),
          path: [i, key],
        });
      else seen.add(value);
    });
  };
