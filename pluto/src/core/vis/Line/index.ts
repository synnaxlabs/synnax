// Copyright 2023 Synnax Labs, Inc.
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { LineC } from "@/core/vis/Line/LineC";
import { LineFactory } from "@/core/vis/Line/LineGL";
export type { LineRenderer, LineProps } from "@/core/vis/Line/core";
export type { LineFactory } from "@/core/vis/Line/LineGL";
export type { LineCProps } from "@/core/vis/Line/LineC";

type CoreLineType = typeof LineC;

interface LineType extends CoreLineType {
  TYPE: "Line";
  Factory: typeof LineFactory;
}

export const Line = LineC as LineType;

Line.TYPE = "Line";
Line.Factory = LineFactory;
