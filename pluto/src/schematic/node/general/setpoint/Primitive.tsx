// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/schematic/node/general/button/button.css";
import "@/schematic/node/general/setpoint/setpoint.css";

import { type CSSProperties, type ReactElement, useState } from "react";

import { Button as BaseButton } from "@/button";
import { CSS } from "@/css";
import { Input as BaseInput } from "@/input";
import { Handle } from "@/schematic/node/common/handle";
import { Primitive } from "@/schematic/node/common/primitive";
import { type Config } from "@/schematic/node/general/setpoint/config";
import { symbolColorVar } from "@/schematic/symbolColor";

interface RenderProps extends Omit<Config, "variant">, BaseInput.Control<number> {
  className?: string;
  style?: CSSProperties;
}

const SETPOINT_STYLE: CSSProperties = { zIndex: 5 };

export const Setpoint = ({
  orientation = "left",
  className,
  style,
  value,
  units,
  color,
  onChange,
  size = "small",
  disabled,
}: RenderProps): ReactElement => {
  const [currValue, setCurrValue] = useState(value);
  return (
    <Primitive.Div
      className={CSS(CSS.B("setpoint"), CSS.B("symbol-colored"), className)}
      orientation={orientation}
      style={{ ...style, [CSS.var("symbol-color")]: symbolColorVar(color) }}
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
          style={SETPOINT_STYLE}
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
        size={size}
        value={currValue}
        onChange={setCurrValue}
        showDragHandle={false}
        selectOnFocus
        endContent={units}
        borderWidth={1}
        disabled={disabled}
      >
        <BaseButton.Button
          size={size}
          variant="filled"
          className={CSS.B("symbol-button")}
          onClick={() => onChange(currValue)}
        >
          Set
        </BaseButton.Button>
      </BaseInput.Numeric>
    </Primitive.Div>
  );
};
