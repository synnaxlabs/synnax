// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type schematic } from "@synnaxlabs/client";
import { type ReactElement } from "react";

import { Grid } from "@/schematic/node/common/grid";
import { Label } from "@/schematic/node/common/label";
import { type NodeProps } from "@/schematic/node/spec";
import { Cylinder } from "@/schematic/node/vessels/cylinder/Primitive";

export const Symbol = ({
  nodeKey,
  onConfigChange,
  selected,
  config: {
    label,
    orientation = "left",
    backgroundColor,
    color,
    dimensions,
    borderRadius,
  },
}: NodeProps<schematic.NodeConfigCylinder>): ReactElement => (
  <Grid.Grid
    editable={selected}
    nodeKey={nodeKey}
    orientation={orientation}
    onRotate={onConfigChange}
    onResize={(dimensions) => onConfigChange({ dimensions })}
  >
    <Label.Label config={label} onChange={onConfigChange} />
    <Cylinder
      orientation={orientation}
      color={color}
      dimensions={dimensions}
      borderRadius={borderRadius}
      backgroundColor={backgroundColor}
    />
  </Grid.Grid>
);
