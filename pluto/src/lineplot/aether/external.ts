// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type aether } from "@/aether/aether";

export * from "@/lineplot/aether/BoundQuerier";
export * from "@/lineplot/aether/LinePlot";
export * from "@/lineplot/aether/XAxis";
export * from "@/lineplot/aether/YAxis";
import { BoundQuerier } from "@/lineplot/aether/BoundQuerier";
import { LinePlot } from "@/lineplot/aether/LinePlot";
import { XAxis } from "@/lineplot/aether/XAxis";
import { YAxis } from "@/lineplot/aether/YAxis";
import { annotation } from "@/lineplot/annotation/aether";
import { range } from "@/lineplot/range/aether";
import { tooltip } from "@/lineplot/tooltip/aether";

export const REGISTRY: aether.ComponentRegistry = {
  ...annotation.REGISTRY,
  ...range.REGISTRY,
  ...tooltip.REGISTRY,
  [LinePlot.TYPE]: LinePlot,
  [XAxis.TYPE]: XAxis,
  [YAxis.TYPE]: YAxis,
  [BoundQuerier.TYPE]: BoundQuerier,
};
