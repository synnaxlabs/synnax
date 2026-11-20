// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type schematic } from "@synnaxlabs/client";
import { Component } from "@synnaxlabs/lyra/component";
import { box, text, xy } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Color } from "@/color";
import { Grid } from "@/schematic/node/common/grid";
import { Label } from "@/schematic/node/common/label";
import { LEVEL_SIZES } from "@/schematic/node/common/size";
import { Stale } from "@/schematic/node/common/stale";
import { Value } from "@/schematic/node/general/value/Primitive";
import { type NodeProps } from "@/schematic/node/spec";
import { Staleness } from "@/vis/staleness";
import { Value as BaseValue } from "@/vis/value";

const VALUE_BACKGROUND_OVERSCAN = xy.construct(1, -4);
const VALUE_BACKGROUND_SHIFT = xy.construct(2, 2);

export const Symbol = ({
  nodeKey,
  position,
  onConfigChange,
  selected,
  config: {
    label,
    level = "p",
    textColor,
    strokeColor,
    channel,
    rollingAverage,
    precision,
    units,
    inlineSize = 70,
    orientation,
    notation,
    stalenessColor,
    stalenessTimeout,
    redline,
    fillColor,
  },
}: NodeProps<schematic.ValueNodeConfig>): ReactElement => {
  const badged = Staleness.useBadged();
  const resolve = Color.useResolve();
  const valueBoxHeight = Component.HEIGHTS[LEVEL_SIZES[level]];
  const t = BaseValue.stringSource({ channel, rollingAverage, precision, notation });
  const backgroundTelem = BaseValue.backgroundTelem(t, redline, resolve(fillColor));
  const { width: oWidth, stale } = BaseValue.use({
    aetherKey: nodeKey,
    color: resolve(textColor),
    level,
    box: box.construct(xy.translateY(position ?? xy.ZERO, 1), {
      height: valueBoxHeight,
      width: inlineSize,
    }),
    telem: t,
    backgroundTelem,
    minWidth: inlineSize,
    stalenessColor,
    stalenessTimeout,
    staleColorHidden: badged,
    notation,
    useWidthForBackground: true,
    valueBackgroundOverScan: VALUE_BACKGROUND_OVERSCAN,
    valueBackgroundShift: VALUE_BACKGROUND_SHIFT,
  });
  // The text can widen the box past inlineSize, so a drag starts from the drawn width.
  const resize = Grid.useInlineResize(oWidth, (inlineSize) =>
    onConfigChange({ inlineSize }),
  );

  return (
    <Grid.Grid editable={selected} nodeKey={nodeKey} allowRotate={false} {...resize}>
      <Label.Label config={label} onChange={onConfigChange} />
      {stale && badged && <Stale.Indicator />}
      <Value
        strokeColor={resolve(strokeColor)}
        orientation={orientation}
        dimensions={{ height: valueBoxHeight, width: oWidth }}
        inlineSize={inlineSize}
        units={units}
        unitsLevel={text.downLevel(level)}
      />
    </Grid.Grid>
  );
};
