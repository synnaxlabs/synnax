// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type MouseEventHandler, type ReactElement } from "react";

import { Button as BaseButton } from "@/button";
import { Handle } from "@/schematic/node/common/handle";
import { Primitive } from "@/schematic/node/common/primitive";
import { type Config } from "@/schematic/node/general/button/config";

interface RenderProps extends Omit<Config, "variant"> {
  className?: string;
  onClick?: MouseEventHandler<HTMLButtonElement>;
  onMouseDown?: MouseEventHandler<HTMLButtonElement>;
  onMouseUp?: MouseEventHandler<HTMLButtonElement>;
}

export const Button = ({
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
  <Primitive.Div orientation={orientation}>
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
    <Handle.Rectangle
      orientation={orientation}
      left={0}
      top={0}
      right={100}
      bottom={100}
    />
  </Primitive.Div>
);
