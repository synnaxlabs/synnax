// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/* eslint-disable @typescript-eslint/no-unnecessary-type-constraint */
import { type compare, type Destructor, shallowCopy, toArray } from "@synnaxlabs/x";
import { deep } from "@synnaxlabs/x/deep";
import { zodutil } from "@synnaxlabs/x/zodutil";
import {
  createContext,
  type PropsWithChildren,
  type ReactElement,
  use as reactUse,
  useCallback,
  useEffect,
  useMemo,
  useState,
} from "react";
import { z } from "zod";

import { useInitializerRef, useSyncedRef } from "@/hooks/ref";
import { type Input } from "@/input";
import { state } from "@/state";
import { type status } from "@/status/aether";

/** Props for the @link useField hook */
export interface UseFieldProps<I, O = I> {
  path: string;
  optional?: false;
  onChange?: (value: O, extra: ContextValue & { path: string }) => void;
}

export interface UseNullableFieldProps<I, O = I>
  extends Omit<UseFieldProps<I, O>, "optional"> {
  optional: true;
}

/** Return type for the @link useField hook */
export interface UseFieldReturn<I extends Input.Value, O extends Input.Value = I>
  extends FieldState<I> {
  onChange: (value: O) => void;
  setStatus: (status: status.CrudeSpec) => void;
  status: status.CrudeSpec;
  variant?: Input.Variant;
}

interface UseField {
  <I extends Input.Value, O extends Input.Value = I>(
    props: UseFieldProps<I, O>,
  ): UseFieldReturn<I, O>;
  <I extends Input.Value, O extends Input.Value = I>(
    props: UseNullableFieldProps<I, O>,
  ): UseFieldReturn<I, O> | null;
}

/**
 * Hook for managing a particular field in a form.
 *
 * @param props - The props for the hook
 * @param props.path - The path to the field in the form.
 */
