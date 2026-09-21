// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color, xy } from "@synnaxlabs/x";
import { z } from "zod";

export const VARIANT = "line" as const;

export const configZ = z.object({
  variant: z.literal(VARIANT),
  color: color.crudeZ.optional(),
  // Endpoints are offsets from the node position, which is their top-left corner.
  start: xy.xyZ,
  end: xy.xyZ,
  strokeWidth: z.number().optional(),
});
export type Config = z.infer<typeof configZ>;
