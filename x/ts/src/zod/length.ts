// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

const codePoints = (value: string): number => [...value].length;

/**
 * @returns a check for `.check()` that rejects a string with fewer than `min` Unicode
 * code points. Zod's own `.min()` counts UTF-16 code units.
 */
export const minLength = (min: number): z.core.$ZodCheck<string> =>
  z.refine<string>(
    (value) => codePoints(value) >= min,
    `must be at least ${min} characters long`,
  );

/**
 * @returns a check for `.check()` that rejects a string with more than `max` Unicode
 * code points. Zod's own `.max()` counts UTF-16 code units.
 */
export const maxLength = (max: number): z.core.$ZodCheck<string> =>
  z.refine<string>(
    (value) => codePoints(value) <= max,
    `must be at most ${max} characters long`,
  );
