// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type schematic } from "@synnaxlabs/client";
import { Theming } from "@synnaxlabs/lyra/theming";
import { type ReactElement } from "react";

import { Deleted } from "@/schematic/node/common/deleted";
import { Grid } from "@/schematic/node/common/grid";
import { Label } from "@/schematic/node/common/label";
import { Telem } from "@/schematic/node/common/telem";
import { Light, WIDTH_PER_SCALE } from "@/schematic/node/general/light/Primitive";
import { type NodeProps } from "@/schematic/node/spec";
import { Light as BaseLight } from "@/vis/light";
import { Staleness } from "@/vis/staleness";

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
    stalenessColor,
    strokeColor,
    onColor,
    ...rest
  },
}: NodeProps<schematic.LightNodeConfig>): ReactElement => {
  const theme = Theming.use();
  const badged = Staleness.useBadged();
  const source = Telem.booleanSource(channel, threshold);
  const { enabled, stale } = BaseLight.use({
    aetherKey: nodeKey,
    source,
    stalenessTimeout,
  });
  const deleted = Deleted.use([channel]);
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
      {deleted ? (
        <Deleted.Item />
      ) : (
        stale &&
        badged && (
          <Grid.Item itemKey="stale" location="bottom">
            <Staleness.Badge />
          </Grid.Item>
        )
      )}
      <Light
        enabled={enabled}
        orientation={orientation}
        strokeColor={
          stale && !badged ? Staleness.resolveColor(stalenessColor, theme) : strokeColor
        }
        onColor={stale && !badged ? undefined : onColor}
        {...rest}
      />
    </Grid.Grid>
  );
};
