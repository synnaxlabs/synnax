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

import { XAxisRangeSelect, YAxisChannelSelect } from "@/feature/lineplot/SelectAxis";
import { XAxisInput } from "@/feature/lineplot/XAxisInput";
import { FLAGS } from "@/flags";
import { CSS } from "@/platform/css";

export const Data = (): ReactElement => {
  const x2Ranges = LinePlot.useXAxisRanges({ axisKey: "x2" });
  const showX2 = FLAGS.lineplotWindows && x2Ranges.length > 0;
  return (
    <Flex.Box className={CSS.BE("line-plot", "toolbar", "data")} full="x">
      <XAxisInput axisKey="x1" />
      <XAxisRangeSelect axisKey="x1" grow />
      <YAxisChannelSelect axisKey="y1" align="center" grow />
      <YAxisChannelSelect axisKey="y2" grow />
      {showX2 && <XAxisInput axisKey="x2" />}
      {showX2 && <XAxisRangeSelect axisKey="x2" grow />}
    </Flex.Box>
  );
};
