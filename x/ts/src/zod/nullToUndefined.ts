// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type z, type ZodTypeDef } from "zod";

export const nullToUndefined = <
  Output = any,
  Def extends ZodTypeDef = ZodTypeDef,
  Input = Output,
>(
  schema: z.ZodSchema<Output, Def, Input>,
): z.ZodEffects<
  z.ZodNullable<z.ZodOptional<z.ZodType<Output, Def, Input>>>,
  (Output & {}) | undefined,
  Input | null | undefined
> =>
  schema
    .optional()
    .nullable()
    .transform((s) => (s === null ? undefined : s));
