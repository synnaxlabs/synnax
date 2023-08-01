// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Box, Location, XY } from "@synnaxlabs/x";
import { z } from "zod";

import { Color } from "@/core/color";
import {
  TickFactoryContext,
  tickFactoryProps,
  tickType,
} from "@/core/vis/Axis/TickFactory";

export interface AxisRenderResult {
  size: number;
}

export const axisState = tickFactoryProps.extend({
  color: Color.z,
  type: tickType,
  font: z.string(),
  showGrid: z.boolean().optional().default(true),
  location: Location.strictOuterZ,
  gridColor: Color.z,
  size: z.number().default(0),
});

export type AxisState = z.input<typeof axisState>;
export type ParsedAxisState = z.output<typeof axisState>;

export interface AxisProps extends Omit<TickFactoryContext, "size"> {
  plot: Box;
  position: XY;
}

export interface Axis {
  setState: (state: AxisState) => void;
  render: (props: AxisProps) => AxisRenderResult;
}
