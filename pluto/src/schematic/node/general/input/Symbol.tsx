// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type schematic } from "@synnaxlabs/client";
import { type ReactElement, useMemo } from "react";

import { Control } from "@/schematic/node/common/control";
import { Grid } from "@/schematic/node/common/grid";
import { Label } from "@/schematic/node/common/label";
import * as CommonTelem from "@/schematic/node/common/telem";
import { Input } from "@/schematic/node/general/input/Primitive";
import { type NodeProps } from "@/schematic/node/spec";
import { Input as InputTelem } from "@/vis/input";

export const Symbol = ({
  nodeKey,
  onConfigChange,
  selected,
  config: { label, control, commandChannel, ...rest },
}: NodeProps<schematic.NodeConfigInput>): ReactElement => {
  const sink = useMemo(() => CommonTelem.stringSink(commandChannel), [commandChannel]);
  const { set } = InputTelem.use({ aetherKey: nodeKey, sink });
  return (
    <Grid.Grid nodeKey={nodeKey} allowRotate={false} editable={selected}>
      <Control.State
        config={control}
        channel={commandChannel}
        onChange={onConfigChange}
      />
      <Label.Label config={label} onChange={onConfigChange} />
      <Input onSend={set} {...rest} />
    </Grid.Grid>
  );
};
