// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/* eslint-disable @typescript-eslint/no-unnecessary-type-constraint */
import {
  array,
  type compare,
  deep,
  type Destructor,
  shallowCopy,
  type status,
} from "@synnaxlabs/x";
import { useCallback, useEffect, useMemo, useSyncExternalStore } from "react";
import { type z } from "zod/v4";

import {
  type BindFunc,
  type ContextValue,
  type Listener,
  type Mode,
  type SetFunc,
  useContext,
} from "@/form/Context";
import {
  type DefaultGetOptions,
  type ExtensionGetOptions,
  type FieldState,
  type GetOptions,
  type OptionalGetOptions,
  type RequiredGetOptions,
  State,
} from "@/form/state";
import { useInitializerRef, useSyncedRef } from "@/hooks/ref";
import { type Input } from "@/input";
import { state } from "@/state";
import { Status } from "@/status";

export type UseOptions<Z extends z.ZodType = z.ZodType> = {
  ctx?: ContextValue<Z>;
};

export type UseFieldOptions<
  I extends Input.Value,
  O extends Input.Value = I,
  Z extends z.ZodType = z.ZodType,
> = UseOptions<Z> & {
  onChange?: (value: O, extra: ContextValue<Z> & { path: string }) => void;
};

/** Return type for the @link useField hook */
export interface UseFieldReturn<I extends Input.Value, O extends Input.Value = I>
  extends FieldState<I> {
  onChange: (value: O) => void;
  setStatus: (status: status.Crude) => void;
  status: status.Crude;
  variant?: Input.Variant;
}

