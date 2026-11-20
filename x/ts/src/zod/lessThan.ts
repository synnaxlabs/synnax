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
 * @returns a check for an object schema that adds an issue at key when the value at key
 * is not less than the value at other.
 */
export const lessThan =
  <K extends string, O extends string>(key: K, other: O, message: string) =>
  <T extends Record<K | O, number>>(ctx: z.core.ParsePayload<T>): void => {
    if (ctx.value[key] < ctx.value[other]) return;
    ctx.issues.push({
      continue: true,
      code: "custom",
      input: ctx.value,
      message,
      path: [key],
    });
  };
