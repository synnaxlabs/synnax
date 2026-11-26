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

import { Channel } from "@/channel";
import { Color } from "@/color";
import { Deleted } from "@/schematic/node/common/deleted";
import { Grid } from "@/schematic/node/common/grid";
import { Indicator } from "@/schematic/node/common/indicator";
import { Label } from "@/schematic/node/common/label";
import { Options } from "@/schematic/node/common/options";
import { Telem } from "@/schematic/node/common/telem";
import { Primitive } from "@/schematic/node/general/stateIndicator/primitive";
import type { NodeProps } from "@/schematic/node/spec";
import { StateIndicator as BaseStateIndicator } from "@/vis/stateIndicator";

export const Symbol = ({
  nodeKey,
  onConfigChange,
  selected,
  config: {
    label,
    channel,
    enumeration,
    strokeColor,
    inlineSize,
    size,
    stalenessTimeout,
  },
}: NodeProps<schematic.StateIndicatorNodeConfig>): ReactElement => {
  const resolve = Color.useResolve();
  const options = Options.use(enumeration);
  const source = Telem.numberSource(channel);
  const { key: optKey, stale } = BaseStateIndicator.use({
    aetherKey: nodeKey,
    source,
    options,
    stalenessTimeout,
  });
  const deleted = Channel.useDeleted(channel);
  return (
    <Grid.Grid
      allowRotate={false}
      editable={selected}
      nodeKey={nodeKey}
      resizeHandles={["left", "right"]}
      onResize={({ width }) => onConfigChange({ inlineSize: width })}
    >
      <Label.Label config={label} onChange={onConfigChange} />
      <Indicator.Slot stale={stale} />
      {deleted && <Deleted.Item />}
      <Primitive.StateIndicator
        matchedOptionKey={optKey}
        options={options}
        strokeColor={resolve(strokeColor)}
        inlineSize={inlineSize}
        size={size}
      />
    </Grid.Grid>
  );
};