export const useField = (<I extends Input.Value, O extends Input.Value = I>({
  path,
  optional = false,
  onChange,
}: UseFieldProps<I, O>): UseFieldReturn<I, O> | null => {
  const ctx = useContext();
  const { get, bind, set, setStatus } = ctx;

  const [state, setState] = useState<FieldState<I> | null>(get<I>(path, { optional }));

  useEffect(() => {
    setState(get<I>(path, { optional }));
    return bind({
      path,
      onChange: setState,
      listenToChildren: false,
    });
  }, [path, onChange, bind, get]);

  const handleChange = useCallback(
    (value: O) => {
      onChange?.(value, { ...ctx, path });
      set(path, value);
    },
    [path, set, onChange],
  );

  const handleSetStatus = useCallback(
    (status: status.CrudeSpec) => setStatus(path, status),
    [path, setStatus],
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
  <I extends Input.Value, O extends Input.Value = I, Z extends z.ZodTypeAny = any>(
    path: string,
    optional?: false,
    ctx?: ContextValue<Z>,
  ): O;
  <I extends Input.Value, O extends Input.Value = I, Z extends z.ZodTypeAny = any>(
    path: string,
    optional: true,
    ctx?: ContextValue<Z>,
  ): O | null;
}

export interface UseFieldState {
  <
    I extends Input.Value,
    O extends Input.Value = I,
    Z extends z.ZodTypeAny = z.ZodTypeAny,
  >(
    path: string,
    optional?: false,
    ctx?: ContextValue<Z>,
  ): FieldState<O>;
  <
    I extends Input.Value,
    O extends Input.Value = I,
    Z extends z.ZodTypeAny = z.ZodTypeAny,
  >(
    path: string,
    optional: true,
    ctx?: ContextValue<Z>,
  ): FieldState<O> | null;
}

export const useFieldState = <I extends Input.Value, O extends Input.Value = I>(
  path: string,
  optional: boolean = false,
  ctx?: ContextValue,
): FieldState<O> | null => {
  const { get, bind } = useContext(ctx);
  const [, setChangeTrigger] = useState(0);
  useEffect(() => {
    setChangeTrigger((prev) => prev + 1);
    return bind<O>({ path, onChange: () => setChangeTrigger((p) => p + 1) });
  }, [path, bind]);
  return get<O>(path, { optional }) ?? null;
};

export const useFieldValue = (<I extends Input.Value, O extends Input.Value = I>(
  path: string,
  optional: boolean = false,
  ctx?: ContextValue,
): O | null => {
  const { get, bind } = useContext(ctx);
  const [, setChangeTrigger] = useState(0);
  useEffect(() => {
    setChangeTrigger((prev) => prev + 1);
    return bind<O>({ path, onChange: () => setChangeTrigger((p) => p + 1) });
  }, [path, bind]);
  return get<O>(path, { optional })?.value ?? null;
}) as UseFieldValue;

export const useFieldValid = (path: string): boolean =>
  useFieldState(path, true)?.status?.variant === "success";

export interface UseFieldListenerProps<
  I extends Input.Value,
  Z extends z.ZodTypeAny = z.ZodTypeAny,
> {
  ctx?: ContextValue<Z>;
  path: string;
  onChange: (state: FieldState<I>, extra: ContextValue<Z>) => void;
}

export const useFieldListener = <
  I extends Input.Value,
  Z extends z.ZodTypeAny = z.ZodTypeAny,
>({
  path,
  ctx: override,
  onChange,
}: UseFieldListenerProps<I, Z>): void => {
  const ctx = useContext(override);
  useEffect(
    () =>
      ctx.bind<I>({
        path,
        onChange: (fs) => onChange(fs, ctx),
        listenToChildren: false,
      }),
    [path, ctx],
  );
};

export interface UseChildFieldValuesProps {
  path: string;
  optional?: false;
}

export interface UseNullableChildFieldValuesProps {
  path: string;
  optional: true;
}

export interface UseChildFieldValues {
  <V extends unknown = unknown>(props: UseChildFieldValuesProps): V;
  <V extends unknown = unknown>(props: UseNullableChildFieldValuesProps): V | null;
}

export const useChildFieldValues = (<V extends unknown = unknown>({
  path,
  optional = false,
}: UseChildFieldValuesProps): V | null => {
  const { bind, get } = useContext();
  const [state, setState] = useState<FieldState<V> | null>(get<V>(path, { optional }));
  useEffect(() => {
    setState(get<V>(path, { optional }));
    return bind<V>({
      path,
      onChange: (fs) => setState({ ...fs, value: shallowCopy(fs.value) }),
      listenToChildren: true,
    });
  }, [path, bind, get]);
  if (state == null && !optional) throw new Error("Field state is null");
  return state?.value ?? null;
}) as UseChildFieldValues;

export interface UseFieldArrayProps {
  path: string;
  updateOnChildren?: boolean;
  ctx?: ContextValue<any>;
}

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
    copy.splice(start, 0, ...toArray(value));
    ctx.set(path, copy, { validateChildren: false });
  },
  push: (value, sort) => {
    const copy = shallowCopy(ctx.get<V[]>(path).value);
    copy.push(...toArray(value));
    if (sort != null) copy.sort(sort);
    ctx.set(path, copy, { validateChildren: false });
  },
  remove: (index) => {
    const val = ctx.get<V[]>(path).value;
    const indices = new Set(toArray(index));
    ctx.set(
      path,
      val.filter((_, i) => !indices.has(i)),
    );
  },
  keepOnly: (index) => {
    const val = ctx.get<V[]>(path).value;
    const indices = new Set(toArray(index));
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

export const useFieldArray = <V extends unknown = unknown>({
  path,
  updateOnChildren = false,
  ctx: pCtx,
}: UseFieldArrayProps): UseFieldArrayReturn<V> => {
  const ctx = useContext(pCtx);
  const { bind, get } = ctx;
  const [fState, setFState] = useState<V[]>(get<V[]>(path).value);
  useEffect(() => {
    setFState(get<V[]>(path).value);
    return bind<V[]>({
      path,
      onChange: (fs) => {
        setFState(shallowCopy<V[]>(fs.value));
      },
      listenToChildren: updateOnChildren,
    });
  }, [path, bind, get, setFState]);
  return useMemo(
    () => ({ value: fState, ...fieldArrayUtils<V>(ctx, path) }),
    [fState, ctx, path],
  );
};

export interface Listener<V = unknown> {
  (state: FieldState<V>): void;
}

export interface FieldState<V = unknown> {
  value: V;
  status: status.CrudeSpec;
  touched: boolean;
  required: boolean;
}

interface RequiredGetOptions<O extends boolean | undefined = boolean | undefined> {
  optional?: O;
}

interface OptionalGetOptions {
  optional?: true;
}

type GetOptions = RequiredGetOptions | OptionalGetOptions;

interface GetFunc {
  <V extends Input.Value>(
    path: string,
    opts: RequiredGetOptions<true>,
  ): FieldState<V> | null;
  <V extends Input.Value>(
    path: string,
    opts?: RequiredGetOptions<boolean | undefined>,
  ): FieldState<V>;
}

interface RemoveFunc {
  (path: string): void;
}

interface SetOptions {
  validateChildren?: boolean;
}

interface SetFunc {
  (path: string, value: unknown, opts?: SetOptions): void;
}

interface BindProps<V = unknown> {
  path: string;
  onChange: Listener<V>;
  listenToChildren?: boolean;
}

interface BindFunc {
  <V = unknown>(props: BindProps<V>): Destructor;
}

type Mode = "normal" | "preview";

export interface ContextValue<Z extends z.ZodTypeAny = z.ZodTypeAny> {
  mode: Mode;
  bind: BindFunc;
  set: SetFunc;
  reset: (values?: z.output<Z>) => void;
  get: GetFunc;
  remove: RemoveFunc;
  value: () => z.output<Z>;
  validate: (path?: string) => boolean;
  validateAsync: (path?: string) => Promise<boolean>;
  has: (path: string) => boolean;
  setStatus: (path: string, status: status.CrudeSpec) => void;
  clearStatuses: () => void;
  setCurrentStateAsInitialValues: () => void;
}

const Context = createContext<ContextValue>({
  mode: "normal",
  bind: () => () => {},
  set: () => {},
  reset: () => {},
  remove: () => {},
  get: <V extends any = unknown>(): FieldState<V> => ({
    value: undefined as V,
    status: { key: "", variant: "success", message: "" },
    touched: false,
    required: false,
  }),
  validate: () => false,
  validateAsync: () => Promise.resolve(false),
  value: () => ({}),
  has: () => false,
  setStatus: () => {},
  clearStatuses: () => {},
  setCurrentStateAsInitialValues: () => {},
});

export const useContext = <Z extends z.ZodTypeAny = z.ZodTypeAny>(
  override?: ContextValue<Z>,
): ContextValue<Z> => {
  const internal = reactUse(Context);
  return override ?? (internal as unknown as ContextValue<Z>);
};

const NO_ERROR_STATUS = (path: string): status.CrudeSpec => ({
  key: path,
  variant: "success",
  message: "",
});

interface UseRef<Z extends z.ZodTypeAny> {
  state: z.output<Z>;
  statuses: Map<string, status.CrudeSpec>;
  touched: Set<string>;
  listeners: Map<string, Set<Listener>>;
  parentListeners: Map<string, Set<Listener>>;
}

export interface OnChangeProps<Z extends z.ZodTypeAny> {
  /** The values in the form AFTER the change. */
  values: z.output<Z>;
  /** The path that was changed. */
  path: string;
  /** The previous value at the path. */
  prev: unknown;
  /** Whether validation succeeded. */
  valid: boolean;
}

export interface UseProps<Z extends z.ZodTypeAny> {
  values: z.output<Z>;
  mode?: Mode;
  sync?: boolean;
  onChange?: (props: OnChangeProps<Z>) => void;
  onHasTouched?: (value: boolean) => void;
  schema?: Z;
}

export interface UseReturn<Z extends z.ZodTypeAny> extends ContextValue<Z> {}

const getVariant = (issue: z.ZodIssue): status.Variant =>
  issue.code === z.ZodIssueCode.custom &&
  issue.params != null &&
  "variant" in issue.params
    ? issue.params.variant
    : "error";

export const use = <Z extends z.ZodTypeAny>({
  values: initialValues,
  sync = false,
  schema,
  mode = "normal",
  onChange,
  onHasTouched,
}: UseProps<Z>): UseReturn<Z> => {
  const ref = useInitializerRef<UseRef<Z>>(() => ({
    state: deep.copy(initialValues),
    statuses: new Map(),
    touched: new Set(),
    listeners: new Map(),
    parentListeners: new Map(),
  }));
  const schemaRef = useSyncedRef(schema);
  const onChangeRef = useSyncedRef(onChange);
  const initialValuesRef = useSyncedRef<z.output<Z>>(initialValues);
  const onHasTouchedRef = useSyncedRef(onHasTouched);

  const setCurrentStateAsInitialValues = useCallback(() => {
    initialValuesRef.current = deep.copy(ref.current.state);
    clearTouched();
  }, []);

  const bind: BindFunc = useCallback(
    <V extends any = unknown>({
      path,
      onChange: callback,
      listenToChildren = false,
    }: BindProps<V>): Destructor => {
      const { parentListeners, listeners } = ref.current;
      const lis = listenToChildren ? parentListeners : listeners;
      if (!lis.has(path)) lis.set(path, new Set());
      lis.get(path)?.add(callback as Listener);
      return () => lis.get(path)?.delete(callback as Listener);
    },
    [],
  );

  const get: GetFunc = useCallback(
    <V extends any = unknown>(
      path: string,
      { optional }: GetOptions = { optional: false },
    ): FieldState<V> | null => {
      const { state, statuses, touched } = ref.current;
      const value = deep.get(state, path, { optional });
      if (value == null) return null;
      const fs = {
        value: value as V,
        status: statuses.get(path) ?? NO_ERROR_STATUS(path),
        touched: touched.has(path),
        required: false,
      };
      if (schemaRef.current == null) return fs;
      const schema = schemaRef.current;
      const zField = zodutil.getFieldSchema(schema, path, { optional: true });
      if (zField == null) return fs;
      fs.required = !zField.isOptional();
      return fs;
    },
    [],
  ) as GetFunc;

  const addTouched = useCallback((path: string) => {
    const { touched } = ref.current;
    const prevEmpty = touched.size === 0;
    touched.add(path);
    const currEmpty = touched.size === 0;
    if (prevEmpty !== currEmpty) onHasTouchedRef.current?.(!currEmpty);
  }, []);

  const removeTouched = useCallback((path: string) => {
    const { touched } = ref.current;
    const prevEmpty = touched.size === 0;
    touched.delete(path);
    const currEmpty = touched.size === 0;
    if (prevEmpty !== currEmpty) onHasTouchedRef.current?.(!currEmpty);
  }, []);

  const clearTouched = useCallback(() => {
    const { touched } = ref.current;
    const prevEmpty = touched.size === 0;
    touched.clear();
    const currEmpty = touched.size === 0;
    if (prevEmpty !== currEmpty) onHasTouchedRef.current?.(!currEmpty);
  }, []);

  const remove = useCallback((path: string) => {
    const { state, statuses, listeners, parentListeners } = ref.current;
    deep.remove(state, path);
    statuses.delete(path);
    removeTouched(path);
    listeners.delete(path);
    parentListeners.delete(path);
  }, []);

  const reset = useCallback((values?: z.output<Z>) => {
    const { statuses } = ref.current;
    ref.current.state = values ?? deep.copy(initialValuesRef.current);
    updateFieldValues("");
    statuses.clear();
    clearTouched();
  }, []);

  const updateFieldState = useCallback((path: string) => {
    const { listeners } = ref.current;
    const fs = get(path, { optional: true });
    if (fs == null) return;
    listeners.get(path)?.forEach((l) => l(fs));
  }, []);

  const updateFieldValues = useCallback((path: string) => {
    const { listeners, parentListeners } = ref.current;
    const fired: string[] = [];
    const lis = listeners.get(path);
    if (path == "") {
      const paths = [...listeners.keys()];
      paths.forEach((p) => {
        const v = get(p, { optional: true });
        if (v != null)
          listeners.get(p)?.forEach((l) => {
            fired.push(p);
            l(v);
          });
      });
      const parentPaths = [...parentListeners.keys()];
      parentPaths.forEach((p) => {
        const v = get(p, { optional: true });
        if (v != null)
          parentListeners.get(p)?.forEach((l) => {
            fired.push(p);
            l(v);
          });
      });
    }
    if (lis != null) {
      const fs = get(path, { optional: true });
      if (fs != null)
        lis.forEach((l) => {
          fired.push(path);
          l(fs);
        });
    }
    parentListeners.forEach((lis, lisPath) => {
      const equalOrChild = deep.pathsMatch(path, lisPath);
      if (equalOrChild) {
        const v = get(lisPath, { optional: true });
        if (v != null)
          lis.forEach((l) => {
            fired.push(`parent->${lisPath}`);
            l(v);
          });
      }
    });
  }, []);

  const processValidationResult = useCallback(
    (
      result: z.SafeParseReturnType<z.input<Z>, z.output<Z>>,
      validationPath: string = "",
      validateChildren: boolean = true,
    ): boolean => {
      const { statuses, listeners } = ref.current;

      // Parse was a complete success. No errors encountered.
      if (result.success) {
        /// Clear statuses for all fields and update relevant listeners.
        const paths = [...statuses.keys()];
        statuses.clear();
        paths.forEach((path) => updateFieldState(path));
        return true;
      }

      // The validation may still be a success if all errors are warnings.
      let success = true;
      const issueKeys = new Set(result.error.issues.map((i) => i.path.join(".")));

      let matcher = (a: string, b: string) => a === b;
      if (validateChildren) matcher = (a: string, b: string) => deep.pathsMatch(a, b);

      result.error.issues.forEach((issue) => {
        const { message } = issue;
        const issuePath = issue.path.join(".");

        // If we're only validating a sub-path and it doesn't match a particular issue,
        // skip it.
        if (!matcher(issuePath, validationPath)) return;

        const variant = getVariant(issue);
        if (variant !== "warning") success = false;

        statuses.set(issuePath, { key: issuePath, variant, message });
        addTouched(issuePath);

        let fs = get(issuePath, { optional: true });
        // If we can't find the field value, this means the user never set it, so
        // instead we just to a best effort construction of the field state. This means
        // that if the user has a field rendered for this path, the error will be displayed.
        fs ??= {
          value: undefined,
          status: statuses.get(issuePath) ?? NO_ERROR_STATUS(issuePath),
          touched: false,
          required: false,
        };
        listeners.get(issuePath)?.forEach((l) => l(fs));
      });

      // Clear any statuses that had previous validation errors, but no longer do.
      statuses.forEach((_, subPath) => {
        if (issueKeys.has(subPath)) return;
        statuses.delete(subPath);
        updateFieldState(subPath);
      });

      return success;
    },
    [],
  );

  const validate = useCallback(
    (path?: string, validateChildren?: boolean): boolean => {
      if (schemaRef.current == null) return true;
      const { state } = ref.current;
      const result = schemaRef.current.safeParse(state);
      return processValidationResult(result, path, validateChildren);
    },
    [processValidationResult],
  );

  const validateAsync = useCallback(
    async (path?: string, validateChildren?: boolean): Promise<boolean> => {
      if (schemaRef.current == null) return true;
      const { state } = ref.current;
      const result = await schemaRef.current.safeParseAsync(state);
      return processValidationResult(result, path, validateChildren);
    },
    [processValidationResult],
  );

  const set: SetFunc = useCallback((path, value, opts = {}): void => {
    const prev = deep.get(ref.current.state, path, { optional: true });
    const { validateChildren = true } = opts;
    const { state } = ref.current;
    // check if the value is the same as the initial value provided
    const initialValue = deep.get(initialValuesRef.current, path, { optional: true });
    const equalsInitial = deep.equal(initialValue, value);
    if (equalsInitial) removeTouched(path);
    else addTouched(path);
    if (path.length === 0) ref.current.state = value as z.output<Z>;
    else deep.set(state, path, value);
    updateFieldValues(path);
    void (async () => {
      let valid: boolean;
      try {
        valid = validate(path, validateChildren);
      } catch {
        valid = await validateAsync(path, validateChildren);
      }
      onChangeRef.current?.({ values: ref.current.state, path, prev, valid });
    })();
  }, []);

  const has = useCallback(
    (path: string): boolean => deep.has(ref.current.state, path),
    [],
  );

  const setStatus = useCallback((path: string, status: status.CrudeSpec): void => {
    ref.current.statuses.set(path, status);
    addTouched(path);
    updateFieldState(path);
  }, []);

  const clearStatuses = useCallback(() => {
    const { statuses } = ref.current;
    statuses.clear();
    statuses.forEach((_, path) => updateFieldState(path));
  }, []);

  useEffect(() => {
    if (!sync) return;
    const { listeners } = ref.current;
    ref.current.state = initialValues;
    listeners.forEach((lis, p) => {
      const v = get(p, { optional: true });
      if (v == null) return;
      lis.forEach((l) => l(v));
    });
  }, [sync, initialValues]);

  return useMemo(
    (): ContextValue<Z> => ({
      bind,
      set,
      get,
      mode,
      validate,
      validateAsync,
      value: () => ref.current.state,
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

export const Form = (props: PropsWithChildren<ContextValue>): ReactElement => (
  <Context value={props}>{props.children}</Context>
);
