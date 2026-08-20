// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color, dimensions, xy } from "@synnaxlabs/x";
import { z } from "zod";

import { Label } from "@/schematic/node/common/label";
import { Scale } from "@/schematic/node/common/scale";

export const VARIANT = "scale" as const;

export const DEFAULT_DIMENSIONS: dimensions.Dimensions = { width: 60, height: 160 };

export const configZ = Label.labeledConfigZ.extend({
  variant: z.literal(VARIANT),
  position: xy.xyZ.optional(),
  dimensions: dimensions.dimensionsZ.optional(),
  // The fill is what the symbol reads as, so its color is the symbol's own rather than
  // one of the indicator's. The toolbar recolors a selection through this key.
  color: color.crudeZ.default(color.ZERO),
  indicator: Scale.configZ.default(() => Scale.defaultConfig()),
});
export type Config = z.infer<typeof configZ>;
