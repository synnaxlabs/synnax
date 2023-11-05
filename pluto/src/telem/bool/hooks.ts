// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { bool } from "@/telem/bool/aether";
import { type telem } from "@/telem/core";

export const useNumericConverterSink = (
  props: bool.NumericConverterSinkProps,
): telem.BooleanSinkSpec => {
  return {
    props,
    type: bool.NumericConverterSink.TYPE,
    variant: "boolean-sink",
  };
};

export const useNumericConverterSource = (
  props: bool.NumericConverterSourceProps,
): telem.BooleanSourceSpec => {
  return {
    props,
    type: bool.NumericConverterSource.TYPE,
    variant: "boolean-source",
  };
};
