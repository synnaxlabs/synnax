// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Flex } from "@synnaxlabs/lyra/flex";
import { type ReactElement } from "react";

import { XAxisRangeSelect, YAxisChannelSelect } from "@/feature/lineplot/SelectAxis";
import { XAxisInput } from "@/feature/lineplot/XAxisInput";
import { CSS } from "@/platform/css";

export const Data = (): ReactElement => (
  <Flex.Box className={CSS.BE("line-plot", "toolbar", "data")} full="x">
    <XAxisInput axisKey="x1" />
    <XAxisRangeSelect axisKey="x1" grow />
    <YAxisChannelSelect axisKey="y1" align="center" grow />
    <YAxisChannelSelect axisKey="y2" grow />
  </Flex.Box>
);
