// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { schematic } from "@synnaxlabs/client";
import type { ReactElement } from "react";

import { Color } from "@/color";
import { Grid } from "@/schematic/node/common/grid";
import { Label } from "@/schematic/node/common/label";
import { Primitive } from "@/schematic/node/common/primitive";
import { Polygon } from "@/schematic/node/general/polygon/Primitive";
import type { NodeProps } from "@/schematic/node/spec";

export const Symbol = ({
  nodeKey,
  onConfigChange,
  selected,
  config,
}: NodeProps<schematic.PolygonNodeConfig>): ReactElement => {
  const resolve = Color.useResolve();
  const { label, orientation = "left", strokeColor, fillColor, ...rest } = config;
  return (
    <Grid.Grid
      keepAspectRatio
      editable={selected}
      nodeKey={nodeKey}
      orientation={orientation}
      onRotate={onConfigChange}
      onResize={({ width }) =>
        onConfigChange({
          sideLength: Math.round(
            (width * Math.sin(Math.PI / config.numSides)) / Primitive.BASE_SCALE,
          ),
        })
      }
    >
      <Label.Label config={label} onChange={onConfigChange} />
      <Polygon
        orientation={orientation}
        strokeColor={resolve(strokeColor)}
        fillColor={resolve(fillColor)}
        {...rest}
      />
    </Grid.Grid>
  );
};

/** Renders the polygon in the symbol list, with its colors resolved for the theme. */
export const Preview = ({
  strokeColor,
  fillColor,
  ...rest
}: schematic.PolygonNodeConfig): ReactElement => {
  const resolve = Color.useResolve();
  return (
    <Polygon
      {...rest}
      strokeColor={resolve(strokeColor)}
      fillColor={resolve(fillColor)}
    />
  );
};
