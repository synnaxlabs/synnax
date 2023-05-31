// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Box, outerLocation, xy } from "@synnaxlabs/x";
import { z } from "zod";

import { hex } from "@/core/color";
import {
  TickFactoryContext,
  tickFactoryProps,
  tickType,
} from "@/core/vis/Axis/TickFactory";

export const axisProps = tickFactoryProps.extend({
  color: hex,
  position: xy,
  label: z.string().optional().default(""),
  type: tickType,
  font: z.string(),
  showGrid: z.boolean().optional().default(false),
  location: outerLocation,
});

export type AxisProps = z.infer<typeof axisProps>;

export interface AxisContext extends Omit<TickFactoryContext, "size"> {
  plottingRegion: Box;
}

export interface Axis {
  setProps: (props: AxisProps) => void;
  render: (ctx: AxisContext) => void;
}
