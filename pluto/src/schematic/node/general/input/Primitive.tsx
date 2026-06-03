// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/schematic/node/general/input/input.css";

import { type ReactElement, useState } from "react";

import { Button as BaseButton } from "@/button";
import { CSS } from "@/css";
import { Input as BaseInput } from "@/input";
import { useColor } from "@/schematic/node/common/color";
import { Handle } from "@/schematic/node/common/handle";
import { Primitive } from "@/schematic/node/common/primitive";
import { type Config } from "@/schematic/node/general/input/config";

interface PrimitiveProps extends Omit<Config, "variant"> {
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
  const resolved = useColor(color);
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
        color={resolved}
      >
        <BaseButton.Button
          size={size}
          variant="filled"
          onClick={() => onSend?.(value)}
          color={resolved}
        >
          Send
        </BaseButton.Button>
      </BaseInput.Text>
    </Primitive.Div>
  );
};
