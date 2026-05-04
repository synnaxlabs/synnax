// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { Grid, type GridItem } from "@/schematic/node/common/grid/Grid";
import { Label } from "@/schematic/node/common/label";
import { type NodeProps } from "@/schematic/node/common/symbol/factories";
import { type Config } from "@/schematic/node/general/stateIndicator/config";
import { Primitive } from "@/schematic/node/general/stateIndicator/Primitive";
import { StateIndicator as BaseStateIndicator } from "@/vis/stateIndicator";

export const Symbol = ({
  nodeKey: symbolKey,
  onConfigChange: onChange,
  selected,
  draggable,
  config: data,
}: NodeProps<Config>): ReactElement => {
  const { label, source, options, color: colorVal, inlineSize } = data;
  const { key: matchedOptionKey } = BaseStateIndicator.use({
    aetherKey: symbolKey,
    source,
    options,
  });

  const gridItems: GridItem[] = [];
  const labelItem = Label.gridItem(label, onChange);
  if (labelItem != null) gridItems.push(labelItem);

  return (
    <Grid
      items={gridItems}
      allowRotate={false}
      editable={selected && !draggable}
      symbolKey={symbolKey}
      onLocationChange={(key, loc) => {
        if (key !== "label") return;
        onChange({ label: { ...label, orientation: loc } });
      }}
    >
      <Primitive
        matchedOptionKey={matchedOptionKey}
        options={options}
        color={colorVal}
        inlineSize={inlineSize}
      />
    </Grid>
  );
};
