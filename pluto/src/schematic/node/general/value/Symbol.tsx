// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { box, scale, text, xy } from "@synnaxlabs/x";
import { type ReactElement, useMemo } from "react";

import { Grid } from "@/schematic/node/common/grid";
import { Label } from "@/schematic/node/common/label";
import { type Config } from "@/schematic/node/general/value/config";
import { Value } from "@/schematic/node/general/value/Primitive";
import { type NodeProps } from "@/schematic/node/spec";
import { telem } from "@/telem/aether";
import { Theming } from "@/theming";
import { Value as BaseValue } from "@/vis/value";

const VALUE_BACKGROUND_OVERSCAN = xy.construct(10, -3);
const VALUE_BACKGROUND_SHIFT = xy.construct(1, 1);

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
    telem: t,
    units,
    inlineSize = 70,
    notation,
    stalenessColor,
    stalenessTimeout,
    redline,
  },
}: NodeProps<Config>): ReactElement => {
  const font = Theming.useTypography(level);
  const valueBoxHeight = (font.lineHeight + 0.5) * font.baseSize + 2;
  const backgroundTelem = useMemo(() => {
    if (t == null || redline == null) return undefined;
    const { bounds, gradient } = redline;
    return telem.sourcePipeline("color", {
      connections: [
        { from: "source", to: "scale" },
        { from: "scale", to: "gradient" },
      ],
      segments: {
        source: t,
        scale: telem.scaleNumber({
          scale: scale.Scale.scale<number>(bounds).scale(0, 1).transform,
        }),
        gradient: telem.colorGradient({ gradient }),
      },
      outlet: "gradient",
    });
  }, [t, redline]);
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
        dimensions={{ height: valueBoxHeight, width: oWidth }}
        inlineSize={inlineSize}
        units={units}
        unitsLevel={text.downLevel(level)}
      />
    </Grid.Grid>
  );
};
