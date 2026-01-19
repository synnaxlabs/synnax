// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { deep, type destructor, type status } from "@synnaxlabs/x";
import { useCallback, useEffect, useMemo } from "react";
import { type z } from "zod";

import {
  type BindFunc,
  type ContextValue,
  type Listener,
  type Mode,
  type SetFunc,
} from "@/form/Context";
import { type FieldState, type GetOptions, State } from "@/form/state";
import { useInitializerRef, useSyncedRef } from "@/hooks/ref";
import { Status } from "@/status/base";

export interface OnChangeArgs<Z extends z.ZodType> {
  /** The values in the form AFTER the change. */
  values: z.infer<Z>;
  /** The path that was changed. */
  path: string;
  /** The previous value at the path. */
  prev: unknown;
  /** Whether validation succeeded. */
  valid: boolean;
}

export interface UseArgs<Z extends z.ZodType> {
  values: z.infer<Z>;
  mode?: Mode;
  sync?: boolean;
  onChange?: (props: OnChangeArgs<Z>) => void;
  onHasTouched?: (value: boolean) => void;
  schema?: Z;
  scope?: string;
}

export interface UseReturn<Z extends z.ZodType> extends ContextValue<Z> {}

export const use = <Z extends z.ZodType>({
  values: initialValues,
  sync = false,
  schema,
  mode = "normal",
  onChange,
  onHasTouched,
}: UseArgs<Z>): UseReturn<Z> => {
  const ref = useInitializerRef<State<Z>>(() => new State<Z>(initialValues, schema));
  const onChangeRef = useSyncedRef(onChange);
  const handleError = Status.useErrorHandler();
  const onHasTouchedRef = useSyncedRef(onHasTouched);

  const setCurrentStateAsInitialValues = useCallback(() => {
    ref.current.setCurrentStateAsInitialValues();
    onHasTouchedRef.current?.(false);
  }, []);

  const bind: BindFunc = useCallback(
    (handleChange: Listener): destructor.Destructor =>
      ref.current.onChange(handleChange),
    [],
  );

  const get: typeof State.prototype.getState = useCallback(
    <V = unknown>(path: string, opts?: GetOptions<V>): FieldState<V> | null =>
      ref.current.getState(path, opts),
    [],
  ) as typeof State.prototype.getState;

  const remove = useCallback((path: string) => {
    ref.current.remove(path);
    ref.current.notify();
  }, []);

  const reset = useCallback((values?: z.infer<Z>) => {
    const prevValues = ref.current.values;
    const prevHasTouched = ref.current.hasBeenTouched;
    ref.current.reset(values);
    const valuesChanged = prevValues !== ref.current.values;
    if (valuesChanged)
      onChangeRef.current?.({
        values: ref.current.values,
        path: "",
        prev: prevValues,
        valid: true,
      });
    ref.current.notify();
    const hasTouched = ref.current.hasBeenTouched;
    if (hasTouched !== prevHasTouched) onHasTouchedRef.current?.(hasTouched);
  }, []);

  const validateAsync = useCallback(async (path?: string): Promise<boolean> => {
    const valid = await ref.current.validateAsync(true, path);
    ref.current.notify();
    return valid;
  }, []);

  const validate = useCallback((path?: string): boolean => {
    const valid = ref.current.validate(true, path);
    ref.current.notify();
    return valid;
  }, []);

  const set: SetFunc = useCallback((path, value, options): void => {
    const { notifyOnChange = true, markTouched = true } = options ?? {};
    const prev = deep.get(ref.current.values, path, { optional: true });
    const prevHasTouched = ref.current.hasBeenTouched;
    ref.current.setValue(path, value, { markTouched });
    const finish = () => {
      if (notifyOnChange)
        onChangeRef.current?.({ values: ref.current.values, path, prev, valid: true });
      ref.current.notify();
      const hasTouched = ref.current.hasBeenTouched;
      if (hasTouched !== prevHasTouched) onHasTouchedRef.current?.(hasTouched);
    };
    try {
      ref.current.validate();
    } catch (_) {
      return handleError(async () => {
        await ref.current.validateAsync();
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
      getStatuses: () => ref.current.getStatuses(),
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
