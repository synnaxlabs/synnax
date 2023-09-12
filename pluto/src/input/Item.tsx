// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { forwardRef, type ReactElement, type Ref } from "react";

import { direction, toArray } from "@synnaxlabs/x";
import {
  type UseControllerProps,
  useController,
  type FieldValues,
  type FieldPath,
} from "react-hook-form";

import { Align } from "@/align";
import { CSS } from "@/css";
import { HelpText } from "@/input/HelpText";
import { Label } from "@/input/Label";
import { Text } from "@/input/Text";
import { type BaseProps, type Control, type Value } from "@/input/types";
import { camelToTitle } from "@/util/case";
import { type RenderProp } from "@/util/renderProp";

import "@/input/Item.css";

interface RenderComponent<P extends Record<string, any>> {
  render: RenderProp<P>;
}

interface ItemExtensionProps<
  I extends Value,
  O extends Value = I,
  P extends Control<I, O> = Control<I, O>,
> extends Align.SpaceExtensionProps {
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

export type ItemProps<
  I extends Value,
  O extends Value = I,
  P extends Control<I, O> = Control<I, O>,
> = P & ItemExtensionProps<I, O, P>;

const CoreItem = <
  I extends Value = string | number,
  O extends Value = I,
  P extends Control<I, O> = BaseProps<I, O>,
>(
  {
    label,
    helpText,
    children = Text as unknown as RenderProp<P>,
    direction: direction_ = "y",
    size = "small",
    empty,
    style,
    className,
    justify,
    align,
    grow,
    showLabel = true,
    ...props
  }: ItemProps<I, O, P>,
  ref: Ref<HTMLInputElement>,
): ReactElement => {
  const children_ = toArray(children).map((c) =>
    typeof c === "object" ? c.render : c,
  );
  const dir = direction.construct(direction_);

  let content: ReactElement | null;
  if (children_.length === 1) {
    content = children_[0]({ ref, key: 0, ...(props as unknown as P) });
  } else {
    content = (
      <Align.Pack direction={dir}>
        {
          children_
            // Unlikely to change order here so we can use index as key
            .map((c, i) => c({ key: i, ...(props as unknown as P) }))
            .filter((c) => c != null) as ReactElement[]
        }
      </Align.Pack>
    );
  }

  return (
    <Align.Space
      justify={justify}
      align={maybeDefaultAlignment(align, dir)}
      empty={empty}
      grow={grow}
      size={size}
      className={CSS(CSS.B("input-item"), className)}
      direction={dir}
      style={style}
    >
      {showLabel && <Label>{label}</Label>}
      {content}
      <HelpText>{helpText}</HelpText>
    </Align.Space>
  );
};

const maybeDefaultAlignment = (
  align?: Align.Alignment,
  dir?: direction.Direction,
): Align.Alignment => {
  if (align != null) return align;
  return dir === "y" ?? false ? "stretch" : "center";
};

export type ItemControlledProps<
  I extends Value = string | number,
  O extends Value = I,
  P extends Control<I, O> = BaseProps<I, O>,
  F extends FieldValues = FieldValues,
  TName extends FieldPath<F> = FieldPath<F>,
> = Omit<P, "onChange" | "value"> &
  ItemExtensionProps<I, O, P> &
  UseControllerProps<F, TName>;

export const Item = forwardRef(CoreItem) as <
  I extends Value = string | number,
  O extends Value = I,
  P extends Control<I, O> = BaseProps<I, O>,
>(
  props: ItemProps<I, O, P> & { ref?: Ref<HTMLInputElement> },
) => ReactElement;
// @ts-expect-error
Item.displayName = "InputItem";

export const ItemControlled = <
  I extends Value = string | number,
  O extends Value = I,
  P extends Control<I, O> = BaseProps<I, O>,
  F extends FieldValues = FieldValues,
  TName extends FieldPath<F> = FieldPath<F>,
>({
  name,
  rules,
  control,
  shouldUnregister,
  defaultValue,
  label,
  ...props
}: ItemControlledProps<I, O, P, F, TName>): ReactElement => {
  const { field, fieldState } = useController<F, TName>({
    control,
    rules,
    name,
    shouldUnregister,
  });
  if (label == null) label = camelToTitle(name);
  return (
    // @ts-expect-error
    <Item<I, O>
      ref={field.ref}
      label={label}
      value={field.value}
      onChange={field.onChange as unknown as (value: O) => void}
      helpText={fieldState.error?.message}
      {...(props as unknown as Omit<ItemProps<I, P>, "onChange" | "value">)}
    />
  );
};
