// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { RangeSelect } from "@/feature/lineplot/RangeSelect";
import { XAxisChannelSelect, YAxisChannelSelect } from "@/feature/lineplot/SelectAxis";
import { CSS } from "@/platform/css";

const ROW = CSS.BE("line-plot", "toolbar", "data-row");
const FULL_ROW = CSS.cls(ROW, CSS.BEM("line-plot", "toolbar-data-row", "full"));

export const Data = (): ReactElement => (
  <div className={CSS.BE("line-plot", "toolbar", "data")}>
    <RangeSelect className={ROW} />
    <XAxisChannelSelect
      axisKey="x1"
      className={CSS.BE("line-plot", "toolbar", "data-x")}
    />
    <YAxisChannelSelect axisKey="y1" className={FULL_ROW} />
    <YAxisChannelSelect axisKey="y2" className={FULL_ROW} />
  </div>
);
