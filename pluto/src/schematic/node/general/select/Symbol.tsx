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
import { type Config } from "@/schematic/node/general/select/config";
import { Select } from "@/schematic/node/general/select/Primitive";
import { type NodeProps } from "@/schematic/node/spec";
import { Setpoint as BaseSetpoint } from "@/vis/setpoint";

export const Symbol = ({
  nodeKey,
  onConfigChange,
  selected,
  config: {
    label,
    orientation = "left",
    control,
    color,
    sink,
    options,
    size,
    disabled,
    inlineSize,
  },
}: NodeProps<Config>): ReactElement => {
  const { set } = BaseSetpoint.use({ aetherKey: nodeKey, sink });
  const [selectedKey, setSelectedKey] = useState<string | undefined>(undefined);
  const handleSelectionChange = (key: string | null): void =>
    setSelectedKey(key ?? undefined);
  return (
    <Grid.Grid
      nodeKey={nodeKey}
      allowRotate={false}
      editable={selected}
      resizeHandles={["left", "right"]}
      onResize={({ width }) => onConfigChange({ inlineSize: width })}
    >
      <Control.State config={control} onChange={onConfigChange} />
      <Label.Label config={label} onChange={onConfigChange} />
      <Select
        value={selectedKey}
        onChange={handleSelectionChange}
        onSend={set}
        color={color}
        orientation={orientation}
        disabled={disabled}
        options={options}
        size={size}
        inlineSize={inlineSize}
      />
    </Grid.Grid>
  );
};
