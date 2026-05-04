// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { box, xy } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { CSS } from "@/css";
import { Grid, type GridItem } from "@/schematic/node/common/grid/Grid";
import { Label } from "@/schematic/node/common/label";
import { type NodeProps } from "@/schematic/node/common/symbol/factories";
import { type Config } from "@/schematic/node/general/gauge/config";
import { type Text } from "@/text";
import { Gauge as BaseGauge } from "@/vis/gauge";

const GAUGE_SIZE_MULTIPLIER: Record<Text.Level, number> = {
  h1: 220,
  h2: 190,
  h3: 160,
  h4: 130,
  h5: 100,
  p: 85,
  small: 80,
} as const;

export const Symbol = ({
  nodeKey: symbolKey,
  position,
  onConfigChange: onChange,
  selected,
  config: data,
}: NodeProps<Config>): ReactElement => {
  const {
    label,
    level = "p",
    color,
    telem: t,
    units,
    notation,
    bounds: b,
    barWidth,
  } = data;
  const baseMultiplier = GAUGE_SIZE_MULTIPLIER[level] ?? 100;
  const gaugeSize = baseMultiplier;

  BaseGauge.use({
    aetherKey: symbolKey,
    box: box.construct(position || xy.ZERO, {
      height: gaugeSize,
      width: gaugeSize,
    }),
    telem: t,
    color,
    level,
    units,
    bounds: b,
    notation,
    barWidth,
  });

  const gridItems: GridItem[] = [];
  const labelItem = Label.gridItem(label, onChange);
  if (labelItem != null) gridItems.push(labelItem);

  return (
    <Grid
      editable={selected}
      symbolKey={symbolKey}
      items={gridItems}
      allowRotate={false}
      onLocationChange={(key, loc) => {
        if (key !== "label") return;
        onChange({ label: { ...label, orientation: loc } });
      }}
    >
      <div
        style={{ width: gaugeSize, height: gaugeSize }}
        className={CSS.B("symbol-primitive")}
      />
    </Grid>
  );
};
