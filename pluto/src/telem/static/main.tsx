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

import { XYTelemSourceMeta } from "@/core/vis/telem";
import { NumericTelemSourceMeta } from "@/core/vis/telem/TelemSource";
import {
  IterativeXYTelem,
  IterativeXYTelemProps,
  StaticPointTelem,
  StaticPointTelemProps,
  StaticXYTelem,
  StaticXYTelemProps,
} from "@/telem/static/aether";
import { useTelemSourceControl } from "@/telem/TelemProvider/TelemProvider";

const useStaticXYTelem = (props: StaticXYTelemProps): XYTelemSourceMeta => {
  const transfer = useMemo(
    () => [...props.x.map((x) => x.buffer), ...props.y.map((y) => y.buffer)],
    [props]
  );
  const key = useTelemSourceControl(StaticXYTelem.TYPE, props, transfer);
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

const usePointTelem = (value: number): NumericTelemSourceMeta => {
  const key = useTelemSourceControl<StaticPointTelemProps>(
    StaticPointTelem.TYPE,
    value
  );
  return {
    variant: "point",
    key,
  };
};

export const StaticTelem = {
  useXY: useStaticXYTelem,
  useIterativeXY: useIterativeXYTelem,
  usePoint: usePointTelem,
};
