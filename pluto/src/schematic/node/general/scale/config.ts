// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { bounds, color, dimensions, text, xy } from "@synnaxlabs/x";
import { z } from "zod";

import { Label } from "@/schematic/node/common/label";
import { telem } from "@/telem/aether";
import { scale } from "@/vis/scale/aether";

export const VARIANT = "scale" as const;

export const sideZ = z.enum(["left", "right"]);
export type Side = z.infer<typeof sideZ>;

export const configZ = Label.labeledConfigZ.extend({
  variant: z.literal(VARIANT),
  position: xy.xyZ.optional(),
  telem: telem.stringSourceSpecZ.optional(),
  bounds: bounds.boundsZ().optional(),
  color: color.crudeZ.optional(),
  style: scale.styleZ.optional(),
  side: sideZ.optional(),
  dimensions: dimensions.dimensionsZ.optional(),
  level: text.levelZ.optional(),
});
export type Config = z.infer<typeof configZ>;