interface UseField {
  <I extends Input.Value, O extends Input.Value = I>(
    path: string,
    opts?: RequiredGetOptions & UseFieldOptions<I, O>,
  ): UseFieldReturn<I, O>;
  <I extends Input.Value, O extends Input.Value = I>(
    path: string,
    opts?: DefaultGetOptions<I> & UseFieldOptions<I, O>,
  ): UseFieldReturn<I, O>;
  <I extends Input.Value, O extends Input.Value = I>(
    path: string,
    opts?: OptionalGetOptions & UseFieldOptions<I, O>,
  ): UseFieldReturn<I, O> | null;
  <I extends Input.Value, O extends Input.Value = I>(
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
export const useField = (<I extends Input.Value, O extends Input.Value = I>(
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
  <I extends Input.Value, O extends Input.Value = I, Z extends z.ZodType = z.ZodType>(
    path: string,
    opts?: RequiredGetOptions & UseOptions<Z>,
  ): O;
  <I extends Input.Value, O extends Input.Value = I, Z extends z.ZodType = z.ZodType>(
    path: string,
    opts?: OptionalGetOptions & UseOptions<Z>,
  ): O | null;
  <I extends Input.Value, O extends Input.Value = I, Z extends z.ZodType = z.ZodType>(
    path: string,
    opts?: DefaultGetOptions<I> & UseOptions<Z>,
  ): O;
  <I extends Input.Value, O extends Input.Value = I, Z extends z.ZodType = z.ZodType>(
    path: string,
    opts?: ExtensionGetOptions<I> & UseOptions<Z>,
  ): O;
}

export interface UseFieldState {
  <I extends Input.Value, O extends Input.Value = I, Z extends z.ZodType = z.ZodType>(
    path: string,
    opts?: RequiredGetOptions & UseOptions<Z>,
  ): FieldState<O>;
  <I extends Input.Value, O extends Input.Value = I, Z extends z.ZodType = z.ZodType>(
    path: string,
    opts?: OptionalGetOptions & UseOptions<Z>,
  ): FieldState<O> | null;
  <I extends Input.Value, O extends Input.Value = I, Z extends z.ZodType = z.ZodType>(
    path: string,
    opts?: DefaultGetOptions<I> & UseOptions<Z>,
  ): FieldState<O>;
  <I extends Input.Value, O extends Input.Value = I, Z extends z.ZodType = z.ZodType>(
    path: string,
    opts?: ExtensionGetOptions<I> & UseOptions<Z>,
  ): FieldState<O>;
}

export const useFieldState = (<
  I extends Input.Value,
  O extends Input.Value = I,
  Z extends z.ZodType = z.ZodType,
>(
  path: string,
  opts?: GetOptions<O> & UseOptions<Z>,
): FieldState<O> | null => {
  const { get, bind } = useContext(opts?.ctx);
  return useSyncExternalStore(
    bind,
    useCallback(() => get<O>(path, opts), [path, get, opts]),
  );
}) as UseFieldState;

export const useFieldValue = (<
  I extends Input.Value,
  O extends Input.Value = I,
  Z extends z.ZodType = z.ZodType,
>(
  path: string,
  opts?: GetOptions<O> & UseOptions<Z>,
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

export const fieldArrayUtils = <V extends unknown = unknown>(
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

export interface UseFieldArrayReturn<V extends unknown> {
  value: V[];
  push: (value: V | V[]) => void;
  add: (value: V | V[], start: number) => void;
  remove: (index: number | number[]) => void;
  keepOnly: (indices: number | number[]) => void;
  set: (values: state.SetArg<V[]>) => void;
}

export const useFieldArray = <
  V extends unknown = unknown,
  Z extends z.ZodType = z.ZodType,
>(
  path: string,
  opts: UseOptions<Z> = {},
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

export interface OnChangeProps<Z extends z.ZodType> {
  /** The values in the form AFTER the change. */
  values: z.infer<Z>;
  /** The path that was changed. */
  path: string;
  /** The previous value at the path. */
  prev: unknown;
  /** Whether validation succeeded. */
  valid: boolean;
}

export interface UseProps<Z extends z.ZodType> {
  values: z.infer<Z>;
  mode?: Mode;
  sync?: boolean;
  onChange?: (props: OnChangeProps<Z>) => void;
  onHasTouched?: (value: boolean) => void;
  schema?: Z;
}

export interface UseReturn<Z extends z.ZodType> extends ContextValue<Z> {}

export const use = <Z extends z.ZodType>({
  values: initialValues,
  sync = false,
  schema,
  mode = "normal",
  onChange,
  onHasTouched,
}: UseProps<Z>): UseReturn<Z> => {
  const ref = useInitializerRef<State<Z>>(() => new State<Z>(initialValues, schema));
  const onChangeRef = useSyncedRef(onChange);
  const handleError = Status.useErrorHandler();
  const onHasTouchedRef = useSyncedRef(onHasTouched);

  const setCurrentStateAsInitialValues = useCallback(() => {
    ref.current.setCurrentStateAsInitialValues();
  }, []);

  const bind: BindFunc = useCallback(
    (handleChange: Listener): Destructor => ref.current.onChange(handleChange),
    [],
  );

  const get: typeof State.prototype.getState = useCallback(
    <V extends unknown = unknown>(
      path: string,
      opts?: GetOptions<V>,
    ): FieldState<V> | null => ref.current.getState(path, opts),
    [],
  ) as typeof State.prototype.getState;

  const remove = useCallback((path: string) => {
    ref.current.remove(path);
    ref.current.notify();
  }, []);

  const reset = useCallback((values?: z.infer<Z>) => {
    ref.current.reset(values);
    ref.current.notify();
  }, []);

  const validateAsync = useCallback(
    async (path?: string, validateChildren?: boolean): Promise<boolean> => {
      const valid = await ref.current.validateAsync(path, validateChildren);
      ref.current.notify();
      return valid;
    },
    [],
  );

  const validate = useCallback((path?: string, validateChildren?: boolean): boolean => {
    const valid = ref.current.validate(path, validateChildren);
    ref.current.notify();
    return valid;
  }, []);

  const set: SetFunc = useCallback((path, value, opts = {}): void => {
    const prev = deep.get(ref.current.values, path, { optional: true });
    const { validateChildren = true } = opts;
    const prevHasTouched = ref.current.hasBeenTouched;
    ref.current.setValue(path, value);
    const finish = () => {
      onChangeRef.current?.({ values: ref.current.values, path, prev, valid: true });
      ref.current.notify();
      const hasTouched = ref.current.hasBeenTouched;
      if (hasTouched !== prevHasTouched) onHasTouchedRef.current?.(hasTouched);
    };
    try {
      ref.current.validate(path, validateChildren);
    } catch (_) {
      return handleError(async () => {
        await ref.current.validateAsync(path, validateChildren);
        finish();
      }, "Failed to validate form");
    }
    finish();
  }, []);

  const has = useCallback(
    (path: string): boolean => deep.has(ref.current.values, path),
    [],
  );

  const setStatus = useCallback((path: string, status: status.Crude): void => {
    const prevHasTouched = ref.current.hasBeenTouched;
    ref.current.setStatus(path, status);
    ref.current.notify();
    const hasTouched = ref.current.hasBeenTouched;
    if (hasTouched !== prevHasTouched) onHasTouchedRef.current?.(hasTouched);
  }, []);

  const clearStatuses = useCallback(() => {
    ref.current.clearStatus();
    ref.current.notify();
  }, []);

  useEffect(() => {
    if (!sync) return;
    ref.current.reset(initialValues);
    ref.current.notify();
  }, [sync, initialValues]);

  return useMemo(
    (): ContextValue<Z> => ({
      bind,
      set,
      get,
      mode,
      validate,
      validateAsync,
      value: () => ref.current.values,
      has,
      remove,
      setStatus,
      clearStatuses,
      reset,
      setCurrentStateAsInitialValues,
    }),
    [
      bind,
      set,
      get,
      validate,
      validateAsync,
      has,
      remove,
      setStatus,
      clearStatuses,
      reset,
      mode,
      setCurrentStateAsInitialValues,
    ],
  );
};
