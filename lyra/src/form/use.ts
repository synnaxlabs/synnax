// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { deep, type destructor, errors } from "@synnaxlabs/x";
import { useLayoutEffect, useRef } from "react";
import { type z } from "zod";

import {
  type BindFunc,
  type ContextValue,
  type InsertFunc,
  type Listener,
  type Mode,
  type MoveFunc,
  type SetFunc,
} from "@/form/Context";
import { type FieldState, type GetOptions, State } from "@/form/state";
import { createLocalStore, type Store } from "@/form/store";
import { useInitializerRef, useSyncedRef } from "@/hooks/ref";
import { Status } from "@/status";

export interface OnChangeParams<Z extends z.ZodType> {
  /** The values in the form AFTER the change. */
  values: z.infer<Z>;
  /** The path that was changed. */
  path: string;
  /** The previous value at the path. */
  prev: unknown;
  /** Whether validation succeeded. */
  valid: boolean;
}

interface BaseUseParams<Z extends z.ZodType> {
  mode?: Mode;
  onChange?: (props: OnChangeParams<Z>) => void;
  /** Called when the form goes from untouched to touched, and back. */
  onHasTouched?: (value: boolean) => void;
  /** Zod schema the values validate against on every write. */
  schema?: Z;
  scope?: string;
}

/**
 * Params for {@link use}. A form holds a copy of `values`, or reads and writes the
 * values that a store holds.
 */
export type UseParams<Z extends z.ZodType> = BaseUseParams<Z> &
  (
    { values: z.infer<Z>; store?: never } | { store: Store<z.infer<Z>>; values?: never }
  );

interface Change {
  path: string;
  prev: unknown;
  /** Whether the change skips onChange. */
  silent: boolean;
}

export interface UseReturn<Z extends z.ZodType> extends ContextValue<Z> {}

/**
 * Creates a form over the given values, or over a store that holds them, validated by
 * the given Zod schema. The returned value is stable, so a field hook re-renders only
 * when its own path changes.
 *
 * @example
 * const form = Form.use({ values: initial, schema: rangeZ, onChange: save });
 * return <Form.Form {...form}><Form.TextField path="name" /></Form.Form>;
 */
export const use = <Z extends z.ZodType>(params: UseParams<Z>): UseReturn<Z> => {
  const { schema, mode = "normal", onChange, onHasTouched, store } = params;
  const ref = useInitializerRef<State<Z>>(
    () =>
      new State<Z>(
        params.store !== undefined ? params.store : createLocalStore(params.values),
        schema,
      ),
  );
  const onChangeRef = useSyncedRef(onChange);
  const handleError = Status.useErrorHandler();
  const onHasTouchedRef = useSyncedRef(onHasTouched);
  // The changes of the open batch, settled together when it closes.
  const batchRef = useRef<Change[] | null>(null);

  const finish = (changes: Change[], prevHasTouched: boolean) => {
    changes.forEach(({ path, prev, silent }) => {
      if (!silent)
        onChangeRef.current?.({ values: ref.current.values, path, prev, valid: true });
    });
    ref.current.notify();
    const hasTouched = ref.current.hasBeenTouched;
    if (hasTouched !== prevHasTouched) onHasTouchedRef.current?.(hasTouched);
  };

  const settle = (changes: Change[], prevHasTouched: boolean) => {
    try {
      ref.current.validate();
    } catch (_) {
      return handleError(async () => {
        await ref.current.validateAsync();
        finish(changes, prevHasTouched);
      }, "Failed to validate form");
    }
    finish(changes, prevHasTouched);
  };

  const commit = (path: string, write: () => void, silent = false) => {
    const change = {
      path,
      prev: deep.get(ref.current.values, path, { optional: true }),
      silent,
    };
    const pending = batchRef.current;
    if (pending != null) {
      write();
      pending.push(change);
      return;
    }
    const prevHasTouched = ref.current.hasBeenTouched;
    write();
    settle([change], prevHasTouched);
  };

  const batch = (fn: () => void) => {
    if (batchRef.current != null) {
      fn();
      return;
    }
    const pending: Change[] = [];
    const prevHasTouched = ref.current.hasBeenTouched;
    batchRef.current = pending;
    const close = () => {
      batchRef.current = null;
      if (pending.length > 0) settle(pending, prevHasTouched);
    };
    try {
      ref.current.batch(fn);
    } catch (e) {
      close();
      throw errors.fromUnknown(e);
    }
    close();
  };

  const setCurrentStateAsInitialValues = () => {
    ref.current.setCurrentStateAsInitialValues();
    onHasTouchedRef.current?.(false);
  };

  const bind: BindFunc = (handleChange: Listener): destructor.Destructor =>
    ref.current.onChange(handleChange);

  const get: typeof State.prototype.getState = (<V = unknown>(
    path: string,
    opts?: GetOptions,
  ): FieldState<V> | null =>
    ref.current.getState(path, opts)) as typeof State.prototype.getState;

  const remove = (path: string) => commit(path, () => ref.current.remove(path));

  const insert: InsertFunc = (path, value, after) =>
    commit(path, () => ref.current.insert(path, value, after));

  const move: MoveFunc = (path, key, after) =>
    commit(path, () => ref.current.move(path, key, after));

  const reset = (values?: z.infer<Z>) => {
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
  };

  const validateAsync = async (path?: string): Promise<boolean> => {
    const valid = await ref.current.validateAsync(true, path);
    ref.current.notify();
    return valid;
  };

  const validate = (path?: string): boolean => {
    const valid = ref.current.validate(true, path);
    ref.current.notify();
    return valid;
  };

  const set: SetFunc = (path, value, options): void => {
    const { notifyOnChange = true, markTouched = true } = options ?? {};
    commit(
      path,
      () => ref.current.setValue(path, value, { markTouched }),
      !notifyOnChange,
    );
  };

  const has = (path: string): boolean => deep.has(ref.current.values, path);

  const setStatus = (path: string, status: Status.Crude): void => {
    const prevHasTouched = ref.current.hasBeenTouched;
    ref.current.setStatus(path, status);
    ref.current.notify();
    const hasTouched = ref.current.hasBeenTouched;
    if (hasTouched !== prevHasTouched) onHasTouchedRef.current?.(hasTouched);
  };

  const clearStatuses = () => {
    ref.current.clearStatus();
    ref.current.notify();
  };

  // A store of another identity replaces the one the form reads, as for another record.
  useLayoutEffect(() => {
    const prevHasTouched = ref.current.hasBeenTouched;
    if (store !== undefined && ref.current.setStore(store)) {
      ref.current.notify();
      if (prevHasTouched) onHasTouchedRef.current?.(false);
    }
    return ref.current.listen();
  }, [ref, store, onHasTouchedRef]);

  return {
    bind,
    set,
    get,
    mode,
    validate,
    validateAsync,
    value: () => ref.current.values,
    has,
    remove,
    insert,
    move,
    batch,
    setStatus,
    clearStatuses,
    reset,
    setCurrentStateAsInitialValues,
    getStatuses: () => ref.current.getStatuses(),
  };
};
