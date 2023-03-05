// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { forwardRef, Ref } from "react";

import { Direction } from "@synnaxlabs/x";
import {
  UseControllerProps,
  useController,
  FieldValues,
  FieldPath,
} from "react-hook-form";

import { Input } from "./Input";
import { InputHelpText } from "./InputHelpText";
import { InputLabel } from "./InputLabel";
import { InputBaseProps, InputControl, InputValue } from "./types";

import { Pack } from "@/core/Pack";
import { Space, SpaceAlignment, SpaceExtensionProps } from "@/core/Space";
import { CSS } from "@/css";
import { camelToTitle } from "@/util/case";
import { RenderProp } from "@/util/renderProp";
import { toArray } from "@/util/toArray";

import "./InputItem.css";

interface RenderComponent<P> {
  render: RenderProp<P>;
}

interface InputItemExtensionProps<
  I extends InputValue,
  O extends InputValue = I,
  P extends InputControl<I, O> = InputControl<I, O>
> extends SpaceExtensionProps {
  label?: string;
  showLabel?: boolean;
  helpText?: string;
  children?:
    | RenderProp<P>
    | RenderComponent<P>
    | Array<RenderProp<P> | RenderComponent<P>>;
  className?: string;
  style?: React.CSSProperties;
}

export type InputItemProps<
  I extends InputValue,
  O extends InputValue = I,
  P extends InputControl<I, O> = InputControl<I, O>
> = P & InputItemExtensionProps<I, O, P>;

const CoreInputItem = <
  I extends InputValue = string | number,
  O extends InputValue = I,
  P extends InputControl<I, O> = InputBaseProps<I, O>
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
    showLabel = true,
    ...props
  }: InputItemProps<I, O, P>,
  ref: Ref<HTMLInputElement>
): JSX.Element => {
  const children_ = toArray(children).map((c) =>
    typeof c === "object" ? c.render : c
  );

  let content: JSX.Element | null;
  if (children_.length === 1) {
    content = children_[0]({ ref, grow, ...(props as unknown as P) });
  } else {
    content = (
      <Pack direction={direction}>
        {
          children_
            .map((c) => c({ ref, grow, ...(props as unknown as P) }))
            .filter((c) => c != null) as JSX.Element[]
        }
      </Pack>
    );
  }

  return (
    <Space
      justify={justify}
      align={maybeDefaultAlignment(align, direction)}
      empty={empty}
      grow={grow}
      size={size}
      className={CSS(CSS.B("input-item"), className)}
      direction={direction}
      style={style}
    >
      {showLabel && <InputLabel>{label}</InputLabel>}
      {content}
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
  I extends InputValue = string | number,
  O extends InputValue = I,
  P extends InputControl<I, O> = InputBaseProps<I, O>,
  F extends FieldValues = FieldValues,
  TName extends FieldPath<F> = FieldPath<F>
> = Omit<P, "onChange" | "value"> &
  InputItemExtensionProps<I, O, P> &
  UseControllerProps<F, TName>;

export const InputItem = forwardRef(CoreInputItem) as <
  I extends InputValue = string | number,
  O extends InputValue = I,
  P extends InputControl<I, O> = InputBaseProps<I, O>
>(
  props: InputItemProps<I, O, P> & { ref?: Ref<HTMLInputElement> }
) => JSX.Element;
// @ts-expect-error
InputItem.displayName = "InputItem";

export const InputItemControlled = <
  I extends InputValue = string | number,
  O extends InputValue = I,
  P extends InputControl<I, O> = InputBaseProps<I, O>,
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
}: InputItemControlledProps<I, O, P, F, TName>): JSX.Element => {
  const { field, fieldState } = useController<F, TName>({
    control,
    rules,
    name,
    shouldUnregister,
  });
  if (label == null) label = camelToTitle(name);
  return (
    // @ts-expect-error
    <InputItem<I, P>
      ref={field.ref}
      label={label}
      value={field.value}
      onChange={field.onChange}
      helpText={fieldState.error?.message}
      {...(props as unknown as Omit<InputItemProps<I, P>, "onChange" | "value">)}
    />
  );
};
