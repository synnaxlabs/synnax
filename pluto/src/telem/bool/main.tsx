// Copyright 2023 synnax labs, inc.
//
// use of this software is governed by the business source license included in the file
// licenses/bsl.txt.
//
// as of the change date specified in that file, in accordance with the business source
// license, use of this software will be governed by the apache license, version 2.0,
// included in the file licenses/apl.txt.

import { BooleanTelemSinkSpec, BooleanTelemSourceSpec } from "@/vis/telem";
import { AetherBooleanTelem } from "@/telem/bool/aether";

export namespace BooleanTelem {
  export const useNumericConverterSink = (
    props: AetherBooleanTelem.NumericConverterSinkProps
  ): BooleanTelemSinkSpec => {
    return {
      props,
      type: AetherBooleanTelem.NumericConverterSink.TYPE,
      variant: "boolean-sink",
    };
  };

  export const useNumericConverterSource = (
    props: AetherBooleanTelem.NumericConverterSourceProps
  ): BooleanTelemSourceSpec => {
    return {
      props,
      type: AetherBooleanTelem.NumericConverterSource.TYPE,
      variant: "boolean-source",
    };
  };
}
