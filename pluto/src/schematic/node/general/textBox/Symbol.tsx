// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { Grid } from "@/schematic/node/common/grid";
import { type Config } from "@/schematic/node/general/textBox/config";
import { Primitive } from "@/schematic/node/general/textBox/Primitive";
import { type NodeProps } from "@/schematic/node/spec";

export const Symbol = ({
  nodeKey,
  onConfigChange,
  config: { color, width, align, autoFit, level, value, orientation },
}: NodeProps<Config>): ReactElement => (
  <Primitive
    className={Grid.DRAG_HANDLE_CLASS}
    onChange={(value) => onConfigChange({ value })}
    value={value}
    level={level}
    color={color}
    key={nodeKey}
    width={width}
    align={align}
    autoFit={autoFit}
    orientation={orientation}
  />
);
