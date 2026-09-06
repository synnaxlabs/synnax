// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { CSS } from "@/css";
import { Icon } from "@/icon";
import { Grid } from "@/schematic/node/common/grid";
import { type Config } from "@/schematic/node/groupBox/config";
import { Primitive } from "@/schematic/node/groupBox/Primitive";
import { type NodeProps } from "@/schematic/node/spec";

export const Symbol = ({
  nodeKey,
  selected,
  draggable,
  onConfigChange,
  config: { members, locked },
}: NodeProps<Config>): ReactElement => (
  <Grid.Grid allowCenter allowRotate={false} editable={selected} nodeKey={nodeKey}>
    <Primitive nodeKey={nodeKey} members={members} />
    {/* A nested group gets no chip. A locked box is undraggable but must keep
    its chip so it can be unlocked. */}
    {selected && (locked === true || draggable !== false) && (
      <button
        className={CSS.cls("nodrag", CSS.BE("group-box", "lock"))}
        onClick={() => onConfigChange({ locked: locked !== true })}
      >
        {locked === true ? <Icon.Lock /> : <Icon.Unlock />}
      </button>
    )}
  </Grid.Grid>
);
