// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useTelemSourceControl } from "../Context";

import { RangeXYTelem, RangeXYTelemProps } from "./worker";

import { XYTelemSourceMeta } from "@/core/vis/telem";

export const useRangeXYTelem = (props: RangeXYTelemProps): XYTelemSourceMeta => {
  return {
    key: useTelemSourceControl(RangeXYTelem.TYPE, props),
    variant: "xy",
  };
};

export const useDynamicRangeXYTelem = (props: RangeXYTelemProps): XYTelemSourceMeta => {
  return {
    key: useTelemSourceControl(RangeXYTelem.TYPE, props),
    variant: "xy",
  };
};

export const RangeTelem = {
  useXY: useRangeXYTelem,
  useDynamicXY: useDynamicRangeXYTelem,
};
