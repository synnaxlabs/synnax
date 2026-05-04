// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement, useState } from "react";

import { Grid, type GridItem } from "@/schematic/node/common/grid/Grid";
import { Label } from "@/schematic/node/common/label";
import {
  controlStateGridItem,
  type NodeProps,
} from "@/schematic/node/common/symbol/factories";
import { type Config } from "@/schematic/node/general/select/config";
import { Primitive } from "@/schematic/node/general/select/Primitive";
import { Setpoint as BaseSetpoint } from "@/vis/setpoint";

export const Symbol = ({
  nodeKey: symbolKey,
  onConfigChange: onChange,
  selected,
  draggable,
  config: data,
}: NodeProps<Config>): ReactElement => {
  const {
    label,
    orientation = "left",
    control,
    color: colorVal,
    sink,
    options,
    size,
    disabled,
    inlineSize,
  } = data;
  const { set } = BaseSetpoint.use({ aetherKey: symbolKey, sink });
  const [selectedKey, setSelectedKey] = useState<string | undefined>(undefined);
  const handleSelectionChange = (key: string | null): void =>
    setSelectedKey(key ?? undefined);

  const gridItems: GridItem[] = [];
  const controlItem = controlStateGridItem(control);
  if (controlItem != null) gridItems.push(controlItem);
  const labelItem = Label.gridItem(label, onChange);
  if (labelItem != null) gridItems.push(labelItem);

  return (
    <Grid
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
        value={selectedKey}
        onChange={handleSelectionChange}
        onSend={set}
        color={colorVal}
        orientation={orientation}
        disabled={disabled}
        options={options}
        size={size}
        inlineSize={inlineSize}
      />
    </Grid>
  );
};
