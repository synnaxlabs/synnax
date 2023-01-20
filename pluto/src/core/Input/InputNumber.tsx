import { forwardRef, useCallback } from "react";

import clsx from "clsx";
import { TiArrowUnsorted } from "react-icons/ti";

import { Button, ButtonProps } from "../Button";

import { useCursorDrag } from "@/hooks";

import { Pack } from "../Pack";

import { Box, Direction, dirToDim } from "@/spatial";
import { directionCls } from "@/util/css";

import "./InputNumber.css";

import { Input } from "./Input";
import { InputBaseProps } from "./types";

export interface InputNumberProps extends Omit<InputBaseProps<number | null>, "type"> {
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
        value={value ?? ""}
        onChange={(v: string | number) => {
          if (v === "") return onChange(null);
          onChange(Number(v));
        }}
        selectOnFocus={selectOnFocus}
        type="number"
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
  <Button.IconOnly
    className={clsx(
      "pluto-input-number__drag-button",
      directionCls(direction),
      className
    )}
    {...props}
  >
    <TiArrowUnsorted />
  </Button.IconOnly>
);
