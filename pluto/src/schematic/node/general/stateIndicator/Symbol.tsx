// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement, useMemo } from "react";

import { Grid } from "@/schematic/node/common/grid";
import { Label } from "@/schematic/node/common/label";
import * as CommonTelem from "@/schematic/node/common/telem";
import { type Config } from "@/schematic/node/general/stateIndicator/config";
import { StateIndicator } from "@/schematic/node/general/stateIndicator/Primitive";
import { type NodeProps } from "@/schematic/node/spec";
import { StateIndicator as BaseStateIndicator } from "@/vis/stateIndicator";

export const Symbol = ({
  nodeKey,
  onConfigChange,
  selected,
  config: { label, channel, options, color, inlineSize },
}: NodeProps<Config>): ReactElement => {
  const source = useMemo(() => CommonTelem.numberSource(channel), [channel]);
  const { key: optKey } = BaseStateIndicator.use({
    aetherKey: nodeKey,
    source,
    options,
  });
  return (
    <Grid.Grid allowRotate={false} editable={selected} nodeKey={nodeKey}>
      <Label.Label config={label} onChange={onConfigChange} />
      <StateIndicator
        matchedOptionKey={optKey}
        options={options}
        color={color}
        inlineSize={inlineSize}
      />
    </Grid.Grid>
  );
};
