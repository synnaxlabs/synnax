// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/schematic/node/common/primitive/primitive.css";

import { type schematic } from "@synnaxlabs/client";
import { box, dimensions, type text, xy } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { CSS } from "@/css";
import { Grid } from "@/schematic/node/common/grid";
import { Label } from "@/schematic/node/common/label";
import { type NodeProps } from "@/schematic/node/spec";
import { Gauge as BaseGauge } from "@/vis/gauge";
import { Value } from "@/vis/value";

const GAUGE_DIMENSIONS: Record<text.Level, dimensions.Dimensions> = {
  h1: dimensions.construct(220),
  h2: dimensions.construct(190),
  h3: dimensions.construct(160),
  h4: dimensions.construct(130),
  h5: dimensions.construct(100),
  p: dimensions.construct(85),
  small: dimensions.construct(80),
  tiny: dimensions.construct(75),
};

const DEFAULT_DIMENSIONS = dimensions.construct(100);

export const Symbol = ({
  nodeKey,
  position,
  onConfigChange,
  selected,
  config: {
    label,
    level = "p",
    strokeColor,
    channel,
    rollingAverage,
    precision,
    units,
    notation,
    bounds,
    barWidth,
    stalenessColor,
    stalenessTimeout,
  },
}: NodeProps<schematic.GaugeNodeConfig>): ReactElement => {
  const dims = GAUGE_DIMENSIONS[level] ?? DEFAULT_DIMENSIONS;
  const telem = Value.stringSource({ channel, rollingAverage, precision, notation });
  BaseGauge.use({
    aetherKey: nodeKey,
    box: box.construct(position ?? xy.ZERO, dims),
    telem,
    color: strokeColor,
    level,
    units,
    bounds,
    notation,
    barWidth,
    stalenessColor,
    stalenessTimeout,
  });
  return (
    <Grid.Grid editable={selected} nodeKey={nodeKey} allowRotate={false}>
      <Label.Label config={label} onChange={onConfigChange} />
      <div style={dims} className={CSS.B("symbol-primitive")} />
    </Grid.Grid>
  );
};
