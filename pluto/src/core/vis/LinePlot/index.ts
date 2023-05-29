// Copyright 2023 Synnax Labs, Inc.
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { LinePlotC } from "@/core/vis/LinePlot/LinePlotC";
import { XAxisC } from "@/core/vis/LinePlot/XAxisC";
import { YAxisC } from "@/core/vis/LinePlot/YAxisC";

type CoreLinePlotType = typeof LinePlotC;

interface LinePlotType extends CoreLinePlotType {
  TYPE: "LinePlot";
  YAxis: typeof YAxisC;
  XAxis: typeof XAxisC;
}

export const LinePlot = LinePlotC as LinePlotType;

LinePlot.TYPE = "LinePlot";
LinePlot.YAxis = YAxisC;
LinePlot.XAxis = XAxisC;
