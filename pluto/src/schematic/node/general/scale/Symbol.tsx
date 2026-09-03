// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { box, dimensions, location, xy } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { CSS } from "@/css";
import { Grid } from "@/schematic/node/common/grid";
import { Label } from "@/schematic/node/common/label";
import {
  axis,
  type Config,
  DEFAULT_DIMENSIONS,
} from "@/schematic/node/general/scale/config";
import { type NodeProps } from "@/schematic/node/spec";
import { Scale as BaseScale } from "@/vis/scale";

export const Symbol = ({
  nodeKey,
  position,
  onConfigChange,
  selected,
  config: {
    label,
    color,
    dimensions: dims = DEFAULT_DIMENSIONS,
    orientation = "top",
    indicator,
  },
}: NodeProps<Config>): ReactElement => {
  const dir = axis(orientation);
  // The configured dimensions are the bar's own. The ticks live beside it, so the
  // symbol takes the gutter on top of them.
  const gutter = BaseScale.gutter(indicator);
  const outer: dimensions.Dimensions =
    dir === "y"
      ? { width: dims.width + gutter, height: dims.height }
      : { width: dims.width, height: dims.height + gutter };
  BaseScale.use({
    ...indicator,
    color,
    aetherKey: nodeKey,
    box: box.construct(position ?? xy.ZERO, outer),
    direction: dir,
  });
  // The bar has one orientation per axis, so rotating it toggles between them.
  const handleRotate = (): void =>
    onConfigChange({
      orientation: dir === "y" ? "right" : "top",
      dimensions: dimensions.swap(dims),
      indicator: {
        ...indicator,
        side: location.swapAxis(indicator.side),
        caretSide: location.swapAxis(indicator.caretSide),
      },
    });
  const handleResize = ({ width, height }: dimensions.Dimensions): void =>
    onConfigChange({
      dimensions:
        dir === "y"
          ? { width: Math.max(0, width - gutter), height }
          : { width, height: Math.max(0, height - gutter) },
    });
  return (
    <Grid.Grid
      editable={selected}
      nodeKey={nodeKey}
      onRotate={handleRotate}
      onResize={handleResize}
    >
      <Label.Label config={label} onChange={onConfigChange} />
      <div style={outer} className={CSS.B("symbol-primitive")} />
    </Grid.Grid>
  );
};
