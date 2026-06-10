// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement, useMemo, useState } from "react";

import { Control } from "@/schematic/node/common/control";
import { Grid } from "@/schematic/node/common/grid";
import { Label } from "@/schematic/node/common/label";
import * as CommonTelem from "@/schematic/node/common/telem";
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
    commandChannel,
    options,
    size,
    disabled,
    inlineSize,
  },
}: NodeProps<Config>): ReactElement => {
  const sink = useMemo(() => CommonTelem.numberSink(commandChannel), [commandChannel]);
  const { set } = BaseSetpoint.use({ aetherKey: nodeKey, sink });
  const [selectedKey, setSelectedKey] = useState<string | undefined>(undefined);
  const handleSelectionChange = (key: string | null): void =>
    setSelectedKey(key ?? undefined);
  return (
    <Grid.Grid nodeKey={nodeKey} allowRotate={false} editable={selected}>
      <Control.State
        config={control}
        channel={commandChannel}
        onChange={onConfigChange}
      />
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
