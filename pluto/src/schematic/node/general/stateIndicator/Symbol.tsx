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
import { type Config } from "@/schematic/node/general/stateIndicator/config";
import { Primitive } from "@/schematic/node/general/stateIndicator/Primitive";
import { type NodeProps } from "@/schematic/node/spec";
import { StateIndicator } from "@/vis/stateIndicator";

export const Symbol = ({
  nodeKey,
  onConfigChange,
  selected,
  draggable,
  config: { label, source, options, color: colorVal, inlineSize },
}: NodeProps<Config>): ReactElement => {
  const { key: optKey } = StateIndicator.use({ aetherKey: nodeKey, source, options });
  const gridItems: Grid.Item[] = [];
  const labelItem = Label.gridItem(label, onConfigChange);
  if (labelItem != null) gridItems.push(labelItem);
  return (
    <Grid.Grid
      items={gridItems}
      allowRotate={false}
      editable={selected && !draggable}
      symbolKey={nodeKey}
      onLocationChange={(key, loc) => {
        if (key !== "label") return;
        onConfigChange({ label: { ...label, orientation: loc } });
      }}
    >
      <Primitive
        matchedOptionKey={optKey}
        options={options}
        color={colorVal}
        inlineSize={inlineSize}
      />
    </Grid.Grid>
  );
};
