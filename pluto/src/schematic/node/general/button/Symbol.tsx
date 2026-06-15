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
import * as CommonTelem from "@/schematic/node/common/telem";
import { Button } from "@/schematic/node/general/button/Primitive";
import { type NodeProps } from "@/schematic/node/spec";
import { Button as BaseButton } from "@/vis/button";

export const Symbol = ({
  nodeKey,
  selected,
  onConfigChange,
  config: { label, orientation = "left", commandChannel, control, mode, ...rest },
}: NodeProps<schematic.NodeConfigButton>): ReactElement => {
  const sink = useMemo(() => CommonTelem.booleanSink(commandChannel), [commandChannel]);
  const { onMouseDown, onMouseUp } = BaseButton.use({ aetherKey: nodeKey, sink, mode });
  return (
    <Grid.Grid
      orientation={orientation}
      onRotate={onConfigChange}
      allowRotate={false}
      editable={selected}
      nodeKey={nodeKey}
    >
      <Control.State
        config={control}
        channel={commandChannel}
        onChange={onConfigChange}
      />
      <Button
        label={label}
        onMouseDown={onMouseDown}
        onMouseUp={onMouseUp}
        orientation={orientation}
        {...rest}
      />
    </Grid.Grid>
  );
};
