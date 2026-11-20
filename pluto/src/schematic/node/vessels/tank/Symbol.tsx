// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type schematic } from "@synnaxlabs/client";
import { border, box, type dimensions, primitive, xy } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Color } from "@/color";
import { Border } from "@/schematic/node/common/border";
import { Grid } from "@/schematic/node/common/grid";
import { Label } from "@/schematic/node/common/label";
import { Scale } from "@/schematic/node/common/scale";
import { Stale } from "@/schematic/node/common/stale";
import { type NodeProps } from "@/schematic/node/spec";
import { Tank } from "@/schematic/node/vessels/tank/Primitive";
import { Scale as VisScale } from "@/vis/scale";
import { Staleness } from "@/vis/staleness";

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

interface BodyProps extends NodeProps<schematic.TankNodeConfig> {
  staleIndicated: boolean;
}

const Body = ({
  nodeKey,
  onConfigChange,
  selected,
  config,
  staleIndicated,
}: BodyProps): ReactElement => {
  const resolve = Color.useResolve();
  const {
    label,
    fillColor,
    strokeColor,
    dimensions = Border.DEFAULT_DIMENSIONS,
    borderRadius,
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
      {staleIndicated && <Stale.Indicator />}
      <Tank
        strokeColor={resolve(strokeColor)}
        dimensions={dimensions}
        borderRadius={borderRadius}
        fillColor={resolve(fillColor)}
      />
    </Grid.Grid>
  );
};

// Filled draws the level of the tank's channel and reports when it goes stale.
const Filled = (props: NodeProps<schematic.TankNodeConfig>): ReactElement => {
  const { nodeKey, position, config } = props;
  const {
    channel,
    rollingAverage,
    borderRadius,
    caretVisible,
    scaleVisible,
    dimensions: dims = Border.DEFAULT_DIMENSIONS,
  } = config;
  const badged = Staleness.useBadged();
  const telem = Scale.source({ channel, rollingAverage });
  const vis = Scale.useVisProps({
    ...config,
    caretHidden: !caretVisible,
    scaleHidden: !scaleVisible,
  });
  const { stale } = VisScale.use({
    ...vis,
    telem,
    aetherKey: nodeKey,
    box: box.construct(xy.translate(position ?? xy.ZERO, OVERLAP), {
      width: dims.width - OVERLAP * 2,
      height: dims.height - OVERLAP * 2,
    }),
    direction: "y",
    externalScale: true,
    cornerRadii: cornerRadii(borderRadius, dims),
    staleColorHidden: badged,
  });
  return <Body {...props} staleIndicated={stale && badged} />;
};

export const Symbol = (props: NodeProps<schematic.TankNodeConfig>): ReactElement =>
  primitive.isNonZero(props.config.channel) ? (
    <Filled {...props} />
  ) : (
    <Body {...props} staleIndicated={false} />
  );

/** Renders the tank in the symbol list, with its colors resolved for the theme. */
export const Preview = ({
  strokeColor,
  fillColor,
  dimensions: _dimensions,
  ...rest
}: schematic.TankNodeConfig): ReactElement => {
  const resolve = Color.useResolve();
  return (
    <Tank {...rest} strokeColor={resolve(strokeColor)} fillColor={resolve(fillColor)} />
  );
};
