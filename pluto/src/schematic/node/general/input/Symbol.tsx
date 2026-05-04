// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement, useState } from "react";

import { Control } from "@/schematic/node/common/control";
import { Grid } from "@/schematic/node/common/grid";
import { Label } from "@/schematic/node/common/label";
import { type Config } from "@/schematic/node/general/input/config";
import { Primitive } from "@/schematic/node/general/input/Primitive";
import { type NodeProps } from "@/schematic/node/spec";
import { Input as InputTelem } from "@/vis/input";

export const Symbol = ({
  nodeKey: symbolKey,
  onConfigChange: onChange,
  selected,
  draggable,
  config: data,
}: NodeProps<Config>): ReactElement => {
  const { label, orientation = "left", control, color, sink, size, disabled } = data;
  const { set } = InputTelem.use({ aetherKey: symbolKey, sink });
  const gridItems: Grid.Item[] = [];
  const controlItem = Control.stateGridItem(control);
  if (controlItem != null) gridItems.push(controlItem);
  const labelItem = Label.gridItem(label, onChange);
  if (labelItem != null) gridItems.push(labelItem);
  const [value, setValue] = useState("");
  return (
    <Grid.Grid
      symbolKey={symbolKey}
      allowRotate={false}
      editable={selected && !draggable}
      items={gridItems}
      onLocationChange={(key, loc) => {
        if (key !== "label") return;
        onChange({ label: { ...label, orientation: loc } });
      }}
    >
      <Primitive
        value={value}
        onChange={setValue}
        onSend={set}
        color={color}
        orientation={orientation}
        disabled={disabled}
        size={size}
      />
    </Grid.Grid>
  );
};
