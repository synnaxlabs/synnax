// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { array, type compare, type record, shallow, type status } from "@synnaxlabs/x";
import { useCallback, useMemo, useSyncExternalStore } from "react";
import { type z } from "zod";

import { type ContextValue, useContext } from "@/form/Context";
import {
  type DefaultGetOptions,
  type ExtensionGetOptions,
  type FieldState,
  type GetOptions,
  type OptionalGetOptions,
  type RequiredGetOptions,
} from "@/form/state";
import { state } from "@/state";

export type ContextOptions<Z extends z.ZodType = z.ZodType> = {
  ctx?: ContextValue<Z>;
};

export type UseFieldOptions<
  I,
  O = I,
  Z extends z.ZodType = z.ZodType,
> = ContextOptions<Z> & {
  onChange?: (value: O, extra: ContextValue<Z> & { path: string }) => void;
};

/** Return type for the @link useField hook */
export interface UseFieldReturn<I, O = I> extends FieldState<I> {
  onChange: (value: O) => void;
  setStatus: (status: status.Crude) => void;
  status: status.Crude;
  variant?: "preview";
}

interface UseField {
  <I, O = I>(
    path: string,
    opts?: RequiredGetOptions & UseFieldOptions<I, O>,
  ): UseFieldReturn<I, O>;
  <I, O = I>(
    path: string,
    opts?: DefaultGetOptions<I> & UseFieldOptions<I, O>,
  ): UseFieldReturn<I, O>;
  <I, O = I>(
    path: string,
    opts?: OptionalGetOptions & UseFieldOptions<I, O>,
  ): UseFieldReturn<I, O> | null;
  <I, O = I>(
    path: string,
    opts?: ExtensionGetOptions<I> & UseFieldOptions<I, O>,
  ): UseFieldReturn<I, O>;
}

/**
 * Hook for managing a particular field in a form.
 *
 * @param props - The props for the hook
 * @param props.path - The path to the field in the form.
 */
export const useField = (<I, O = I>(
  path: string,
  opts: UseFieldOptions<I, O> & GetOptions<I> = {},
): UseFieldReturn<I, O> | null => {
  const { optional = false, onChange, defaultValue } = opts;
  const ctx = useContext(opts?.ctx, `useField(${path})`);
  const { get, bind, set, setStatus } = ctx;

  const handleChange = useCallback(
    (value: O) => {
      onChange?.(value, { ...ctx, path });
      set(path, value);
    },
    [path, set, onChange],
  );

  const handleSetStatus = useCallback(
    (status: status.Crude) => setStatus(path, status),
    [path, setStatus],
  );
  const state = useSyncExternalStore(
    bind,
    useCallback(
      () => get<I>(path, { optional, defaultValue }),
      [path, get, optional, defaultValue],
    ),
  );
  if (state == null) {
    if (!optional) throw new Error(`Field state is null: ${path}`);
    return null;
  }
  const variant = ctx.mode === "preview" ? "preview" : undefined;
  return { onChange: handleChange, setStatus: handleSetStatus, variant, ...state };
}) as UseField;

export interface UseFieldValue {
  <I, O = I, Z extends z.ZodType = z.ZodType>(
    path: string,
    opts?: RequiredGetOptions & ContextOptions<Z>,
  ): O;
  <I, O = I, Z extends z.ZodType = z.ZodType>(
    path: string,
    opts?: OptionalGetOptions & ContextOptions<Z>,
  ): O | null;
  <I, O = I, Z extends z.ZodType = z.ZodType>(
    path: string,
    opts?: DefaultGetOptions<I> & ContextOptions<Z>,
  ): O;
  <I, O = I, Z extends z.ZodType = z.ZodType>(
    path: string,
    opts?: ExtensionGetOptions<I> & ContextOptions<Z>,
  ): O;
}

