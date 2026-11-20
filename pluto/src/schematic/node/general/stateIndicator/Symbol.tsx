// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { schematic } from "@synnaxlabs/client";
import { Theming } from "@synnaxlabs/lyra/theming";
import type { ReactElement } from "react";

import { Color } from "@/color";
import { Grid } from "@/schematic/node/common/grid";
import { Label } from "@/schematic/node/common/label";
import { Stale } from "@/schematic/node/common/stale";
import { Telem } from "@/schematic/node/common/telem";
import { StateIndicator } from "@/schematic/node/general/stateIndicator/Primitive";
import type { NodeProps } from "@/schematic/node/spec";
import { Staleness } from "@/vis/staleness";
import { StateIndicator as BaseStateIndicator } from "@/vis/stateIndicator";

export const Symbol = ({
  nodeKey,
  onConfigChange,
  selected,
  config: {
    label,
    channel,
    options,
    strokeColor,
    inlineSize,
    size,
    stalenessTimeout,
    stalenessColor,
  },
}: NodeProps<schematic.StateIndicatorNodeConfig>): ReactElement => {
  const resolve = Color.useResolve();
  const theme = Theming.use();
  const badged = Staleness.useBadged();
  const source = Telem.numberSource(channel);
  const { key: optKey, stale } = BaseStateIndicator.use({
    aetherKey: nodeKey,
    source,
    options,
    stalenessTimeout,
  });
  return (
    <Grid.Grid
      allowRotate={false}
      editable={selected}
      nodeKey={nodeKey}
      resizeHandles={["left", "right"]}
      onResize={({ width }) => onConfigChange({ inlineSize: width })}
    >
      <Label.Label config={label} onChange={onConfigChange} />
      {stale && badged && <Stale.Indicator />}
      <StateIndicator
        matchedOptionKey={optKey}
        options={options}
        strokeColor={resolve(strokeColor)}
        inlineSize={inlineSize}
        size={size}
        staleColor={
          stale && !badged ? Staleness.resolveColor(stalenessColor, theme) : undefined
        }
      />
    </Grid.Grid>
  );
};
