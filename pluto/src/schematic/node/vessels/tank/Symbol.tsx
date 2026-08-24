// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { border, box, type dimensions, xy } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Border } from "@/schematic/node/common/border";
import { Grid } from "@/schematic/node/common/grid";
import { Label } from "@/schematic/node/common/label";
import { type Scale } from "@/schematic/node/common/scale";
import { type NodeProps } from "@/schematic/node/spec";
import { type Config } from "@/schematic/node/vessels/tank/config";
import { Tank } from "@/schematic/node/vessels/tank/Primitive";
import { Scale as BaseScale } from "@/vis/scale";

const STROKE_WIDTH = 2;
// The canvas fill and the DOM wall round to device pixels independently, so the fill is
// aimed at the middle of the stroke. Half the stroke absorbs the difference.
const OVERLAP = STROKE_WIDTH / 2;

// Converts the tank's percentage-based CSS border radius into the pixel radii of the
// curve the fill is clipped to, inset by the same overlap.
const cornerRadii = (
  borderRadius: Config["borderRadius"],
  dims: dimensions.Dimensions,
): border.Radius => {
  const detailed = border.constructRadius(borderRadius ?? Border.DEFAULT_RADIUS);
  const radius = (corner: xy.XY): xy.XY =>
    xy.construct(
      Math.max(0, (corner.x / 100) * dims.width - OVERLAP),
      Math.max(0, (corner.y / 100) * dims.height - OVERLAP),
    );
  return {
    topLeft: radius(detailed.topLeft),
    topRight: radius(detailed.topRight),
    bottomLeft: radius(detailed.bottomLeft),
    bottomRight: radius(detailed.bottomRight),
  };
};

interface FillProps extends Pick<Config, "borderRadius"> {
  nodeKey: string;
  position?: xy.XY;
  dimensions: dimensions.Dimensions;
  fill: Scale.Config;
}

const Fill = ({
  nodeKey,
  position,
  dimensions: dims,
  fill,
  borderRadius,
}: FillProps): null => {
  BaseScale.use({
    ...fill,
    aetherKey: nodeKey,
    box: box.construct(xy.translate(position ?? xy.ZERO, OVERLAP), {
      width: dims.width - OVERLAP * 2,
      height: dims.height - OVERLAP * 2,
    }),
    direction: "y",
    externalScale: true,
    cornerRadii: cornerRadii(borderRadius, dims),
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
    fill,
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
    {fill?.telem != null && (
      <Fill
        nodeKey={nodeKey}
        position={position}
        dimensions={dimensions}
        fill={fill}
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
