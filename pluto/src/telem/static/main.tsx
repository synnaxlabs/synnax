// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Rate } from "@synnaxlabs/x";

import { XYTelemSourceMeta } from "@/core/vis/telem";
import { useTelemSourceControl } from "@/telem/Context";
import {
  IterativeXYTelem,
  IterativeXYTelemProps,
  StaticXYTelem,
  StaticXYTelemProps,
} from "@/telem/static/worker";

const useStaticXYTelem = (props: StaticXYTelemProps): XYTelemSourceMeta => {
  const key = useTelemSourceControl(StaticXYTelem.TYPE, props, [
    ...props.x.map((x) => x.buffer),
    ...props.y.map((y) => y.buffer),
  ]);
  return {
    variant: "xy",
    key,
  };
};

const useIterativeXYTelem = (props: IterativeXYTelemProps): XYTelemSourceMeta => {
  const key = useTelemSourceControl(
    IterativeXYTelem.TYPE,
    {
      ...props,
      rate: new Rate(props.rate).valueOf(),
    },
    [...props.x.map((x) => x.buffer), ...props.y.map((y) => y.buffer)]
  );
  return {
    variant: "xy",
    key,
  };
};

export const StaticTelem = {
  useXY: useStaticXYTelem,
  useIterativeXY: useIterativeXYTelem,
};
