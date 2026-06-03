// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { Grid } from "@/schematic/node/common/grid";
import { Label } from "@/schematic/node/common/label";
import { type NodeProps } from "@/schematic/node/spec";
import { type Config } from "@/schematic/node/vessels/tank/config";
import { Tank } from "@/schematic/node/vessels/tank/Primitive";

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
}: NodeProps<Config>): ReactElement => (
  <Grid.Grid
    allowCenter
    allowRotate={false}
    editable={selected}
    nodeKey={nodeKey}
    onResize={(dimensions) => onConfigChange({ dimensions })}
  >
    <Label.Label config={label} onChange={onConfigChange} />
    <Tank
      orientation={orientation}
      color={color}
      dimensions={dimensions}
      borderRadius={borderRadius}
      backgroundColor={backgroundColor}
    />
  </Grid.Grid>
);
