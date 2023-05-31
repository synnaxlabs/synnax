// Copyright 2023 Synnax Labs, Inc.
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { LinePlotC } from "@/core/vis/LinePlot/main/LinePlot";
import { XAxis } from "@/core/vis/LinePlot/main/XAxis";
import { YAxis } from "@/core/vis/LinePlot/main/YAxis";

type CoreLinePlotType = typeof LinePlotC;

interface LinePlotType extends CoreLinePlotType {
  TYPE: "LinePlot";
  YAxis: typeof YAxis;
  XAxis: typeof XAxis;
}

export const LinePlot = LinePlotC as LinePlotType;

LinePlot.TYPE = "LinePlot";
LinePlot.YAxis = YAxis;
LinePlot.XAxis = XAxis;
