// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

export const VARIANT = "groupBox" as const;

/** PADDING is the gap between a group box's edge and its members' bounds. */
export const PADDING = 20;

/** TOP_PADDING is the larger gap above the members' bounds. */
export const TOP_PADDING = 40;

export const configZ = z.object({
  variant: z.literal(VARIANT),
  members: z.string().array(),
  locked: z.boolean().optional(),
});
export type Config = z.infer<typeof configZ>;
