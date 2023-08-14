// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TelemSpec } from "@/core/vis/telem";
import {
  Numeric,
  NumericProps as RemoteTelemNumericProps,
} from "@/telem/remote/aether/numeric";
import {
  XYProps as RemoteTelemXYProps,
  DynamicXYProps as RemoteTelemDynamicXyProps,
  XY,
  DynamicXY,
} from "@/telem/remote/aether/xy";

export const useXY = (props: RemoteTelemXYProps): TelemSpec => {
  return {
    type: XY.TYPE,
    props,
  };
};

export const useDynamicXY = (props: RemoteTelemDynamicXyProps): TelemSpec => {
  return {
    type: DynamicXY.TYPE,
    props,
  };
};

export const useNumeric = (props: RemoteTelemNumericProps): TelemSpec => {
  return {
    type: Numeric.TYPE,
    props,
  };
};
