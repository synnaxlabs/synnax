// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { box, xy } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { CSS } from "@/css";
import { Grid } from "@/schematic/node/common/grid";
import { Label } from "@/schematic/node/common/label";
import { type Config, DEFAULT_DIMENSIONS } from "@/schematic/node/general/scale/config";
import { type NodeProps } from "@/schematic/node/spec";
import { Scale as BaseScale } from "@/vis/scale";

export const Symbol = ({
  nodeKey,
  position,
  onConfigChange,
  selected,
  config: { label, color, dimensions: dims = DEFAULT_DIMENSIONS, indicator },
}: NodeProps<Config>): ReactElement => {
  BaseScale.use({
    ...indicator,
    color,
    aetherKey: nodeKey,
    box: box.construct(position ?? xy.ZERO, dims),
    direction: "y",
  });
  return (
    <Grid.Grid
      editable={selected}
      nodeKey={nodeKey}
      allowRotate={false}
      onResize={(dimensions) => onConfigChange({ dimensions })}
    >
      <Label.Label config={label} onChange={onConfigChange} />
      <div style={dims} className={CSS.B("symbol-primitive")} />
    </Grid.Grid>
  );
};
