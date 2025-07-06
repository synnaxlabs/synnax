// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { caseconv, deep } from "@synnaxlabs/x";
import { type FC, type ReactElement } from "react";

import { type RenderProp, renderProp } from "@/component/renderProp";
import { CSS } from "@/css";
import { type ContextValue, useContext } from "@/form/Context";
import { type FieldState, type GetOptions } from "@/form/state";
import { useField, type UseFieldOptions } from "@/form/useField";
import { Input } from "@/input";

interface FieldChild<I, O> extends Input.Control<I, O> {
  variant?: Input.Variant;
}

export type FieldProps<I = string | number, O = I> = GetOptions<I> &
  UseFieldOptions<I, O> &
  Omit<Input.ItemProps, "children" | "onChange" | "defaultValue"> & {
    path: string;
    children?: RenderProp<FieldChild<I, O>>;
    padHelpText?: boolean;
    visible?: boolean | ((state: FieldState<I>, ctx: ContextValue) => boolean);
    hideIfNull?: boolean;
  };

const defaultInput = renderProp(Input.Text);

export type FieldT<I, O = I> = (props: FieldProps<I, O>) => ReactElement | null;

export const Field = <I = string | number, O = I>({
  path,
  children = defaultInput as unknown as RenderProp<FieldChild<I, O>>,
  label,
  padHelpText = true,
  visible = true,
  hideIfNull = false,
  optional,
  onChange,
  className,
  defaultValue,
  ...rest
}: FieldProps<I, O>): ReactElement | null => {
  const field = useField<I, O>(path, {
    optional: optional ?? hideIfNull,
    onChange,
    defaultValue,
  });
  const ctx = useContext();
  if (field == null) return null;
  if (path == null) throw new Error("No path provided to Form Field");
  label ??= caseconv.capitalize(deep.element(path, -1));
  visible = typeof visible === "function" ? visible(field, ctx) : visible;
  if (!visible) return null;
  const helpText = field.status.message;
  const { onChange: fieldOnChange, value } = field;
  const childrenProps: FieldChild<I, O> = { onChange: fieldOnChange, value };
  if (field.variant != null) childrenProps.variant = field.variant;
  return (
    <Input.Item
      padHelpText={padHelpText}
      helpText={helpText}
      helpTextVariant={field.status.variant}
      label={label}
      required={field.required}
      className={CSS(
        className,
        CSS.BE("field", path.split(".").join("-")),
        CSS.M(field.status.variant),
      )}
      {...rest}
    >
      {children(childrenProps)}
    </Input.Item>
  );
};

export interface FieldBuilderProps<I, O, P extends {}> {
  fieldKey?: string;
  fieldProps?: Partial<FieldProps<I, O>>;
  inputProps?: Partial<P>;
}

export type BuiltFieldProps<I, O, P extends {}> = FieldProps<I, O> & {
  inputProps?: Partial<P>;
  fieldKey?: string;
};

export const fieldBuilder =
  <I, O, P extends {}>(Component: FC<P & Input.Control<I, O>>) =>
  ({
    fieldKey: baseFieldKey,
    fieldProps,
    inputProps: baseInputProps,
  }: FieldBuilderProps<I, O, P>): FC<BuiltFieldProps<I, O, P>> => {
    const C = ({
      inputProps,
      path,
      fieldKey = baseFieldKey,
      optional,
      defaultValue,
      ...rest
    }: BuiltFieldProps<I, O, P>) => (
      <Field<I, O>
        {...fieldProps}
        {...rest}
        defaultValue={defaultValue}
        optional={optional}
        path={fieldKey ? `${path}.${fieldKey}` : path}
      >
        {(cp) => <Component {...cp} {...baseInputProps} {...(inputProps as P)} />}
      </Field>
    );
    C.displayName = Component.displayName;
    return C;
  };

export type NumericFieldProps = BuiltFieldProps<number, number, Input.NumericProps>;
export const buildNumericField = fieldBuilder(Input.Numeric);
export const NumericField = buildNumericField({});

export type TextFieldProps = BuiltFieldProps<string, string, Input.TextProps>;
export const buildTextField = fieldBuilder(Input.Text);
export const TextField = buildTextField({});

export type SwitchFieldProps = BuiltFieldProps<boolean, boolean, Input.SwitchProps>;
export const buildSwitchField = fieldBuilder(Input.Switch);
export const SwitchField = buildSwitchField({});
