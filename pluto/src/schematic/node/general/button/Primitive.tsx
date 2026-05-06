// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button as BaseButton } from "@synnaxlabs/charon";
import { type MouseEventHandler, type ReactElement } from "react";

import { Handle } from "@/schematic/node/common/handle";
import { Primitive as Base } from "@/schematic/node/common/primitive";
import { type Config } from "@/schematic/node/general/button/config";

interface RenderProps extends Omit<Config, "variant"> {
  className?: string;
  onClick?: MouseEventHandler<HTMLButtonElement>;
  onMouseDown?: MouseEventHandler<HTMLButtonElement>;
  onMouseUp?: MouseEventHandler<HTMLButtonElement>;
}

export const Primitive = ({
  onClick,
  onMouseDown,
  onMouseUp,
  orientation = "left",
  label,
  color,
  size,
  level,
  onClickDelay: delay,
}: RenderProps): ReactElement => (
  <Base.Div orientation={orientation}>
    <BaseButton.Button
      variant="filled"
      onClick={onClick}
      onMouseDown={onMouseDown}
      onMouseUp={onMouseUp}
      color={color}
      size={size}
      level={level}
      onClickDelay={delay}
    >
      {label?.label ?? ""}
    </BaseButton.Button>
    <Handle.Boundary orientation={orientation}>
      <Handle.Handle
        location="left"
        orientation={orientation}
        left={0}
        top={50}
        id="1"
      />
      <Handle.Handle
        location="right"
        orientation={orientation}
        left={100}
        top={50}
        id="2"
      />
      <Handle.Handle
        location="top"
        orientation={orientation}
        left={50}
        top={0}
        id="3"
      />
      <Handle.Handle
        location="bottom"
        orientation={orientation}
        left={50}
        top={100}
        id="4"
      />
    </Handle.Boundary>
  </Base.Div>
);
