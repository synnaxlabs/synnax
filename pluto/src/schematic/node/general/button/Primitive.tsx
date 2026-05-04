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
import { Div, Handle, HandleBoundary } from "@/schematic/node/common/symbol/primitives";
import { type Config } from "@/schematic/node/general/button/config";

interface RenderProps extends Omit<Config, "label"> {
  label?: string;
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
  label = "",
  color,
  size,
  level,
  onClickDelay: delay,
}: RenderProps): ReactElement => (
  <Div orientation={orientation}>
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
      {label}
    </BaseButton.Button>
    <HandleBoundary orientation={orientation}>
      <Handle location="left" orientation={orientation} left={0} top={50} id="1" />
      <Handle location="right" orientation={orientation} left={100} top={50} id="2" />
      <Handle location="top" orientation={orientation} left={50} top={0} id="3" />
      <Handle location="bottom" orientation={orientation} left={50} top={100} id="4" />
    </HandleBoundary>
  </Div>
);
