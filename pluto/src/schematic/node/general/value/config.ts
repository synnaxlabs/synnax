// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color, location, notation, text, xy } from "@synnaxlabs/x";
import { z } from "zod";

import { Label } from "@/schematic/node/common/label";
import { telem } from "@/telem/aether";
import { Staleness } from "@/vis/staleness";
import { redlineZ } from "@/vis/value/redline";

export const VARIANT = "value" as const;

export const configZ = Label.labeledConfigZ.extend({
  variant: z.literal(VARIANT),
  position: xy.xyZ.optional(),
  color: color.crudeZ.optional(),
  textColor: color.crudeZ.optional(),
  tooltip: z.array(z.string()).optional(),
  redline: redlineZ.optional(),
  units: z.string().optional(),
  inlineSize: z.number().optional(),
  telem: telem.stringSourceSpecZ.optional(),
  backgroundTelem: telem.colorSourceSpecZ.optional(),
  level: text.levelZ.optional(),
  precision: z.number().optional(),
  ...Staleness.configZ.shape,
  minWidth: z.number().optional(),
  notation: notation.notationZ.optional(),
  location: location.xy.optional(),
  useWidthForBackground: z.boolean().optional(),
  valueBackgroundShift: xy.xyZ.optional(),
  valueBackgroundOverScan: xy.xyZ.optional(),
});
export type Config = z.infer<typeof configZ>;
