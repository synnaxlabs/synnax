// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type z } from "zod";

export const nullToUndefined = <Z extends z.ZodType>(
  schema: Z,
): z.ZodOptional<
  z.ZodPipe<
    z.ZodNullable<Z>,
    z.ZodTransform<Awaited<z.infer<Z>> | undefined, z.infer<Z> | null>
  >
> =>
  schema
    .nullable()
    .transform((s) => (s === null ? undefined : s))
    .optional() as z.ZodOptional<
    z.ZodPipe<
      z.ZodNullable<Z>,
      z.ZodTransform<Awaited<z.infer<Z>> | undefined, z.infer<Z> | null>
    >
  >;
