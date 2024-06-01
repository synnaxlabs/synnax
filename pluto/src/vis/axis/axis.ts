// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type box, location, type xy } from "@synnaxlabs/x";
import { z } from "zod";

import { color } from "@/color/core";
import { type TickFactoryContext, tickFactoryProps, tickType } from "@/vis/axis/ticks";

export interface RenderResult {
  size: number;
}

export const axisStateZ = tickFactoryProps.extend({
  color: color.Color.z,
  type: tickType.optional().default("linear"),
  font: z.string(),
  showGrid: z.boolean().optional().default(true),
  location: location.outer,
  gridColor: color.Color.z,
});

export type AxisState = z.input<typeof axisStateZ>;
export type ParsedAxisState = z.output<typeof axisStateZ>;

export interface AxisProps extends Omit<TickFactoryContext, "size"> {
  plot: box.Box;
  position: xy.XY;
  size: number;
}

export interface Axis {
  setState: (state: AxisState) => void;
  render: (props: AxisProps) => RenderResult;
}
