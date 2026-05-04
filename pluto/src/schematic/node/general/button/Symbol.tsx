// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { location } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Control } from "@/schematic/node/common/control";
import { Grid } from "@/schematic/node/common/grid";
import { type Config } from "@/schematic/node/general/button/config";
import { Primitive } from "@/schematic/node/general/button/Primitive";
import { type NodeProps } from "@/schematic/node/spec";
import { Button as ButtonTelem } from "@/vis/button";

export const Symbol = ({
  nodeKey: symbolKey,
  selected,
  onConfigChange: onChange,
  config: data,
}: NodeProps<Config>): ReactElement => {
  const { label, orientation = "left", sink, control, mode, ...rest } = data;
  const { onMouseDown, onMouseUp } = ButtonTelem.use({
    aetherKey: symbolKey,
    sink,
    mode,
  });
  const gridItems: Grid.Item[] = [];
  const controlItem = Control.stateGridItem(control);
  if (controlItem != null) gridItems.push(controlItem);
  return (
    <Grid.Grid
      onRotate={() =>
        onChange({
          orientation: location.rotate(orientation, "clockwise"),
        } as Partial<Config>)
      }
      allowRotate={false}
      editable={selected}
      symbolKey={symbolKey}
      items={gridItems}
      onLocationChange={(key, loc) => {
        if (key !== "label") return;
        onChange({ label: { ...label, orientation: loc } });
      }}
    >
      <Primitive
        label={label?.label}
        onMouseDown={onMouseDown}
        onMouseUp={onMouseUp}
        orientation={orientation}
        {...rest}
      />
    </Grid.Grid>
  );
};
