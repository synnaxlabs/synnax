// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { direction } from "@synnaxlabs/x";
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
import { type Control, type Value } from "@/input/types";
import { camelToTitle } from "@/util/case";

import { type RenderProp } from "..";

import "@/input/Item.css";

interface ItemExtensionProps extends Align.SpaceExtensionProps {
  label?: string;
  showLabel?: boolean;
  helpText?: string;
  className?: string;
  style?: React.CSSProperties;
}

export interface ItemProps extends Align.SpaceProps {
  label?: string;
  showLabel?: boolean;
  helpText?: string;
}

const maybeDefaultAlignment = (
  align?: Align.Alignment,
  dir: direction.Crude = "x",
): Align.Alignment => {
  if (align != null) return align;
  return direction.construct(dir) === "y" ? "stretch" : "center";
};

export const Item = ({
  label,
  showLabel = true,
  helpText,
  direction = "y",
  className,
  children,
  align,
  size = "small",
  ...props
}: ItemProps): ReactElement => {
  let inputAndHelp: ReactElement;
  if (direction === "x")
    inputAndHelp = (
      <Align.Space direction="y" size="small">
        {children}
        {helpText != null && <HelpText>{helpText}</HelpText>}
      </Align.Space>
    );
  else
    inputAndHelp = (
      <>
        {children}
        {helpText != null && <HelpText>{helpText}</HelpText>}
      </>
    );

  return (
    <Align.Space
      className={CSS(CSS.B("input-item"), className)}
      direction={direction}
      size={size}
      align={maybeDefaultAlignment(align, direction)}
      {...props}
    >
      {showLabel && <Label>{label}</Label>}
      {inputAndHelp}
    </Align.Space>
  );
};

export type ItemControlledProps<
  I extends Value = string | number,
  O extends Value = I,
  F extends FieldValues = FieldValues,
  TName extends FieldPath<F> = FieldPath<F>,
> = ItemExtensionProps &
  UseControllerProps<F, TName> & {
    children: RenderProp<Control<I, O>>;
  };

export const ItemControlled = <
  I extends Value = string | number,
  O extends Value = I,
  F extends FieldValues = FieldValues,
  TName extends FieldPath<F> = FieldPath<F>,
>({
  name,
  rules,
  control,
  shouldUnregister,
  defaultValue,
  label,
  children,
  ...props
}: ItemControlledProps<I, O, F, TName>): ReactElement => {
  const { field, fieldState } = useController<F, TName>({
    control,
    rules,
    name,
    shouldUnregister,
  });
  if (label == null) label = camelToTitle(name);
  return (
    <Item ref={field.ref} label={label} helpText={fieldState.error?.message} {...props}>
      {children({ value: field.value, onChange: field.onChange })}
    </Item>
  );
};
