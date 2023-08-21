// Copyright 2023 Synnax Labs, Inc.
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Legend } from "@/vis/lineplot/main/Legend";
import { Line } from "@/vis/lineplot/main/Line";
import { LinePlot as CoreLinePlot } from "@/vis/lineplot/main/LinePlot";
import { Title } from "@/vis/lineplot/main/Title";
import { Viewport } from "@/vis/lineplot/main/Viewport";
import { XAxis } from "@/vis/lineplot/main/XAxis";
import { YAxis } from "@/vis/lineplot/main/YAxis";
import { Rule } from "@/vis/rule";

export type { LinePlotProps } from "@/vis/lineplot/main/LinePlot";

type CoreLinePlotType = typeof CoreLinePlot;

interface LinePlotType extends CoreLinePlotType {
  YAxis: typeof YAxis;
  XAxis: typeof XAxis;
  Line: typeof Line;
  Legend: typeof Legend;
  Title: typeof Title;
  Rule: typeof Rule;
  Viewport: typeof Viewport;
}

export const LinePlot = CoreLinePlot as LinePlotType;

LinePlot.YAxis = YAxis;
LinePlot.XAxis = XAxis;
LinePlot.Line = Line;
LinePlot.Legend = Legend;
LinePlot.Title = Title;
LinePlot.Rule = Rule;
LinePlot.Viewport = Viewport;
