// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type schematic } from "@synnaxlabs/client";
import { type ReactElement, useMemo } from "react";

import { Grid } from "@/schematic/node/common/grid";
import { Label } from "@/schematic/node/common/label";
import * as CommonTelem from "@/schematic/node/common/telem";
import { Light, WIDTH_PER_SCALE } from "@/schematic/node/general/light/Primitive";
import { type NodeProps } from "@/schematic/node/spec";
import { Light as BaseLight } from "@/vis/light";

export const Symbol = ({
  nodeKey,
  onConfigChange,
  selected,
  config: { label, channel, threshold, orientation = "left", ...rest },
}: NodeProps<schematic.NodeConfigLight>): ReactElement => {
  const source = useMemo(
    () => CommonTelem.booleanSource(channel, threshold),
    [channel, threshold],
  );
  const { enabled } = BaseLight.use({ aetherKey: nodeKey, source });
  return (
    <Grid.Grid
      orientation={orientation}
      onRotate={onConfigChange}
      editable={selected}
      nodeKey={nodeKey}
      keepAspectRatio
      onResize={({ width }) => onConfigChange({ scale: width / WIDTH_PER_SCALE })}
    >
      <Label.Label config={label} onChange={onConfigChange} />
      <Light enabled={enabled} orientation={orientation} {...rest} />
    </Grid.Grid>
  );
};
