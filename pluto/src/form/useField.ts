// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { array, type compare, shallowCopy, type status } from "@synnaxlabs/x";
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
import { type Input } from "@/input";
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
  variant?: Input.Variant;
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
  const ctx = useContext(opts?.ctx);
  const { get: getState, bind, set, setStatus } = ctx;

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
      () => getState<I>(path, { optional, defaultValue }),
      [path, getState, optional, defaultValue],
    ),
  );
  if (state == null) {
    if (!optional) throw new Error(`Field state is null: ${path}`);
    return null;
  }
  let variant: Input.Variant | undefined;
  if (ctx.mode === "preview") variant = "preview";
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
): O | null => {
  const { get, bind } = useContext(opts?.ctx);
  return useSyncExternalStore(
    bind,
    useCallback(() => get<O>(path, opts)?.value ?? null, [path, get, opts]),
  );
}) as UseFieldValue;

export const useFieldValid = (path: string): boolean =>
  useFieldState(path, { optional: true })?.status?.variant === "success";

export interface FieldArrayUtils<V> {
  push: (value: V | V[], sort?: compare.CompareF<V>) => void;
  add: (value: V | V[], start: number) => void;
  remove: (index: number | number[]) => void;
  keepOnly: (indices: number | number[]) => void;
  set: (values: state.SetArg<V[]>) => void;
  sort?: (compareFn: compare.CompareF<V>) => void;
}

export const fieldArrayUtils = <V = unknown>(
  ctx: ContextValue<any>,
  path: string,
): FieldArrayUtils<V> => ({
  add: (value, start) => {
    const copy = shallowCopy(ctx.get<V[]>(path).value);
    copy.splice(start, 0, ...array.toArray(value));
    ctx.set(path, copy, { validateChildren: false });
  },
  push: (value, sort) => {
    const copy = shallowCopy(ctx.get<V[]>(path).value);
    copy.push(...array.toArray(value));
    if (sort != null) copy.sort(sort);
    ctx.set(path, copy, { validateChildren: false });
  },
  remove: (index) => {
    const val = ctx.get<V[]>(path).value;
    const indices = new Set(array.toArray(index));
    ctx.set(
      path,
      val.filter((_, i) => !indices.has(i)),
    );
  },
  keepOnly: (index) => {
    const val = ctx.get<V[]>(path).value;
    const indices = new Set(array.toArray(index));
    ctx.set(
      path,
      val.filter((_, i) => indices.has(i)),
    );
  },
  set: (values) => ctx.set(path, state.executeSetter(values, ctx.get<V[]>(path).value)),
  sort: (compareFn) => {
    const copy = shallowCopy(ctx.get<V[]>(path).value);
    copy.sort(compareFn);
    ctx.set(path, copy);
  },
});

export interface UseFieldArrayReturn<V = unknown> {
  value: V[];
  push: (value: V | V[]) => void;
  add: (value: V | V[], start: number) => void;
  remove: (index: number | number[]) => void;
  keepOnly: (indices: number | number[]) => void;
  set: (values: state.SetArg<V[]>) => void;
}

export const useFieldArray = <V = unknown, Z extends z.ZodType = z.ZodType>(
  path: string,
  opts: ContextOptions<Z> = {},
): UseFieldArrayReturn<V> => {
  const ctx = useContext(opts?.ctx);
  const { get: getState } = ctx;
  const fState = useSyncExternalStore(
    ctx.bind,
    useCallback(() => getState<V[]>(path).value, [path, getState]),
  );
  return useMemo(
    () => ({ value: fState, ...fieldArrayUtils<V>(ctx, path) }),
    [fState, ctx, path],
  );
};
