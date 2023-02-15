// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { forwardRef, useCallback } from "react";

import clsx from "clsx";
import { TiArrowUnsorted } from "react-icons/ti";

import { Button, ButtonProps } from "../Button";

import { Box, Direction, dirToDim } from "@/spatial";

import { Pack } from "../Pack";

import { useCursorDrag } from "@/spatial/useCursorDrag";
import { directionCls } from "@/util/css";

import "./InputNumber.css";

import { Input } from "./Input";
import { InputBaseProps } from "./types";

export interface InputNumberProps extends Omit<InputBaseProps<number>, "type"> {
  showDragHandle?: boolean;
  dragDirection?: Direction;
  dragScale?: number;
  selectOnFocus?: boolean;
}

export const InputNumber = forwardRef<HTMLInputElement, InputNumberProps>(
  (
    {
      size = "medium",
      onChange,
      value,
      dragDirection = "x",
      showDragHandle = true,
      dragScale = 1,
      selectOnFocus = true,
      ...props
    },
    ref
  ): JSX.Element => {
    const onStart = useCursorDrag({
      onMove: useCallback(
        (box: Box) => onChange?.(box[dirToDim(dragDirection)] * dragScale),
        [onChange, dragDirection, dragScale]
      ),
    });
    const input = (
      <Input
        ref={ref}
        type="number"
        value={String(value) ?? ""}
        onChange={(v: string) => {
          if (v === "") return onChange(NaN);
          onChange(Number(v));
        }}
        selectOnFocus={selectOnFocus}
        {...props}
      />
    );

    if (!showDragHandle) return input;
    return (
      <Pack>
        {input}
        <DragButton direction={dragDirection} onMouseDown={onStart} />
      </Pack>
    );
  }
);
InputNumber.displayName = "InputNumber";

export interface DragButtonProps extends ButtonProps {
  direction: Direction;
}

export const DragButton = ({
  direction,
  className,
  ...props
}: DragButtonProps): JSX.Element => (
  <Button.Icon
    variant="outlined"
    className={clsx(
      "pluto-input-number__drag-button",
      directionCls(direction),
      className
    )}
    {...props}
  >
    <TiArrowUnsorted />
  </Button.Icon>
);
