// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { box, scale, xy } from "@synnaxlabs/x";
import { type ReactElement, useMemo } from "react";

import { Grid, type GridItem } from "@/schematic/node/common/grid/Grid";
import { Label } from "@/schematic/node/common/label";
import { type NodeProps } from "@/schematic/node/common/symbol/factories";
import { type Config } from "@/schematic/node/general/value/config";
import { Primitive } from "@/schematic/node/general/value/Primitive";
import { telem } from "@/telem/aether";
import { Text } from "@/text";
import { Theming } from "@/theming";
import { Value as BaseValue } from "@/vis/value";

const VALUE_BACKGROUND_OVERSCAN = xy.construct(10, -3);
const VALUE_BACKGROUND_SHIFT = xy.construct(1, 1);

export const Symbol = ({
  nodeKey: symbolKey,
  position,
  onConfigChange: onChange,
  selected,
  draggable,
  config: data,
}: NodeProps<Config>): ReactElement => {
  const {
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
  } = data;
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
    aetherKey: symbolKey,
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

  const gridItems: GridItem[] = [];
  const labelItem = Label.gridItem(label, onChange);
  if (labelItem != null) gridItems.push(labelItem);

  return (
    <Grid
      editable={selected && !draggable}
      symbolKey={symbolKey}
      items={gridItems}
      allowRotate={false}
      onLocationChange={(key, loc) => {
        if (key !== "label") return;
        onChange({ label: { ...label, orientation: loc } });
      }}
    >
      <Primitive
        color={color}
        dimensions={{ height: valueBoxHeight, width: oWidth }}
        inlineSize={inlineSize}
        units={units}
        unitsLevel={Text.downLevel(level)}
      />
    </Grid>
  );
};