export interface UseFieldState {
  <I, O = I, Z extends z.ZodType = z.ZodType>(
    path: string,
    opts?: RequiredGetOptions & ContextOptions<Z>,
  ): FieldState<O>;
  <I, O = I, Z extends z.ZodType = z.ZodType>(
    path: string,
    opts?: OptionalGetOptions & ContextOptions<Z>,
  ): FieldState<O> | null;
  <I, O = I, Z extends z.ZodType = z.ZodType>(
    path: string,
    opts?: DefaultGetOptions<I> & ContextOptions<Z>,
  ): FieldState<O>;
  <I, O = I, Z extends z.ZodType = z.ZodType>(
    path: string,
    opts?: ExtensionGetOptions<I> & ContextOptions<Z>,
  ): FieldState<O>;
}

export const useFieldState = (<I, O = I, Z extends z.ZodType = z.ZodType>(
  path: string,
  opts?: GetOptions<O> & ContextOptions<Z>,
): FieldState<O> | null => {
  const { get, bind } = useContext(opts?.ctx);
  return useSyncExternalStore(
    bind,
    useCallback(() => get<O>(path, opts), [path, get, opts]),
  );
}) as UseFieldState;

export const useFieldValue = (<I, O = I, Z extends z.ZodType = z.ZodType>(
  path: string,
  opts?: GetOptions<O> & ContextOptions<Z>,
): O | null => useFieldState(path, opts)?.value ?? null) as UseFieldValue;

export const useFieldValid = (path: string): boolean =>
  useFieldState(path, { optional: true })?.status?.variant === "success";

export interface FieldListUtils<K extends record.Key, E extends record.Keyed<K>> {
  push: (value: E | E[], sort?: compare.Comparator<E>) => void;
  add: (value: E | E[], start: number) => void;
  remove: (keys: K | K[]) => K[];
  keepOnly: (keys: K | K[]) => K[];
  set: (values: state.SetArg<E[]>) => void;
  value(): E[];
  sort?: (compareFn: compare.Comparator<E>) => void;
}

export const fieldListUtils = <K extends record.Key, E extends record.Keyed<K>>(
  ctx: ContextValue<any>,
  path: string,
): FieldListUtils<K, E> => ({
  value: () => ctx.get<E[]>(path).value,
  add: (value, start) => {
    const copy = shallow.copy(ctx.get<E[]>(path).value);
    copy.splice(start, 0, ...array.toArray(value));
    ctx.set(path, copy);
  },
  push: (value, sort) => {
    const copy = shallow.copy(ctx.get<E[]>(path).value);
    copy.push(...array.toArray(value));
    if (sort != null) copy.sort(sort);
    ctx.set(path, copy);
  },
  remove: (key) => {
    const val = ctx.get<E[]>(path).value;
    const keys = new Set(array.toArray(key));
    const next = val.filter(({ key }) => !keys.has(key));
    ctx.set(path, next);
    return next.map(({ key }) => key);
  },
  keepOnly: (key) => {
    const val = ctx.get<E[]>(path).value;
    const keys = new Set(array.toArray(key));
    const next = val.filter(({ key }) => keys.has(key));
    ctx.set(path, next);
    return next.map(({ key }) => key);
  },
  set: (values) => ctx.set(path, state.executeSetter(values, ctx.get<E[]>(path).value)),
  sort: (compareFn) => {
    const copy = shallow.copy(ctx.get<E[]>(path).value);
    copy.sort(compareFn);
    ctx.set(path, copy);
  },
});

export const useFieldListUtils = <K extends record.Key, E extends record.Keyed<K>>(
  path: string,
  opts?: ContextOptions<z.ZodType>,
): FieldListUtils<K, E> => fieldListUtils<K, E>(useContext(opts?.ctx), path);

export interface UseFieldListReturn<K extends record.Key, E extends record.Keyed<K>>
  extends FieldListUtils<K, E> {
  data: K[];
}

export const useFieldList = <
  K extends record.Key,
  E extends record.Keyed<K>,
  Z extends z.ZodType = z.ZodType,
>(
  path: string,
  opts: ContextOptions<Z> & GetOptions<E[]> = {},
): UseFieldListReturn<K, E> => {
  const ctx = useContext(opts?.ctx);
  const value = useFieldValue<E[]>(path, opts);
  return useMemo(
    () => ({
      data: value?.map(({ key }) => key) ?? [],
      ...fieldListUtils<K, E>(ctx, path),
    }),
    [value, ctx, path],
  );
};
