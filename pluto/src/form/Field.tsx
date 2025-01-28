// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file licenses/APL.txt.

import { caseconv, deep, type Key, type Keyed } from "@synnaxlabs/x";
import { type FC, type ReactElement } from "react";

import { CSS } from "@/css";
import {
  type ContextValue,
  type FieldState,
  useContext,
  useField,
  type UseFieldProps,
  type UseNullableFieldProps,
} from "@/form/Form";
import { Input } from "@/input";
import { Select } from "@/select";
import { componentRenderProp, type RenderProp } from "@/util/renderProp";

interface FieldChild<I extends Input.Value, O extends Input.Value>
  extends Input.Control<I, O> {
  variant?: Input.Variant;
}

export type FieldProps<
  I extends Input.Value = string | number,
  O extends Input.Value = I,
> = (UseFieldProps<I, O> | UseNullableFieldProps<I, O>) &
  Omit<Input.ItemProps, "children" | "onChange" | "defaultValue"> & {
    children?: RenderProp<FieldChild<I, O>>;
    padHelpText?: boolean;
    visible?: boolean | ((state: FieldState<I>, ctx: ContextValue) => boolean);
    hideIfNull?: boolean;
  };

const defaultInput = componentRenderProp(Input.Text);

export type FieldT<I extends Input.Value, O extends Input.Value = I> = (
  props: FieldProps<I, O>,
) => ReactElement | null;

export const Field = <
  I extends Input.Value = string | number,
  O extends Input.Value = I,
>({
  path,
  children = defaultInput as unknown as RenderProp<FieldChild<I, O>>,
  label,
  padHelpText = true,
  visible = true,
  hideIfNull = false,
  optional,
  onChange,
  className,
  ...props
}: FieldProps<I, O>): ReactElement | null => {
  const field = useField<I, O>({
    path,
    optional: (optional as true) ?? (hideIfNull as true),
    onChange,
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
  console.log(field.status.variant, helpText);
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
      {...props}
    >
      {children(childrenProps)}
    </Input.Item>
  );
};

export interface FieldBuilderProps<
  I extends Input.Value,
  O extends Input.Value,
  P extends {},
> {
  fieldKey?: string;
  fieldProps?: Partial<FieldProps<I, O>>;
  inputProps?: Partial<P>;
}

export type BuiltFieldProps<
  I extends Input.Value,
  O extends Input.Value,
  P extends {},
> = FieldProps<I, O> & {
  inputProps?: Partial<P>;
  fieldKey?: string;
};

export const fieldBuilder =
  <I extends Input.Value, O extends Input.Value, P extends {}>(
    Component: FC<P & Input.Control<I, O>>,
  ) =>
  ({
    fieldKey: baseFieldKey,
    fieldProps,
    inputProps: baseInputProps,
  }: FieldBuilderProps<I, O, P>): FC<BuiltFieldProps<I, O, P>> => {
    const C = ({
      inputProps,
      path,
      fieldKey = baseFieldKey,
      ...props
    }: BuiltFieldProps<I, O, P>) => (
      <Field<I, O>
        {...fieldProps}
        {...props}
        path={fieldKey ? `${path}.${fieldKey}` : path}
      >
        {(cp) => <Component {...cp} {...baseInputProps} {...(inputProps as P)} />}
      </Field>
    );
    C.displayName = Component.displayName;
    return C;
  };

export const buildNumericField = fieldBuilder(Input.Numeric);
export const NumericField = buildNumericField({});
export const buildTextField = fieldBuilder(Input.Text);
export const TextField = buildTextField({});
export const buildSwitchField = fieldBuilder(Input.Switch);
export const SwitchField = buildSwitchField({});
export const buildSelectSingleField = fieldBuilder(Select.Single) as <
  K extends Key,
  E extends Keyed<K>,
>({
  fieldProps,
  inputProps,
}: FieldBuilderProps<K, K, Select.SingleProps<K, E>>) => FC<
  BuiltFieldProps<K, K, Select.SingleProps<K, E>>
>;

export const buildSelectMultiField = fieldBuilder(Select.Multiple) as <
  K extends Key,
  E extends Keyed<K>,
>({
  fieldProps,
  inputProps,
}: FieldBuilderProps<K, K, Select.MultipleProps<K, E>>) => FC<
  BuiltFieldProps<K, K, Select.MultipleProps<K, E>>
>;

export const buildDropdownButtonSelectField = fieldBuilder(Select.DropdownButton) as <
  K extends Key,
  E extends Keyed<K>,
>({
  fieldProps,
  inputProps,
}: FieldBuilderProps<K, K, Select.DropdownButtonProps<K, E>>) => FC<
  BuiltFieldProps<K, K, Select.DropdownButtonProps<K, E>>
>;
