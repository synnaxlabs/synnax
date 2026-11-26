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

import { Grid } from "@/schematic/node/common/grid";
import { Indicator } from "@/schematic/node/common/indicator";
import { Label } from "@/schematic/node/common/label";
import { Telem } from "@/schematic/node/common/telem";
import { Primitive } from "@/schematic/node/general/light/primitive";
import type { NodeProps } from "@/schematic/node/spec";
import { Light as BaseLight } from "@/vis/light";

export const Symbol = ({
  nodeKey,
  onConfigChange,
  selected,
  config: {
    label,
    channel,
    threshold,
    orientation = "left",
    stalenessTimeout,
    ...rest
  },
}: NodeProps<schematic.LightNodeConfig>): ReactElement => {
  const source = Telem.booleanSource(channel, threshold);
  const { enabled, stale } = BaseLight.use({
    aetherKey: nodeKey,
    source,
    stalenessTimeout,
  });
  return (
    <Grid.Grid
      orientation={orientation}
      onRotate={onConfigChange}
      editable={selected}
      nodeKey={nodeKey}
      keepAspectRatio
      onResize={({ width }) =>
        onConfigChange({ scale: width / Primitive.WIDTH_PER_SCALE })
      }
    >
      <Label.Label config={label} onChange={onConfigChange} />
      <Indicator.Slot stale={stale} />
      <Primitive.Light enabled={enabled} orientation={orientation} {...rest} />
    </Grid.Grid>
  );
};
