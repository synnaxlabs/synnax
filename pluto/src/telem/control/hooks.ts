// Copyrght 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { NumericTelemSinkSpec } from "@/core/vis/telem";
import {
  ControlNumericTelemSinkProps,
  ControlledNumericTelemSink,
} from "@/telem/control/aether";

export namespace ControlTelem {
  export const useNumeric = (
    props: ControlNumericTelemSinkProps
  ): NumericTelemSinkSpec => {
    return {
      type: ControlledNumericTelemSink.TYPE,
      props,
      variant: "numeric-sink",
    };
  };
}
