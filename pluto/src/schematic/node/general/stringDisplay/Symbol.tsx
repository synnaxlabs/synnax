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
import { StringDisplay } from "@/schematic/node/general/stringDisplay/Primitive";
import { type NodeProps } from "@/schematic/node/spec";
import { telem } from "@/telem/aether";
import { StringValue as BaseStringValue } from "@/vis/stringValue";

export const Symbol = ({
  nodeKey,
  onConfigChange,
  selected,
  config: {
    label,
    channel,
    stalenessTimeout,
    stalenessColor,
    color,
    textColor,
    level,
    inlineSize,
    orientation,
  },
}: NodeProps<schematic.StringDisplayNodeConfig>): ReactElement => {
  const source = useMemo(
    () => telem.streamChannelStringValue({ channel: channel ?? 0 }),
    [channel],
  );
  const { value, stale } = BaseStringValue.use({
    aetherKey: nodeKey,
    telem: source,
    stalenessTimeout,
  });
  return (
    <Grid.Grid editable={selected} nodeKey={nodeKey} allowRotate={false}>
      <Label.Label config={label} onChange={onConfigChange} />
      <StringDisplay
        color={color}
        textColor={textColor}
        stalenessColor={stalenessColor}
        level={level}
        inlineSize={inlineSize}
        orientation={orientation}
        value={value}
        stale={stale}
      />
    </Grid.Grid>
  );
};
