// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type box } from "@synnaxlabs/x/box";
import { color } from "@synnaxlabs/x/color";
import { location } from "@synnaxlabs/x/location";
import { type xy } from "@synnaxlabs/x/xy";
import { z } from "zod";

import { tickFactoryProps, type TickFactoryRenderArgs, tickType } from "@/axis/ticks";

export interface RenderResult {
  size: number;
}

export const axisStateZ = tickFactoryProps.extend({
  color: color.colorZ,
  type: tickType.default("linear"),
  font: z.string(),
  showGrid: z.boolean().default(true),
  location: location.outerZ,
  gridColor: color.colorZ,
});

export type AxisState = z.input<typeof axisStateZ>;
export type ParsedAxisState = z.infer<typeof axisStateZ>;

export interface AxisProps extends Omit<TickFactoryRenderArgs, "size"> {
  plot: box.Box;
  position: xy.XY;
  size: number;
}

export interface Axis {
  setState: (state: AxisState) => void;
  render: (props: AxisProps) => RenderResult;
}
