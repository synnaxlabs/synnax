// Copyright 2023 synnax labs, inc.
//
// use of this software is governed by the business source license included in the file
// licenses/bsl.txt.
//
// as of the change date specified in that file, in accordance with the business source
// license, use of this software will be governed by the apache license, version 2.0,
// included in the file licenses/apl.txt.

import { bool } from "@/telem/bool/aether";
import { telem } from "@/telem/core";

export const useNumericConverterSink = (
  props: bool.NumericConverterSinkProps
): telem.BooleanSinkSpec => {
  return {
    props,
    type: bool.NumericConverterSink.TYPE,
    variant: "boolean-sink",
  };
};

export const useNumericConverterSource = (
  props: bool.NumericConverterSourceProps
): telem.BooleanSourceSpec => {
  return {
    props,
    type: bool.NumericConverterSource.TYPE,
    variant: "boolean-source",
  };
};
