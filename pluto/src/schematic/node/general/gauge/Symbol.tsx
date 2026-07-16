// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type schematic } from "@synnaxlabs/client";
import { box, dimensions, type text, xy } from "@synnaxlabs/x";
import { type ReactElement, useMemo } from "react";

import { CSS } from "@/css";
import { Grid } from "@/schematic/node/common/grid";
import { Label } from "@/schematic/node/common/label";
import * as CommonTelem from "@/schematic/node/common/telem";
import { type NodeProps } from "@/schematic/node/spec";
import { Gauge as BaseGauge } from "@/vis/gauge";

const GAUGE_SIZE_MULTIPLIER: Record<text.Level, number> = {
  h1: 220,
  h2: 190,
  h3: 160,
  h4: 130,
  h5: 100,
  p: 85,
  small: 80,
} as const;

export const Symbol = ({
  nodeKey,
  position,
  onConfigChange,
  selected,
  config: {
    label,
    level = "p",
    color,
    channel,
    rollingAverage,
    precision,
    units,
    notation,
    bounds,
    barWidth,
  },
}: NodeProps<schematic.NodeConfigGauge>): ReactElement => {
  const dims = useMemo(
    () => dimensions.construct(GAUGE_SIZE_MULTIPLIER[level] ?? 100),
    [level],
  );
  const telem = useMemo(
    () => CommonTelem.stringSource({ channel, rollingAverage, precision, notation }),
    [channel, rollingAverage, precision, notation],
  );
  BaseGauge.use({
    aetherKey: nodeKey,
    box: box.construct(position ?? xy.ZERO, dims),
    telem,
    color,
    level,
    units,
    bounds,
    notation,
    barWidth,
  });
  return (
    <Grid.Grid editable={selected} nodeKey={nodeKey} allowRotate={false}>
      <Label.Label config={label} onChange={onConfigChange} />
      <div style={dims} className={CSS.B("symbol-primitive")} />
    </Grid.Grid>
  );
};
