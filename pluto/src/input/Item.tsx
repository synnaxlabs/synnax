// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useCallback, type ReactElement } from "react";

import { direction } from "@synnaxlabs/x";
import {
  type UseControllerProps,
  useController,
  useFormContext,
} from "react-hook-form";

import { Align } from "@/align";
import { CSS } from "@/css";
import { HelpText } from "@/input/HelpText";
import { Label } from "@/input/Label";
import { Text } from "@/input/Text";
import { type Control, type Value } from "@/input/types";
import { camelToTitle } from "@/util/case";
import { componentRenderProp, type RenderProp } from "@/util/renderProp";

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
> = ItemExtensionProps &
  Omit<UseControllerProps<any, string>, "controller"> & {
    children?: RenderProp<
      Control<I, O> & { onBlur?: () => void; ref?: React.Ref<any> }
    >;
    alsoValidate?: string[];
  };

const defaultChild = componentRenderProp(Text);

export const HFItem = <I extends Value = string | number, O extends Value = I>({
  name,
  rules,
  shouldUnregister,
  defaultValue,
  label,
  children = defaultChild as unknown as RenderProp<Control<I, O>>,
  alsoValidate,
  ...props
}: ItemControlledProps<I, O>): ReactElement => {
  const { field, fieldState } = useController({
    rules,
    name,
    shouldUnregister,
  });
  const { trigger } = useFormContext();
  if (label == null) label = camelToTitle(name);

  const handleBlur = useCallback(() => {
    field.onBlur();
    if (alsoValidate != null) void trigger(alsoValidate);
  }, [field.onBlur, trigger, alsoValidate, name]);

  return (
    <Item label={label} helpText={fieldState.error?.message} {...props}>
      {children({
        ref: field.ref,
        onChange: field.onChange,
        value: field.value,
        onBlur: handleBlur,
      })}
    </Item>
  );
};
