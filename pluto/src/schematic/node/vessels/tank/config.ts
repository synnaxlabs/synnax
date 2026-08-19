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

import { Border } from "@/schematic/node/common/border";
import { Label } from "@/schematic/node/common/label";
import { Scale } from "@/schematic/node/common/scale";

export const VARIANT = "tank" as const;

export const configZ = Label.labeledConfigZ.extend({
  variant: z.literal(VARIANT),
  position: xy.xyZ.optional(),
  color: color.crudeZ.optional(),
  backgroundColor: color.crudeZ.optional(),
  dimensions: dimensions.dimensionsZ.optional(),
  borderRadius: Border.radiusZ.optional(),
  // The fill is rendered only when its telem is set, so a tank without telemetry, and
  // one stored before the fill existed, render unchanged.
  fill: Scale.configZ.optional(),
});
export type Config = z.infer<typeof configZ>;

/** Fill defaults for a tank, which draws its own wall and reads best with no scale. */
export const FILL_DEFAULTS: Partial<Scale.Config> = {
  showCaret: false,
  showScale: false,
  side: "left",
};
