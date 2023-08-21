// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { AetherComponentRegistry } from "@/aether/aether";
import { AetherLinePlot as CoreLinePlot } from "@/vis/lineplot/aether/LinePlot";
import { AetherXAxis } from "@/vis/lineplot/aether/XAxis";
import { AetherYAxis } from "@/vis/lineplot/aether/YAxis";

export const LinePlotRegistry: AetherComponentRegistry = {
  [CoreLinePlot.TYPE]: CoreLinePlot,
  [AetherXAxis.TYPE]: AetherXAxis,
  [AetherYAxis.TYPE]: AetherYAxis,
};

type CoreLinePlotType = typeof CoreLinePlot;

interface AetherLinePlotType extends CoreLinePlotType {
  REGISTRY: AetherComponentRegistry;
  XAxis: typeof AetherXAxis;
  YAxis: typeof AetherYAxis;
}

export const AetherLinePlot = CoreLinePlot as AetherLinePlotType;

AetherLinePlot.REGISTRY = LinePlotRegistry;
AetherLinePlot.XAxis = AetherXAxis;
AetherLinePlot.YAxis = AetherYAxis;
