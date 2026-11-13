// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  caseconv,
  deep,
  type optional,
  type record,
  type TimeSpan,
} from "@synnaxlabs/x";
import { type FC, type ReactElement } from "react";

import { type RenderProp, renderProp } from "@/component/renderProp";
import { CSS } from "@/css";
import { type FieldState } from "@/form/state";
import { useField, type UseFieldOptions, type UseFieldReturn } from "@/form/useField";
import { Input } from "@/input";
import { Select } from "@/select";

interface FieldChild<I, O>
  extends Input.Control<I, O>, Pick<UseFieldReturn<I, O>, "preview"> {}

/**
 * @returns the text under a field: its status message, or "Mixed" when the subtrees
 * that `Form.Fanout` edits differ there and the field has no problem.
 */
export const helpTextOf = ({
  mixed,
  status,
}: Pick<FieldState, "mixed" | "status">): string | undefined =>
  mixed === true && status.variant === "success" ? "Mixed" : status.message;

/** Props for {@link Field}. */
export type FieldProps<I = string | number, O = I> = UseFieldOptions<I, O> &
  Omit<Input.ItemProps, "children" | "onChange" | "defaultValue"> & {
    /** The path into the form values. */
    path: deep.Target<I>;
    /** Renders the input. Defaults to a text input. */
    children?: RenderProp<FieldChild<I, O>>;
    /** Whether to hold room for help text, so the layout does not jump. */
    padHelpText?: boolean;
    /** Hides the field when false. */
    visible?: boolean;
    /**
     * Whether a person can empty the input. Only an emptiable field shows a required
     * mark. Defaults to true for the default text input and false for `children`.
     */
    emptiable?: boolean;
  };

const defaultInput = renderProp((p: Input.TextProps) => <Input.Text {...p} />);

export type FieldT<I, O = I> = (props: FieldProps<I, O>) => ReactElement | null;

/**
 * One labeled row of a form: its label, its input, and its validation message. The
 * label falls back to the last path element in sentence case.
 *
 * @example <Form.Field path={path.range.name} />
 * @example <Form.Field path={path.rate}>{(p) => <Input.Numeric {...p} />}</Form.Field>
 */
export const Field = <I = string | number, O = I>({
  path,
  children,
  emptiable,
  label,
  padHelpText = true,
  visible = true,
  onChange,
  className,
  ...rest
}: FieldProps<I, O>): ReactElement | null => {
  // A hidden field may bind a path its config lacks.
  const field = useField<I, O>(path, { onChange, optional: !visible });
  const markable = emptiable ?? children == null;
  if (field == null || !visible) return null;
  const key = String(path);
  if (label == null) label = caseconv.toSentence(deep.element(key, -1));
  const helpText = helpTextOf(field);
  const { onChange: fieldOnChange, value } = field;
  const childrenProps: FieldChild<I, O> = { onChange: fieldOnChange, value };
  if (field.preview != null) childrenProps.preview = field.preview;
  return (
    <Input.Item
      padHelpText={padHelpText}
      helpText={helpText}
      status={field.status.variant}
      label={label}
      required={markable && field.required && field.preview !== true}
      className={CSS.cls(
        className,
        CSS.BE("field", key.split(".").join("-")),
        CSS.M(field.status.variant),
      )}
      {...rest}
    >
      {(children ?? (defaultInput as unknown as RenderProp<FieldChild<I, O>>))(
        childrenProps,
      )}
    </Input.Item>
  );
};

export interface FieldBuilderProps<I, O, P extends {}> {
  fieldProps?: Partial<FieldProps<I, O>>;
  inputProps: Omit<P, "value" | "onChange">;
}

export type BuiltFieldProps<
  I,
  O,
  P extends {},
  OptionalFields extends keyof Omit<P, "value" | "onChange"> = never,
> = Omit<FieldProps<I, O>, "path"> & {
  /** The path into the form values. A built field takes an optional value. */
  path: deep.Target<I | undefined>;
  inputProps?: optional.Optional<Omit<P, "value" | "onChange">, OptionalFields>;
};

/**
 * Binds an input component to a form field once, so callers write the field instead of
 * a render prop. Use it for an input the app reaches for often.
 *
 * @example
 * export const RateField = fieldBuilder(Input.Numeric)({
 *   inputProps: { endContent: "Hz" },
 * });
 */
export const fieldBuilder =
  <
    I,
    O,
    P extends {},
    OptionalFields extends keyof Omit<P, "value" | "onChange"> = never,
  >(
    Component: FC<P & Input.Control<I, O>>,
    defaults: Partial<FieldProps<I, O>> = {},
  ) =>
  ({
    fieldProps,
    inputProps: baseInputProps,
  }: FieldBuilderProps<I, O, P>): FC<BuiltFieldProps<I, O, P, OptionalFields>> => {
    const C = ({ inputProps, path, ...rest }: BuiltFieldProps<I, O, P>) => (
      <Field<I, O>
        {...defaults}
        {...fieldProps}
        {...rest}
        path={path as deep.Target<I>}
      >
        {(cp) => <Component {...cp} {...baseInputProps} {...(inputProps as P)} />}
      </Field>
    );
    C.displayName = Component.displayName;
    return C as FC<BuiltFieldProps<I, O, P, OptionalFields>>;
  };

