// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/schematic/node/general/input/input.css";

import { type schematic } from "@synnaxlabs/client";
import { type ReactElement, useState } from "react";

import { Button as BaseButton } from "@/button";
import { CSS } from "@/css";
import { Input as BaseInput } from "@/input";
import { Handle } from "@/schematic/node/common/handle";
import { Primitive } from "@/schematic/node/common/primitive";

interface PrimitiveProps extends Omit<schematic.NodeConfigInput, "variant"> {
  initialValue?: string;
  className?: string;
  onSend?: (value: string) => void;
}

export const Input = ({
  className,
  initialValue = "",
  orientation = "left",
  color,
  size,
  onSend,
  disabled,
}: PrimitiveProps): ReactElement => {
  const [value, setValue] = useState(initialValue);
  return (
    <Primitive.Div
      orientation={orientation}
      className={CSS(CSS.B("input-symbol"), className)}
    >
      <Handle.Rectangle
        orientation={orientation}
        left={0}
        top={0}
        right={100}
        bottom={100}
      />
      <BaseInput.Text
        value={value}
        onChange={setValue}
        size={size}
        borderWidth={1}
        disabled={disabled}
        color={color}
      >
        <BaseButton.Button
          size={size}
          variant="filled"
          onClick={() => onSend?.(value)}
          color={color}
        >
          Send
        </BaseButton.Button>
      </BaseInput.Text>
    </Primitive.Div>
  );
};
