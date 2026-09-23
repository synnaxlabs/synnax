import { type compare, type record, state } from "@synnaxlabs/x";
import { type z } from "zod";
import { type ContextValue } from "./Context";
import { type DefaultGetOptions, type ExtensionGetOptions, type FieldState, type GetOptions, type OptionalGetOptions, type RequiredGetOptions } from "./state";
import { type Status } from "../status";
export type ContextOptions<Z extends z.ZodType = z.ZodType> = {
    ctx?: ContextValue<Z>;
};
export type UseFieldOptions<I, O = I, Z extends z.ZodType = z.ZodType> = ContextOptions<Z> & {
    onChange?: (value: O, extra: ContextValue<Z> & {
        path: string;
    }) => void;
};
/** Return value for {@link useField}. Spread it onto an input. */
export interface UseFieldReturn<I, O = I> extends FieldState<I> {
    onChange: (value: O) => void;
    setStatus: (status: Status.Crude) => void;
    status: Status.Crude;
    preview?: boolean;
}
interface UseField {
    <I, O = I>(path: string, opts?: (RequiredGetOptions | DefaultGetOptions<I>) & UseFieldOptions<I, O>): UseFieldReturn<I, O>;
    <I, O = I>(path: string, opts?: OptionalGetOptions & UseFieldOptions<I, O>): UseFieldReturn<I, O> | null;
    <I, O = I>(path: string, opts?: ExtensionGetOptions<I> & UseFieldOptions<I, O>): UseFieldReturn<I, O>;
}
/**
 * Binds one field of the enclosing form, re-rendering the caller only when that field
 * changes. The return value satisfies {@link Input.Control}, so it spreads onto any
 * input.
 *
 * @example const { value, onChange } = Form.useField<string>("name");
 * @throws {Error} if the path holds no value and `optional` is not set.
 */
export declare const useField: UseField;
export interface UseFieldValue {
    <I, O = I, Z extends z.ZodType = z.ZodType>(path: string, opts?: (RequiredGetOptions | DefaultGetOptions<I>) & ContextOptions<Z>): O;
    <I, O = I, Z extends z.ZodType = z.ZodType>(path: string, opts?: OptionalGetOptions & ContextOptions<Z>): O | null;
    <I, O = I, Z extends z.ZodType = z.ZodType>(path: string, opts?: ExtensionGetOptions<I> & ContextOptions<Z>): O;
}
export interface UseFieldState {
    <I, O = I, Z extends z.ZodType = z.ZodType>(path: string, opts?: (RequiredGetOptions | DefaultGetOptions<I>) & ContextOptions<Z>): FieldState<O>;
    <I, O = I, Z extends z.ZodType = z.ZodType>(path: string, opts?: OptionalGetOptions & ContextOptions<Z>): FieldState<O> | null;
    <I, O = I, Z extends z.ZodType = z.ZodType>(path: string, opts?: ExtensionGetOptions<I> & ContextOptions<Z>): FieldState<O>;
}
/** Reads a field's value together with its status and required flag. */
export declare const useFieldState: UseFieldState;
/** Reads just a field's value. Use it to render from a field the caller does not edit. */
export declare const useFieldValue: UseFieldValue;
/** @returns whether the field at the path passed its last validation. */
export declare const useFieldValid: (path: string) => boolean;
/** Edits for a form field holding an array of keyed entries. */
export interface FieldListUtils<K extends record.Key, E extends record.Keyed<K>> {
    /** Appends entries, optionally re-sorting after. */
    push: (value: E | E[], sort?: compare.Comparator<E>) => void;
    /** Inserts entries at an index. */
    add: (value: E | E[], start: number) => void;
    /** Drops the named entries. @returns the keys that remain. */
    remove: (keys: K | K[]) => K[];
    /** Drops everything but the named entries. @returns the keys that remain. */
    keepOnly: (keys: K | K[]) => K[];
    set: (values: state.SetArg<E[]>) => void;
    value(): E[];
    sort?: (compareFn: compare.Comparator<E>) => void;
}
/** Builds {@link FieldListUtils} over a form context, outside of a React render. */
export declare const fieldListUtils: <K extends record.Key, E extends record.Keyed<K>>(ctx: ContextValue<any>, path: string) => FieldListUtils<K, E>;
/** @returns edits for an array field, without subscribing to its contents. */
export declare const useFieldListUtils: <K extends record.Key, E extends record.Keyed<K>>(path: string, opts?: ContextOptions<z.ZodType>) => FieldListUtils<K, E>;
export interface UseFieldListReturn<K extends record.Key, E extends record.Keyed<K>> extends FieldListUtils<K, E> {
    data: K[];
}
/**
 * Binds an array field, returning its keys alongside the edits. Feed `data` straight
 * into a {@link List.Frame}.
 */
export declare const useFieldList: <K extends record.Key, E extends record.Keyed<K>, Z extends z.ZodType = z.ZodType>(path: string, opts?: ContextOptions<Z> & GetOptions<E[]>) => UseFieldListReturn<K, E>;
export {};
//# sourceMappingURL=useField.d.ts.map