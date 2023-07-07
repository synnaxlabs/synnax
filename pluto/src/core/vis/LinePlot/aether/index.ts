// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { AetherComponentRegistry } from "@/core/aether/worker";
import { AetherLinePlot as CoreLinePlot } from "@/core/vis/LinePlot/aether/LinePlot";
import { AetherXAxis } from "@/core/vis/LinePlot/aether/XAxis";
import { AetherYAxis } from "@/core/vis/LinePlot/aether/YAxis";

export const LinePlotRegistry: AetherComponentRegistry = {
  [CoreLinePlot.TYPE]: (u) => new CoreLinePlot(u),
  [AetherXAxis.TYPE]: (u) => new AetherXAxis(u),
  [AetherYAxis.TYPE]: (u) => new AetherYAxis(u),
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
