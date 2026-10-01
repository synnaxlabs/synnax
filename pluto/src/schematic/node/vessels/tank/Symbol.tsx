// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type schematic } from "@synnaxlabs/client";
import { border, box, type dimensions, xy } from "@synnaxlabs/x";
import { type ReactElement, useMemo } from "react";

import { Border } from "@/schematic/node/common/border";
import { Grid } from "@/schematic/node/common/grid";
import { Label } from "@/schematic/node/common/label";
import { Scale } from "@/schematic/node/common/scale";
import { type NodeProps } from "@/schematic/node/spec";
import { Tank } from "@/schematic/node/vessels/tank/Primitive";
import { Scale as VisScale } from "@/vis/scale";

const STROKE_WIDTH = 2;
// The canvas fill and the DOM wall round to device pixels independently, so the fill is
// aimed at the middle of the stroke. Half the stroke absorbs the difference.
const OVERLAP = STROKE_WIDTH / 2;

// Converts the tank's percentage-based CSS border radius into the pixel radii of the
// curve the fill is clipped to, inset by the same overlap.
const cornerRadii = (
  borderRadius: schematic.TankNodeConfig["borderRadius"],
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

interface FillProps {
  nodeKey: string;
  position?: xy.XY;
  config: schematic.TankNodeConfig;
}

const Fill = ({ nodeKey, position, config }: FillProps): null => {
  const {
    channel,
    rollingAverage,
    borderRadius,
    caretVisible,
    scaleVisible,
    dimensions: dims = Border.DEFAULT_DIMENSIONS,
  } = config;
  const telem = useMemo(
    () => Scale.source({ channel, rollingAverage }),
    [channel, rollingAverage],
  );
  VisScale.use({
    ...Scale.visProps({
      ...config,
      caretHidden: !caretVisible,
      scaleHidden: !scaleVisible,
    }),
    telem,
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
  config,
}: NodeProps<schematic.TankNodeConfig>): ReactElement => {
  const {
    label,
    fillColor,
    strokeColor,
    dimensions = Border.DEFAULT_DIMENSIONS,
    borderRadius,
    channel,
  } = config;
  return (
    <Grid.Grid
      allowCenter
      allowRotate={false}
      editable={selected}
      nodeKey={nodeKey}
      onResize={(dimensions) => onConfigChange({ dimensions })}
    >
      <Label.Label config={label} onChange={onConfigChange} />
      {channel != null && (
        <Fill nodeKey={nodeKey} position={position} config={config} />
      )}
      <Tank
        strokeColor={strokeColor}
        dimensions={dimensions}
        borderRadius={borderRadius}
        fillColor={fillColor}
      />
    </Grid.Grid>
  );
};
