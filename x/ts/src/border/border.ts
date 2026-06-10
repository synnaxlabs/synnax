// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

import { xy } from "@/spatial";

const cornerRadiusZ = z.object({
  topLeft: xy.xyZ,
  topRight: xy.xyZ,
  bottomLeft: xy.xyZ,
  bottomRight: xy.xyZ,
});

const cornerNumberZ = z.object({
  topLeft: z.number(),
  topRight: z.number(),
  bottomLeft: z.number(),
  bottomRight: z.number(),
});

const directionRadiusZ = z.object({ x: z.number(), y: z.number() });

/**
 * radiusZ accepts a border radius in any of its crude forms: a bare number, a
 * single { x, y } pair, per-corner numbers, or the canonical per-corner pairs.
 */
export const radiusZ = z.union([
  z.number(),
  directionRadiusZ,
  cornerNumberZ,
  cornerRadiusZ,
]);

export type Radius = z.infer<typeof radiusZ>;
