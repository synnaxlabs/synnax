import { type optional, type record } from "@synnaxlabs/x";
import { type FC, type ReactElement } from "react";
import { type RenderProp } from "../component/renderProp";
import { type ContextValue } from "./Context";
import { type FieldState, type GetOptions } from "./state";
import { type UseFieldOptions, type UseFieldReturn } from "./useField";
import { Input } from "../input";
import { Select } from "../select";
interface FieldChild<I, O> extends Input.Control<I, O>, Pick<UseFieldReturn<I, O>, "preview"> {
}
/** Props for {@link Field}. */
export type FieldProps<I = string | number, O = I> = GetOptions<I> & UseFieldOptions<I, O> & Omit<Input.ItemProps, "children" | "onChange" | "defaultValue"> & {
    /** Dot-separated path into the form values. */
    path: string;
    /** Renders the input. Defaults to a text input. */
    children?: RenderProp<FieldChild<I, O>>;
    /** Whether to hold room for help text, so the layout does not jump. */
    padHelpText?: boolean;
    /** Hides the field when false, or when the predicate rejects its state. */
    visible?: boolean | ((state: FieldState<I>, ctx: ContextValue) => boolean);
    /** Whether an absent value hides the field instead of throwing. */
    hideIfNull?: boolean;
};
export type FieldT<I, O = I> = (props: FieldProps<I, O>) => ReactElement | null;
/**
 * One labeled row of a form: its label, its input, and its validation message. The
 * label falls back to the last path element in sentence case.
 *
 * @example <Form.Field path="range.name" />
 * @example <Form.Field path="rate">{(p) => <Input.Numeric {...p} />}</Form.Field>
 */
export declare const Field: <I = string | number, O = I>({ path, children, label, padHelpText, visible, hideIfNull, optional, onChange, className, defaultValue, ...rest }: FieldProps<I, O>) => ReactElement | null;
export interface FieldBuilderProps<I, O, P extends {}> {
    fieldKey?: string;
    fieldProps?: Partial<FieldProps<I, O>>;
    inputProps: Omit<P, "value" | "onChange">;
}
export type BuiltFieldProps<I, O, P extends {}, OptionalFields extends keyof Omit<P, "value" | "onChange"> = never> = FieldProps<I, O> & {
    inputProps?: optional.Optional<Omit<P, "value" | "onChange">, OptionalFields>;
    fieldKey?: string;
};
/**
 * Binds an input component to a form field once, so callers write the field instead of
 * a render prop. Use it for an input the app reaches for often.
 *
 * @example
 * export const RateField = fieldBuilder(Input.Numeric)({ inputProps: { units: "Hz" } });
 */
export declare const fieldBuilder: <I, O, P extends {}, OptionalFields extends keyof Omit<P, "value" | "onChange"> = never>(Component: FC<P & Input.Control<I, O>>) => ({ fieldKey: baseFieldKey, fieldProps, inputProps: baseInputProps, }: FieldBuilderProps<I, O, P>) => FC<BuiltFieldProps<I, O, P, OptionalFields>>;
export type NumericFieldProps = BuiltFieldProps<number, number, Input.NumericProps>;
export declare const buildNumericField: ({ fieldKey: baseFieldKey, fieldProps, inputProps: baseInputProps, }: FieldBuilderProps<number, number, Input.NumericProps>) => FC<BuiltFieldProps<number, number, Input.NumericProps, never>>;
export declare const NumericField: FC<BuiltFieldProps<number, number, Input.NumericProps, never>>;
export type TextFieldProps = BuiltFieldProps<string, string, Input.TextProps>;
export declare const buildTextField: ({ fieldKey: baseFieldKey, fieldProps, inputProps: baseInputProps, }: FieldBuilderProps<string, string, Input.TextProps>) => FC<BuiltFieldProps<string, string, Input.TextProps, never>>;
export declare const TextField: FC<BuiltFieldProps<string, string, Input.TextProps, never>>;
export type SwitchFieldProps = BuiltFieldProps<boolean, boolean, Input.SwitchProps>;
export declare const buildSwitchField: ({ fieldKey: baseFieldKey, fieldProps, inputProps: baseInputProps, }: FieldBuilderProps<boolean, boolean, Input.SwitchProps>) => FC<BuiltFieldProps<boolean, boolean, Input.SwitchProps, never>>;
export declare const SwitchField: FC<BuiltFieldProps<boolean, boolean, Input.SwitchProps, never>>;
export type DateTimeFieldProps = BuiltFieldProps<number, number, Input.DateTimeProps>;
export declare const buildDateTimeField: ({ fieldKey: baseFieldKey, fieldProps, inputProps: baseInputProps, }: FieldBuilderProps<number, number, Input.DateTimeProps>) => FC<BuiltFieldProps<number, number, Input.DateTimeProps, never>>;
export declare const DateTimeField: FC<BuiltFieldProps<number, number, Input.DateTimeProps, never>>;
export type SelectFieldProps<K extends record.Key, E extends record.KeyedNamed<K>> = BuiltFieldProps<K, K, Select.StaticProps<K, E>, "data" | "resourceName">;
export declare const buildSelectField: <K extends record.Key, E extends record.KeyedNamed<K>>(props: FieldBuilderProps<K, K, Select.StaticProps<K, E>>) => FC<BuiltFieldProps<K, K, Select.StaticProps<K, E>, "data" | "resourceName">>;
export {};
//# sourceMappingURL=Field.d.ts.map