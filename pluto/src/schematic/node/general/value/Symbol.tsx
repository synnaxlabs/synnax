// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type schematic } from "@synnaxlabs/client";
import { box, text, xy } from "@synnaxlabs/x";
import { type ReactElement, useMemo } from "react";

import { HEIGHTS } from "@/component/size";
import { Grid } from "@/schematic/node/common/grid";
import { Label } from "@/schematic/node/common/label";
import { LEVEL_SIZES } from "@/schematic/node/common/size";
import { BORDER_WIDTH, Value } from "@/schematic/node/general/value/Primitive";
import { type NodeProps } from "@/schematic/node/spec";
import { Value as BaseValue } from "@/vis/value";

export const Symbol = ({
  nodeKey,
  position,
  onConfigChange,
  selected,
  config: {
    label,
    level = "p",
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
  },
}: NodeProps<schematic.ValueNodeConfig>): ReactElement => {
  const valueBoxHeight = HEIGHTS[LEVEL_SIZES[level]];
  const t = useMemo(
    () => BaseValue.stringSource({ channel, rollingAverage, precision, notation }),
    [channel, rollingAverage, precision, notation],
  );
  const backgroundTelem = useMemo(
    () => BaseValue.backgroundTelem(t, redline),
    [t, redline],
  );
  BaseValue.use({
    aetherKey: nodeKey,
    level,
    box: box.construct(xy.translate(position ?? xy.ZERO, BORDER_WIDTH), {
      height: valueBoxHeight - BORDER_WIDTH * 2,
      width: inlineSize,
    }),
    telem: t,
    backgroundTelem,
    stalenessColor,
    stalenessTimeout,
  });

  return (
    <Grid.Grid editable={selected} nodeKey={nodeKey} allowRotate={false}>
      <Label.Label config={label} onChange={onConfigChange} />
      <Value
        color={color}
        orientation={orientation}
        height={valueBoxHeight}
        inlineSize={inlineSize}
        units={units}
        unitsLevel={text.downLevel(level)}
      />
    </Grid.Grid>
  );
};
