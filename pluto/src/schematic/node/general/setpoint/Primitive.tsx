// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/schematic/node/general/setpoint/setpoint.css";

import { type schematic } from "@synnaxlabs/client";
import { Button as BaseButton } from "@synnaxlabs/lyra/button";
import { CSS } from "@synnaxlabs/lyra/css";
import { Input as BaseInput } from "@synnaxlabs/lyra/input";
import { type CSSProperties, type ReactElement, useRef, useState } from "react";

import { Handle } from "@/schematic/node/common/handle";
import { Primitive } from "@/schematic/node/common/primitive";

interface RenderProps
  extends
    Partial<Omit<schematic.SetpointNodeConfig, "variant" | "label" | "scale">>,
    Omit<BaseInput.Control<number>, "value"> {
  className?: string;
  style?: CSSProperties;
}

export const Setpoint = ({
  orientation = "left",
  className,
  style,
  units,
  color,
  onChange,
  size = "small",
  disabled,
  onClickDelay,
}: RenderProps): ReactElement => {
  const [currValue, setCurrValue] = useState(0);
  const inputRef = useRef<HTMLInputElement>(null);
  return (
    <Primitive.Div
      className={CSS.cls(CSS.B("setpoint"), className)}
      orientation={orientation}
      style={style}
    >
      <Handle.Boundary orientation={orientation}>
        <Handle.Handle
          location="left"
          orientation={orientation}
          left={0.5}
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
          top={-2}
          id="3"
        />
        <Handle.Handle
          location="bottom"
          orientation={orientation}
          left={50}
          top={102}
          id="4"
        />
      </Handle.Boundary>
      <BaseInput.Numeric
        ref={inputRef}
        size={size}
        value={currValue}
        onChange={setCurrValue}
        showDragHandle={false}
        selectOnFocus
        endContent={units}
        color={color}
        borderWidth={1}
        disabled={disabled}
      >
        <BaseButton.Button
          size={size}
          variant="filled"
          onClick={() => onChange(currValue)}
          onClickDelay={onClickDelay}
          color={color}
          // WebKit leaves the input focused on a button press, so the typed value would
          // never commit. Blurring commits it before the click or hold sends.
          onMouseDown={() => inputRef.current?.blur()}
        >
          Set
        </BaseButton.Button>
      </BaseInput.Numeric>
    </Primitive.Div>
  );
};
