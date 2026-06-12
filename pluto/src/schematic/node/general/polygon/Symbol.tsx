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
import { Primitive } from "@/schematic/node/common/primitive";
import { type Config } from "@/schematic/node/general/polygon/config";
import { Polygon } from "@/schematic/node/general/polygon/Primitive";
import { type NodeProps } from "@/schematic/node/spec";

export const Symbol = ({
  nodeKey,
  onConfigChange,
  selected,
  config,
}: NodeProps<Config>): ReactElement => {
  const { label, orientation = "left", ...rest } = config;
  return (
    <Grid.Grid
      keepAspectRatio
      editable={selected}
      nodeKey={nodeKey}
      orientation={orientation}
      onRotate={onConfigChange}
      onResize={({ width }) =>
        onConfigChange({
          sideLength:
            (width * Math.sin(Math.PI / config.numSides)) / Primitive.BASE_SCALE,
        })
      }
    >
      <Label.Label config={label} onChange={onConfigChange} />
      <Polygon orientation={orientation} {...rest} />
    </Grid.Grid>
  );
};
