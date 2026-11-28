// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { schematic } from "@synnaxlabs/client";
import { Component } from "@synnaxlabs/lyra/component";
import { box, text, xy } from "@synnaxlabs/x";
import type { ReactElement } from "react";

import { Channel } from "@/channel";
import { Color } from "@/color";
import { Deleted } from "@/schematic/node/common/deleted";
import { Grid } from "@/schematic/node/common/grid";
import { Indicator } from "@/schematic/node/common/indicator";
import { Label } from "@/schematic/node/common/label";
import { size } from "@/schematic/node/common/size";
import { Primitive } from "@/schematic/node/general/value/primitive";
import type { NodeProps } from "@/schematic/node/spec";
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
    stalenessTimeout,
    redline,
    fillColor,
  },
}: NodeProps<schematic.ValueNodeConfig>): ReactElement => {
  const resolve = Color.useResolve();
  const valueBoxHeight = Component.HEIGHTS[size.BY_LEVEL[level]];
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
    stalenessTimeout,
    notation,
    useWidthForBackground: true,
    valueBackgroundOverScan: VALUE_BACKGROUND_OVERSCAN,
    valueBackgroundShift: VALUE_BACKGROUND_SHIFT,
  });
  const deleted = Channel.useDeleted(channel);
  // The text can widen the box past inlineSize, so a drag starts from the drawn width.
  const resize = Grid.useInlineResize(oWidth, (inlineSize) =>
    onConfigChange({ inlineSize }),
  );

  return (
    <Grid.Grid editable={selected} nodeKey={nodeKey} allowRotate={false} {...resize}>
      <Label.Label config={label} onChange={onConfigChange} />
      <Indicator.Slot stale={stale} />
      {deleted && <Deleted.Item />}
      <Primitive.Value
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
