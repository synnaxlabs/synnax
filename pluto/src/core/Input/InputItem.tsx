// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { forwardRef, Ref } from "react";

import clsx from "clsx";
import { UseControllerProps, useController } from "react-hook-form";

import {
  Space,
  SpaceAlignment,
  SpaceExtensionProps,
  SpaceJustification,
} from "@/core/Space";
import { Direction } from "@/spatial";
import { camelToTitle } from "@/util/case";

import { Input } from "./Input";

import { RenderProp } from "@/util/renderable";

import { InputHelpText } from "./InputHelpText";

import "./InputItem.css";

import { InputLabel } from "./InputLabel";
import { InputBaseProps, InputControlProps, InputValue } from "./types";

interface RenderComponent<P> {
  render: RenderProp<P>;
}

interface InputItemExtensionProps<
  I extends InputValue,
  O extends InputValue = I,
  P extends InputControlProps<I, O> = InputControlProps<I, O>
> extends SpaceExtensionProps {
  label?: string;
  helpText?: string;
  children?: RenderProp<P> | RenderComponent<P>;
  className?: string;
  style?: React.CSSProperties;
}

export type InputItemProps<
  I extends InputValue,
  O extends InputValue = I,
  P extends InputControlProps<I, O> = InputControlProps<I, O>
> = P & InputItemExtensionProps<I, O, P>;

const CoreInputItem = <
  I extends InputValue = string | number,
  O extends InputValue = I,
  P extends InputControlProps<I, O> = InputBaseProps<I, O>
>(
  {
    label,
    helpText,
    children = Input as unknown as RenderProp<P>,
    direction = "vertical",
    size = "small",
    empty,
    style,
    className,
    justify,
    align,
    grow,
    ...props
  }: InputItemProps<I, O, P>,
  ref: Ref<any>
): JSX.Element => {
  if (typeof children === "object") children = children.render;
  return (
    <Space
      justify={maybeDefaultJustify(justify, direction)}
      align={maybeDefaultAlignment(align, direction)}
      empty={empty}
      grow={grow}
      size={size}
      className={clsx("pluto-input-item", className)}
      direction={direction}
      style={style}
    >
      <InputLabel>{label}</InputLabel>
      {children({ ref, ...(props as unknown as P) })}
      <InputHelpText>{helpText}</InputHelpText>
    </Space>
  );
};

const maybeDefaultAlignment = (
  align?: SpaceAlignment,
  direction?: Direction
): SpaceAlignment => {
  if (align != null) return align;
  return direction === "vertical" ? "stretch" : "center";
};

const maybeDefaultJustify = (
  justify?: SpaceJustification,
  direction?: Direction
): SpaceJustification => {
  if (justify != null) return justify;
  return direction === "vertical" ? "start" : "center";
};

export type InputItemControlledProps<
  I extends InputValue = string | number,
  O extends InputValue = I,
  P extends InputControlProps<I, O> = InputBaseProps<I, O>
> = Omit<P, "onChange" | "value"> &
  InputItemExtensionProps<I, O, P> &
  UseControllerProps;

export const InputItem = forwardRef(CoreInputItem) as <
  I extends InputValue = string | number,
  O extends InputValue = I,
  P extends InputControlProps<I, O> = InputBaseProps<I, O>
>(
  props: InputItemProps<I, O, P> & { ref?: Ref<HTMLInputElement> }
) => JSX.Element;
// @ts-expect-error
InputItem.displayName = "InputItem";

export const InputItemControlled = <
  I extends InputValue = string | number,
  O extends InputValue = I,
  P extends InputControlProps<I, O> = InputBaseProps<I, O>
>({
  name,
  rules,
  control,
  shouldUnregister,
  defaultValue,
  label,
  ...props
}: InputItemControlledProps<I, O, P>): JSX.Element => {
  const { field, fieldState } = useController({
    control,
    rules,
    name,
    shouldUnregister,
  });
  if (label == null) label = camelToTitle(name);
  return (
    /** @ts-expect-error */
    <InputItem
      ref={field.ref}
      label={label}
      value={field.value}
      onChange={field.onChange}
      helpText={fieldState.error?.message}
      {...(props as unknown as Omit<InputItemProps<I, O, P>, "onChange" | "value">)}
    />
  );
};
