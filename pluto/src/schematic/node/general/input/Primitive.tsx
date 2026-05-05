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
import { Handle } from "@/schematic/node/common/handle";
import { Primitive as Base } from "@/schematic/node/common/primitive";
import { type Config } from "@/schematic/node/general/input/config";

interface PrimitiveProps extends Omit<Config, "variant"> {
  initialValue?: string;
  className?: string;
  onSend?: (value: string) => void;
}

export const Primitive = ({
  className,
  initialValue = "",
  orientation = "left",
  color: colorVal,
  size,
  onSend,
  disabled,
}: PrimitiveProps): ReactElement => {
  const [value, setValue] = useState(initialValue);
  return (
    <Base.Div
      orientation={orientation}
      className={CSS(CSS.B("input-symbol"), className)}
    >
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
      <BaseInput.Text
        value={value}
        onChange={setValue}
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
    </Base.Div>
  );
};
