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
  config: { members },
}: NodeProps<Config>): ReactElement => (
  <Grid.Grid allowCenter allowRotate={false} editable={selected} nodeKey={nodeKey}>
    <Primitive nodeKey={nodeKey} members={members} />
    {/* Visual anchor only: the whole box drags. A nested group is locked to its
    parent, so it gets no anchor. */}
    {selected && draggable !== false && (
      <div className={CSS.BE("group-box", "move")}>
        <Icon.Pan />
      </div>
    )}
  </Grid.Grid>
);
