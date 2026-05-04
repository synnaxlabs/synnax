// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { Grid, type GridItem } from "@/schematic/node/common/grid/Grid";
import { Label } from "@/schematic/node/common/label";
import { type NodeProps } from "@/schematic/node/common/symbol/factories";
import { type Config } from "@/schematic/node/general/box/config";
import { Primitive as TankPrimitive } from "@/schematic/node/vessels/tank/Primitive";

export const Symbol = ({
  nodeKey: symbolKey,
  onConfigChange: onChange,
  selected,
  config: data,
}: NodeProps<Config>): ReactElement => {
  const {
    label,
    orientation = "left",
    backgroundColor,
    borderRadius,
    color,
    dimensions,
    strokeWidth,
  } = data;
  const gridItems: GridItem[] = [];
  const labelItem = Label.gridItem(label, onChange);
  if (labelItem != null) gridItems.push(labelItem);
  return (
    <Grid
      allowCenter
      allowRotate={false}
      items={gridItems}
      editable={selected}
      symbolKey={symbolKey}
      onLocationChange={(key, loc) => {
        if (key === "label") onChange({ label: { ...label, orientation: loc } });
      }}
    >
      <TankPrimitive
        onResize={(dims) => onChange({ dimensions: dims })}
        orientation={orientation}
        color={color}
        dimensions={dimensions}
        boxBorderRadius={borderRadius}
        backgroundColor={backgroundColor}
        strokeWidth={strokeWidth}
      />
    </Grid>
  );
};
