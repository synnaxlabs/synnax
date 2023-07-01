// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { XYTelemSourceProps } from "@/core/vis/telem";
import { NumericTelemSourceProps } from "@/core/vis/telem/TelemSource";
import {
  DynamicRangeXYTelem,
  DynamicRangeXYTelemProps,
  RangeNumericTelem,
  RangeNumerictelemProps,
  RangeXYTelem,
  RangeXYTelemProps,
} from "@/telem/range/aether";

export const useRangeXYTelem = (props: RangeXYTelemProps): XYTelemSourceProps => {
  return {
    variant: "xy",
    type: RangeXYTelem.TYPE,
    props,
  };
};

export const useDynamicRangeXYTelem = (
  props: DynamicRangeXYTelemProps
): XYTelemSourceProps => {
  return {
    type: DynamicRangeXYTelem.TYPE,
    variant: "xy",
    props,
  };
};

export const usePointRangeTelem = (
  props: RangeNumerictelemProps
): NumericTelemSourceProps => {
  return {
    type: RangeNumericTelem.TYPE,
    variant: "numeric",
    props,
  };
};

export const RangeTelem = {
  useXY: useRangeXYTelem,
  useDynamicXY: useDynamicRangeXYTelem,
  useNumeric: usePointRangeTelem,
};
