// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { bounds, color, scale } from "@synnaxlabs/x";
import { z } from "zod";

import { telem } from "@/telem/aether";

export const redlineZ = z.object({
  bounds: bounds.boundsZ(),
  gradient: color.gradientZ,
});
export type Redline = z.infer<typeof redlineZ>;
export const ZERO_REDLINE: Redline = { bounds: { lower: 0, upper: 1 }, gradient: [] };

/**
 * Builds the color source that paints a value's background, mapping the value's own
 * telemetry through the redline bounds onto its gradient.
 * @param source - The value's telemetry, read as the number to map.
 */
export const backgroundTelem = (
  source: telem.StringSourceSpec,
  { bounds, gradient }: Redline,
): telem.ColorSourceSpec =>
  telem.sourcePipeline("color", {
    connections: [
      { from: "source", to: "scale" },
      { from: "scale", to: "gradient" },
    ],
    segments: {
      source,
      scale: telem.scaleNumber({
        scale: scale.Scale.scale<number>(bounds).scale(0, 1).transform,
      }),
      gradient: telem.colorGradient({ gradient }),
    },
    outlet: "gradient",
  });
