// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { forwardRef, FunctionComponent, Ref } from "react";

import clsx from "clsx";
import { useController, UseControllerProps } from "react-hook-form";

import { Input } from "./Input";
import { InputHelpText } from "./InputHelpText";
import { InputLabel } from "./InputLabel";
import { InputBaseProps, InputControlProps, InputValue } from "./types";

import {
  Space,
  SpaceAlignment,
  SpaceExtensionProps,
  SpaceJustification,
} from "@/core/Space";
import { camelToTitle } from "@/util/case";
import { Direction } from "@/spatial";

import "./InputItem.css";

interface RenderComponent<P> {
  render: FunctionComponent<P>;
}

interface InputItemExtensionProps<T extends InputValue, P extends InputControlProps<T>>
  extends SpaceExtensionProps {
  label?: string;
  helpText?: string;
  children?: FunctionComponent<P> | RenderComponent<P>;
  className?: string;
  style?: React.CSSProperties;
}

export type InputItemProps<T extends InputValue, P extends InputControlProps<T>> = P &
  InputItemExtensionProps<T, P>;

const CoreInputItem = <
  T extends InputValue = string | number,
  P extends InputControlProps<T> = InputBaseProps<T>
>(
  {
    label,
    helpText,
    children = Input as unknown as FunctionComponent<P>,
    direction = "vertical",
    size = "small",
    empty,
    style,
    className,
    justify,
    align,
    grow,
    ...props
  }: InputItemProps<T, P>,
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
  T extends InputValue = string | number,
  P extends InputControlProps<T> = InputBaseProps<T>
> = Omit<P, "onChange" | "value"> & InputItemExtensionProps<T, P> & UseControllerProps;

export const InputItem = forwardRef(CoreInputItem) as <
  T extends InputValue = string | number,
  P extends InputControlProps<T> = InputBaseProps<T>
>(
  props: InputItemProps<T, P> & { ref?: Ref<HTMLInputElement> }
) => JSX.Element;
// @ts-expect-error
InputItem.displayName = "InputItem";

export const InputItemControlled = <
  T extends InputValue = string | number,
  P extends InputControlProps<T> = InputBaseProps<T>
>({
  name,
  rules,
  control,
  shouldUnregister,
  defaultValue,
  label,
  ...props
}: InputItemControlledProps<T, P>): JSX.Element => {
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
      {...(props as unknown as Omit<InputItemProps<T, P>, "onChange" | "value">)}
    />
  );
};
