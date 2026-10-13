// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Flex } from "@synnaxlabs/lyra/flex";
import { LinePlot } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import {
  XAxisChannelSelect,
  XAxisRangeSelect,
  YAxisChannelSelect,
} from "@/feature/lineplot/SelectAxis";
import { FLAGS } from "@/flags";
import { CSS } from "@/platform/css";

export const Data = (): ReactElement => {
  const x1 = LinePlot.useXAxis({ axisKey: "x1" });
  const x2 = LinePlot.useXAxis({ axisKey: "x2" });
  const x2Ranges = LinePlot.useXAxisRanges({ axisKey: "x2" });
  const showX2 = FLAGS.lineplotWindows && x2Ranges.length > 0;
  return (
    <Flex.Box className={CSS.BE("line-plot", "toolbar", "data")} full="x">
      {x1.mode !== "spectrum" && (
        <XAxisChannelSelect
          axisKey="x1"
          className={CSS.BE("line-plot", "toolbar", "data-x")}
        />
      )}
      <XAxisRangeSelect axisKey="x1" grow />
      <YAxisChannelSelect axisKey="y1" align="center" grow />
      <YAxisChannelSelect axisKey="y2" grow />
      {showX2 && x2.mode !== "spectrum" && (
        <XAxisChannelSelect
          axisKey="x2"
          className={CSS.BE("line-plot", "toolbar", "data-x")}
        />
      )}
      {showX2 && <XAxisRangeSelect axisKey="x2" grow />}
    </Flex.Box>
  );
};
