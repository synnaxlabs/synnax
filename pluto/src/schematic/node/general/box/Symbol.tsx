// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/schematic/node/general/box/box.css";

import type { schematic } from "@synnaxlabs/client";
import type { ReactElement } from "react";

import { Color } from "@/color";
import { CSS } from "@/css";
import { Border } from "@/schematic/node/common/border";
import { Grid } from "@/schematic/node/common/grid";
import { Label } from "@/schematic/node/common/label";
import { Box } from "@/schematic/node/general/box/Primitive";
import type { NodeProps } from "@/schematic/node/spec";
import { Tank as TankPrimitive } from "@/schematic/node/vessels/tank/Primitive";

export const Symbol = ({
  nodeKey,
  onConfigChange,
  selected,
  config: {
    label,
    fillColor,
    borderRadius = Border.DEFAULT_BOX_RADIUS,
    strokeColor,
    dimensions,
    strokeWidth,
  },
}: NodeProps<schematic.BoxNodeConfig>): ReactElement => {
  const resolve = Color.useResolve();
  return (
    <Grid.Grid
      allowCenter
      allowRotate={false}
      editable={selected}
      nodeKey={nodeKey}
      onResize={(dimensions) => onConfigChange({ dimensions })}
    >
      <Label.Label config={label} onChange={onConfigChange} />
      <TankPrimitive
        className={CSS.B("box")}
        strokeColor={resolve(strokeColor)}
        dimensions={dimensions}
        boxBorderRadius={borderRadius}
        fillColor={resolve(fillColor)}
        strokeWidth={strokeWidth}
      />
      <svg className={CSS.B("box-frame")}>
        <rect width="100%" height="100%" rx={borderRadius} />
      </svg>
    </Grid.Grid>
  );
};

/** Renders the box in the symbol list, with its colors resolved for the theme. */
export const Preview = ({
  strokeColor,
  fillColor,
  dimensions: _dimensions,
  ...rest
}: schematic.BoxNodeConfig): ReactElement => {
  const resolve = Color.useResolve();
  return (
    <Box {...rest} strokeColor={resolve(strokeColor)} fillColor={resolve(fillColor)} />
  );
};
