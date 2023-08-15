// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useMemo } from "react";

import { Rate } from "@synnaxlabs/x";

import { NumericTelemSourceSpec, XYTelemSourceSpec } from "@/core/vis/telem";
import {
  IterativeXYTelem,
  IterativeXYTelemProps,
  StaticNumericTelem,
  StaticXYTelem,
  StaticXYTelemProps,
} from "@/telem/static/aether";

export namespace StaticTelem {
  export const useStaticXY = (props: StaticXYTelemProps): XYTelemSourceSpec => {
    const transfer = useMemo(
      () => [...props.x.map((x) => x.buffer), ...props.y.map((y) => y.buffer)],
      [props]
    );
    return {
      type: StaticXYTelem.TYPE,
      props,
      transfer,
      variant: "xy-source",
    };
  };

  export const useIterativeXY = (props: IterativeXYTelemProps): XYTelemSourceSpec => {
    return useMemo(
      () => ({
        variant: "xy-source",
        type: IterativeXYTelem.TYPE,
        props: {
          ...props,
          rate: new Rate(props.rate).valueOf(),
        },
        transfer: [...props.x.map((x) => x.buffer), ...props.y.map((y) => y.buffer)],
      }),
      []
    );
  };

  export const useNumeric = (value: number): NumericTelemSourceSpec => {
    return {
      type: StaticNumericTelem.TYPE,
      props: value,
      variant: "numeric-source",
    };
  };
}
