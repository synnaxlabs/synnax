// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { bounds, color, location, notation, text, xy } from "@synnaxlabs/x";
import { z } from "zod";

import { Label } from "@/schematic/node/common/label";
import { telem } from "@/telem/aether";

export const VARIANT = "gauge" as const;

export const configZ = Label.labeledConfigZ.extend({
  variant: z.literal(VARIANT),
  position: xy.xyZ.optional(),
  color: color.crudeZ.optional(),
  bounds: bounds.boundsZ().optional(),
  barWidth: z.number().optional(),
  telem: telem.stringSourceSpecZ.optional(),
  precision: z.number().optional(),
  stalenessTimeout: z.number().optional(),
  stalenessColor: color.colorZ.optional(),
  minWidth: z.number().optional(),
  width: z.number().optional(),
  notation: notation.notationZ.optional(),
  location: location.xy.optional(),
  units: z.string().optional(),
  level: text.levelZ.optional(),
});
export type Config = z.infer<typeof configZ>;