export type NumericFieldProps = BuiltFieldProps<number, number, Input.NumericProps>;
export const buildNumericField = fieldBuilder<number, number, Input.NumericProps>(
  Input.Numeric,
);
export const NumericField = buildNumericField({ inputProps: {} });

export type TextFieldProps = BuiltFieldProps<string, string, Input.TextProps>;
export const buildTextField = fieldBuilder<string, string, Input.TextProps>(
  Input.Text,
  { emptiable: true },
);
export const TextField = buildTextField({ inputProps: {} });

export type SwitchFieldProps = BuiltFieldProps<boolean, boolean, Input.SwitchProps>;
export const buildSwitchField = fieldBuilder<boolean, boolean, Input.SwitchProps>(
  Input.Switch,
);
export const SwitchField = buildSwitchField({ inputProps: {} });

export interface CheckboxFieldProps extends BuiltFieldProps<
  boolean,
  boolean,
  Input.CheckboxProps
> {
  /**
   * Checks the box when the stored value is false, for a flag that names the
   * non-default condition, such as `dataSavingDisabled` under "Data saving".
   */
  inverted?: boolean;
}

/** A checkbox field, for a setting that the form saves on a button. */
export const CheckboxField = ({
  inverted = false,
  inputProps,
  path,
  ...rest
}: CheckboxFieldProps): ReactElement | null => (
  <Field<boolean> {...rest} path={path as deep.Target<boolean>}>
    {({ value, onChange, ...p }) => (
      <Input.Checkbox
        {...p}
        {...inputProps}
        value={inverted ? !value : value}
        onChange={(v) => onChange(inverted ? !v : v)}
      />
    )}
  </Field>
);

export type DateTimeFieldProps = BuiltFieldProps<number, number, Input.DateTimeProps>;
export const buildDateTimeField = fieldBuilder<number, number, Input.DateTimeProps>(
  Input.DateTime,
);
export const DateTimeField = buildDateTimeField({ inputProps: {} });

export interface TimeSpanFieldProps extends BuiltFieldProps<
  number | undefined,
  number | undefined,
  Omit<Input.TimeSpanProps, "unit" | "inline" | "check" | "onClear">
> {
  /** The unit of the stored number, and of a number typed alone. */
  unit: TimeSpan;
  /** The shortest and longest spans the field accepts. */
  bounds?: { lower?: TimeSpan; upper?: TimeSpan };
}

/**
 * A duration field, typed in place. It stores a number of `unit`. An emptiable field
 * stores undefined when emptied, and any other field reverts.
 *
 * @example <Form.TimeSpanField path={path.timeoutMs} unit={TimeSpan.MILLISECOND} />
 */
export const TimeSpanField = ({
  unit,
  bounds,
  inputProps,
  path,
  label,
  emptiable,
  ...rest
}: TimeSpanFieldProps): ReactElement | null => {
  const name = label ?? caseconv.toSentence(deep.element(String(path), -1));
  const check = (span: TimeSpan): string | null => {
    if (bounds?.lower != null && span.lessThan(bounds.lower))
      return `${name} must be ${Input.formatTimeSpan(bounds.lower)} or more`;
    if (bounds?.upper != null && span.greaterThan(bounds.upper))
      return `${name} must be ${Input.formatTimeSpan(bounds.upper)} or less`;
    return null;
  };
  return (
    <Field<number | undefined> {...rest} label={name} emptiable={emptiable} path={path}>
      {({ value, onChange, ...p }) => (
        <Input.TimeSpan
          {...p}
          {...inputProps}
          inline
          unit={unit}
          value={value == null ? undefined : Number(unit.mult(value).valueOf())}
          onChange={(ns) => onChange(ns / Number(unit.valueOf()))}
          onClear={emptiable === true ? () => onChange(undefined) : undefined}
          check={bounds == null ? undefined : check}
        />
      )}
    </Field>
  );
};

export type SelectFieldProps<K extends record.Key> = BuiltFieldProps<
  K,
  K,
  Select.SingleSimpleProps<K>,
  "children" | "resourceName"
>;
export const buildSelectField = <K extends record.Key>(
  props: FieldBuilderProps<K, K, Select.SingleSimpleProps<K>>,
) =>
  fieldBuilder<K, K, Select.SingleSimpleProps<K>, "children" | "resourceName">(
    Select.Simple<K>,
  )(props);
