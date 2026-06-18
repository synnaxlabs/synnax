// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { box, type dimensions, xy } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Border } from "@/schematic/node/common/border";
import { Grid } from "@/schematic/node/common/grid";
import { Label } from "@/schematic/node/common/label";
import { type NodeProps } from "@/schematic/node/spec";
import { type Config } from "@/schematic/node/vessels/tank/config";
import { Tank } from "@/schematic/node/vessels/tank/Primitive";
import { Scale as BaseScale } from "@/vis/scale";
import { type scale } from "@/vis/scale/aether";

const STROKE_WIDTH = 2;

// innerCornerRadii converts the tank's percentage-based CSS border radius into the
// per-corner pixel radii of the interior wall, so the fill clips to the rounded tank.
const innerCornerRadii = (
  borderRadius: Config["borderRadius"],
  dims: dimensions.Dimensions,
): scale.CornerRadii => {
  const detailed = Border.parseRadius(borderRadius ?? Border.DEFAULT_RADIUS);
  const inner = (corner: xy.XY): xy.XY =>
    xy.construct(
      Math.max(0, (corner.x / 100) * dims.width - STROKE_WIDTH),
      Math.max(0, (corner.y / 100) * dims.height - STROKE_WIDTH),
    );
  return {
    topLeft: inner(detailed.topLeft),
    topRight: inner(detailed.topRight),
    bottomLeft: inner(detailed.bottomLeft),
    bottomRight: inner(detailed.bottomRight),
  };
};

interface FillProps extends Pick<
  Config,
  "telem" | "bounds" | "fillColor" | "showScale" | "scaleSide" | "borderRadius"
> {
  nodeKey: string;
  position?: xy.XY;
  dimensions: dimensions.Dimensions;
}

const Fill = ({
  nodeKey,
  position,
  dimensions: dims,
  telem,
  bounds,
  fillColor,
  showScale,
  scaleSide,
  borderRadius,
}: FillProps): null => {
  BaseScale.use({
    aetherKey: nodeKey,
    box: box.construct(position ?? xy.ZERO, dims),
    telem,
    bounds,
    color: fillColor,
    direction: "y",
    style: "fill",
    showScale: showScale ?? false,
    side: scaleSide ?? "left",
    externalScale: true,
    showTrack: false,
    inset: STROKE_WIDTH,
    cornerRadii: innerCornerRadii(borderRadius, dims),
  });
  return null;
};

export const Symbol = ({
  nodeKey,
  position,
  onConfigChange,
  selected,
  config: {
    label,
    orientation = "left",
    backgroundColor,
    color,
    dimensions = Border.DEFAULT_DIMENSIONS,
    borderRadius,
    telem,
    bounds,
    fillColor,
    showScale,
    scaleSide,
  },
}: NodeProps<Config>): ReactElement => (
  <Grid.Grid
    allowCenter
    allowRotate={false}
    editable={selected}
    nodeKey={nodeKey}
    onResize={(dimensions) => onConfigChange({ dimensions })}
  >
    <Label.Label config={label} onChange={onConfigChange} />
    {telem != null && (
      <Fill
        nodeKey={nodeKey}
        position={position}
        dimensions={dimensions}
        telem={telem}
        bounds={bounds}
        fillColor={fillColor}
        showScale={showScale}
        scaleSide={scaleSide}
        borderRadius={borderRadius}
      />
    )}
    <Tank
      orientation={orientation}
      color={color}
      dimensions={dimensions}
      borderRadius={borderRadius}
      backgroundColor={backgroundColor}
    />
  </Grid.Grid>
);
