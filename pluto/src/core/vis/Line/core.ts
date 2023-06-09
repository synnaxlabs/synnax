// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Bounds, Box, XYScale } from "@synnaxlabs/x";
import { z } from "zod";

import { xyTelemSourceMeta } from "../telem/TelemSource";

import { AetherComponent } from "@/core/aether/worker";
import { Color } from "@/core/color";

export const lineState = z.object({
  telem: xyTelemSourceMeta,
  color: Color.z,
  strokeWidth: z.number().default(1),
});

export type LineState = z.input<typeof lineState>;
export type ParsedLineState = z.output<typeof lineState>;

export interface LineContext {
  /**
   * A box in pixel space representing the region of the display that the line
   * should be rendered in. The root of the pixel coordinate system is the top
   * left of the canvas.
   */
  region: Box;
  /**
   * An XY scale that maps from the data space to decimal space rooted in the
   * bottom of the region.
   */
  scale: XYScale;
}

export interface LineComponent extends AetherComponent {
  state: LineState;
  render: (ctx: LineContext) => void;
  xBounds: () => Promise<Bounds>;
  yBounds: () => Promise<Bounds>;
}
