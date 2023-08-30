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

import { telem } from "@/telem/core";
import { staticTelem } from "@/telem/static/aether";

export const useStaticXY = (props: staticTelem.XYProps): telem.XYSourceSpec => {
  const transfer = useMemo(
    () => [...props.x.map((x) => x.buffer), ...props.y.map((y) => y.buffer)],
    [props]
  );
  return {
    type: staticTelem.XY.TYPE,
    props,
    transfer,
    variant: "xy-source",
  };
};

export const useIterativeXY = (
  props: staticTelem.IterativeXYProps
): telem.XYSourceSpec => {
  return useMemo(
    () => ({
      variant: "xy-source",
      type: staticTelem.IterativeXY.TYPE,
      props: {
        ...props,
        rate: new Rate(props.rate).valueOf(),
      },
      transfer: [...props.x.map((x) => x.buffer), ...props.y.map((y) => y.buffer)],
    }),
    []
  );
};

export const useNumeric = (value: number): telem.NumericSourceSpec => {
  return {
    type: staticTelem.Numeric.TYPE,
    props: value,
    variant: "numeric-source",
  };
};
