// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TelemSourceProps } from "@/core/vis/telem/TelemSource";
import {
  DynamicRangeXYTelem,
  DynamicRangeXYTelemProps,
  RangeNumericTelem,
  RangeNumerictelemProps,
  RangeXYTelem,
  RangeXYTelemProps,
} from "@/telem/remote/aether";

const useXY = (props: RangeXYTelemProps): TelemSourceProps => {
  return {
    type: RangeXYTelem.TYPE,
    props,
  };
};

const useDynamicXY = (props: DynamicRangeXYTelemProps): TelemSourceProps => {
  return {
    type: DynamicRangeXYTelem.TYPE,
    props,
  };
};

const useNumeric = (props: RangeNumerictelemProps): TelemSourceProps => {
  return {
    type: RangeNumericTelem.TYPE,
    props,
  };
};

export const RangeTelem = {
  useXY,
  useDynamicXY,
  useNumeric,
};
