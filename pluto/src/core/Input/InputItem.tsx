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
import {
  UseControllerProps,
  useController,
  FieldValues,
  FieldPath,
} from "react-hook-form";

import { Space, SpaceAlignment, SpaceExtensionProps } from "@/core/Space";
import { Direction } from "@/spatial";
import { camelToTitle } from "@/util/case";

import { Input } from "./Input";

import { RenderProp } from "@/util/renderProp";

import { InputHelpText } from "./InputHelpText";

import "./InputItem.css";

import { InputLabel } from "./InputLabel";
import { InputBaseProps, InputControl, InputValue } from "./types";

interface RenderComponent<P> {
  render: RenderProp<P>;
}

interface InputItemExtensionProps<
  V extends InputValue,
  P extends InputControl<V> = InputControl<V>
> extends SpaceExtensionProps {
  label?: string;
  helpText?: string;
  children?: RenderProp<P> | RenderComponent<P>;
  className?: string;
  style?: React.CSSProperties;
}

export type InputItemProps<
  V extends InputValue,
  P extends InputControl<V> = InputControl<V>
> = P & InputItemExtensionProps<V, P>;

const CoreInputItem = <
  V extends InputValue = string | number,
  P extends InputControl<V> = InputBaseProps<V>
>(
  {
    label,
    helpText,
    children = Input as unknown as RenderProp<P>,
    direction = "y",
    size = "small",
    empty,
    style,
    className,
    justify,
    align,
    grow,
    ...props
  }: InputItemProps<V, P>,
  ref: Ref<any>
): JSX.Element => {
  if (typeof children === "object") children = children.render;
  return (
    <Space
      justify={justify}
      align={maybeDefaultAlignment(align, direction)}
      empty={empty}
      grow={grow}
      size={size}
      className={clsx("pluto-input-item", className)}
      direction={direction}
      style={style}
    >
      <InputLabel>{label}</InputLabel>
      {children({ ref, grow, ...(props as unknown as P) })}
      <InputHelpText>{helpText}</InputHelpText>
    </Space>
  );
};

const maybeDefaultAlignment = (
  align?: SpaceAlignment,
  direction?: Direction
): SpaceAlignment => {
  if (align != null) return align;
  return direction === "y" ? "stretch" : "center";
};

export type InputItemControlledProps<
  V extends InputValue = string | number,
  P extends InputControl<V> = InputBaseProps<V>,
  F extends FieldValues = FieldValues,
  TName extends FieldPath<F> = FieldPath<F>
> = Omit<P, "onChange" | "value"> &
  InputItemExtensionProps<V, P> &
  UseControllerProps<F, TName>;

export const InputItem = forwardRef(CoreInputItem) as <
  V extends InputValue = string | number,
  P extends InputControl<V> = InputBaseProps<V>
>(
  props: InputItemProps<V, P> & { ref?: Ref<HTMLInputElement> }
) => JSX.Element;
// @ts-expect-error
InputItem.displayName = "InputItem";

export const InputItemControlled = <
  V extends InputValue = string | number,
  P extends InputControl<V> = InputBaseProps<V>,
  F extends FieldValues = FieldValues,
  TName extends FieldPath<F> = FieldPath<F>
>({
  name,
  rules,
  control,
  shouldUnregister,
  defaultValue,
  label,
  ...props
}: InputItemControlledProps<V, P, F, TName>): JSX.Element => {
  const { field, fieldState } = useController<F, TName>({
    control,
    rules,
    name,
    shouldUnregister,
  });
  if (label == null) label = camelToTitle(name);
  return (
    // @ts-expect-error
    <InputItem<V, P>
      ref={field.ref}
      label={label}
      value={field.value}
      onChange={field.onChange}
      helpText={fieldState.error?.message}
      {...(props as unknown as Omit<InputItemProps<V, P>, "onChange" | "value">)}
    />
  );
};
