// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { Control } from "@/schematic/node/common/control";
import { Grid } from "@/schematic/node/common/grid";
import { Label } from "@/schematic/node/common/label";
import { type Config } from "@/schematic/node/general/setpoint/config";
import { Primitive } from "@/schematic/node/general/setpoint/Primitive";
import { type NodeProps } from "@/schematic/node/spec";
import { Setpoint } from "@/vis/setpoint";

export const Symbol = ({
  nodeKey,
  onConfigChange,
  selected,
  draggable,
  config: {
    label,
    orientation = "left",
    control,
    units,
    source,
    sink,
    color,
    size,
    disabled,
  },
}: NodeProps<Config>): ReactElement => {
  const { value, set } = Setpoint.use({ aetherKey: nodeKey, source, sink });
  return (
    <Grid.Grid nodeKey={nodeKey} allowRotate={false} editable={selected && !draggable}>
      <Control.State config={control} onChange={onConfigChange} />
      <Label.Label config={label} onChange={onConfigChange} />
      <Primitive
        value={value}
        onChange={set}
        units={units}
        color={color}
        orientation={orientation}
        disabled={disabled}
        size={size}
      />
    </Grid.Grid>
  );
};
