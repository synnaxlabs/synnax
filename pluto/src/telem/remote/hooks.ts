// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type telem } from "@/telem/core";
import {
  NumericSource,
  type NumericSourceProps as RemoteTelemNumericProps,
} from "@/telem/remote/aether/numeric";
import {
  type XYSourceProps as RemoteTelemXYProps,
  type DynamicXYSourceProps as RemoteTelemDynamicXyProps,
  XYSource,
  DynamicXYSource,
} from "@/telem/remote/aether/xy";

export const useXYSource = (props: RemoteTelemXYProps): telem.XYSourceSpec => {
  return {
    type: XYSource.TYPE,
    props,
    variant: "xy-source",
  };
};

export const useDynamicXYSource = (
  props: RemoteTelemDynamicXyProps,
): telem.XYSourceSpec => {
  return {
    type: DynamicXYSource.TYPE,
    props,
    variant: "xy-source",
  };
};

export const useNumericSource = (
  props: RemoteTelemNumericProps,
): telem.NumericSourceSpec => {
  return {
    type: NumericSource.TYPE,
    props,
    variant: "numeric-source",
  };
};
