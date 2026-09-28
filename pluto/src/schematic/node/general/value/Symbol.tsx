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
import { type ReactElement, useMemo } from "react";

import { Grid } from "@/schematic/node/common/grid";
import { Label } from "@/schematic/node/common/label";
import { LEVEL_SIZES } from "@/schematic/node/common/size";
import { Value } from "@/schematic/node/general/value/Primitive";
import { type NodeProps } from "@/schematic/node/spec";
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
    color,
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
    backgroundColor,
  },
}: NodeProps<schematic.ValueNodeConfig>): ReactElement => {
  const valueBoxHeight = Component.HEIGHTS[LEVEL_SIZES[level]];
  const t = useMemo(
    () => BaseValue.stringSource({ channel, rollingAverage, precision, notation }),
    [channel, rollingAverage, precision, notation],
  );
  const backgroundTelem = useMemo(
    () => BaseValue.backgroundTelem(t, redline, backgroundColor),
    [t, redline, backgroundColor],
  );
  const { width: oWidth } = BaseValue.use({
    aetherKey: nodeKey,
    color: textColor,
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
    notation,
    useWidthForBackground: true,
    valueBackgroundOverScan: VALUE_BACKGROUND_OVERSCAN,
    valueBackgroundShift: VALUE_BACKGROUND_SHIFT,
  });

  return (
    <Grid.Grid editable={selected} nodeKey={nodeKey} allowRotate={false}>
      <Label.Label config={label} onChange={onConfigChange} />
      <Value
        color={color}
        orientation={orientation}
        dimensions={{ height: valueBoxHeight, width: oWidth }}
        inlineSize={inlineSize}
        units={units}
        unitsLevel={text.downLevel(level)}
      />
    </Grid.Grid>
  );
};
