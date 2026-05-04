// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { Button as BaseButton } from "@/button";
import { CSS } from "@/css";
import { Input as BaseInput } from "@/input";
import { Div, Handle, HandleBoundary } from "@/schematic/node/common/symbol/primitives";
import { type Config } from "@/schematic/node/general/input/config";

interface RenderProps extends Config {
  className?: string;
  value: string;
  onChange: (value: string) => void;
  onSend?: (value: string) => void;
}

export const Primitive = ({
  className,
  orientation = "left",
  color: colorVal,
  value,
  onChange,
  size,
  onSend,
  disabled,
}: RenderProps): ReactElement => (
  <Div orientation={orientation} className={CSS(CSS.B("input-symbol"), className)}>
    <HandleBoundary orientation={orientation}>
      <Handle location="left" orientation={orientation} left={0} top={50} id="1" />
      <Handle location="right" orientation={orientation} left={100} top={50} id="2" />
      <Handle location="top" orientation={orientation} left={50} top={0} id="3" />
      <Handle location="bottom" orientation={orientation} left={50} top={100} id="4" />
    </HandleBoundary>
    <BaseInput.Text
      value={value}
      onChange={onChange}
      size={size}
      borderWidth={1}
      disabled={disabled}
      color={colorVal}
    >
      <BaseButton.Button
        size={size}
        variant="filled"
        onClick={() => onSend?.(value)}
        color={colorVal}
      >
        Send
      </BaseButton.Button>
    </BaseInput.Text>
  </Div>
);
