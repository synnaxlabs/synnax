// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { DRAG_HANDLE_CLASS } from "@/schematic/node/common/grid/Grid";
import { type NodeProps } from "@/schematic/node/common/symbol/factories";
import { type Config } from "@/schematic/node/general/textBox/config";
import { Primitive } from "@/schematic/node/general/textBox/Primitive";

export const Symbol = ({
  nodeKey: symbolKey,
  onConfigChange: onChange,
  config: data,
}: NodeProps<Config>): ReactElement => {
  const { color, width, align, autoFit, level, value, orientation } = data;
  return (
    <Primitive
      className={DRAG_HANDLE_CLASS}
      onChange={(v) => onChange({ value: v })}
      value={value}
      level={level}
      color={color}
      key={symbolKey}
      width={width}
      align={align}
      autoFit={autoFit}
      orientation={orientation}
    />
  );
};
