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
import { type Config } from "@/schematic/node/general/light/config";
import { Light } from "@/schematic/node/general/light/Primitive";
import { type NodeProps } from "@/schematic/node/spec";
import { Light as BaseLight } from "@/vis/light";

// Width the Light primitive renders at scale 1: base 64px x SVG BASE_SCALE (0.8).
const SCALE_1_WIDTH = 51.2;

export const Symbol = ({
  nodeKey,
  onConfigChange,
  selected,
  config: { label, source, orientation = "left", ...rest },
}: NodeProps<Config>): ReactElement => {
  const { enabled } = BaseLight.use({ aetherKey: nodeKey, source });
  return (
    <Grid.Grid
      orientation={orientation}
      onRotate={onConfigChange}
      editable={selected}
      nodeKey={nodeKey}
      keepAspectRatio
      onResize={({ width }) => onConfigChange({ scale: width / SCALE_1_WIDTH })}
    >
      <Label.Label config={label} onChange={onConfigChange} />
      <Light enabled={enabled} orientation={orientation} {...rest} />
    </Grid.Grid>
  );
};
