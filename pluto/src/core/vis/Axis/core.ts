// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Box, OuterLocation, XY } from "@synnaxlabs/x";

import {
  TickType,
  TickFactoryProps,
  TickFactoryContext,
} from "@/core/vis/Axis/TickFactory";

export interface AxisProps extends TickFactoryProps {
  color: string;
  label: string;
  tickSpacing: number;
  type: TickType;
  tickFont: string;
  showGrid?: boolean;
  gridSize?: number;
  location: OuterLocation;
}

export interface AxisContext extends TickFactoryContext {
  region: Box;
  position: XY;
}

export interface Axis {
  setProps: (props: AxisProps) => void;
  render: (ctx: AxisContext) => void;
}
