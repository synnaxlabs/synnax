// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/* eslint-disable @typescript-eslint/no-unnecessary-type-constraint */
import {
  type ReactElement,
  type PropsWithChildren,
  createContext,
  useContext as reactUseContext,
  useState,
  useLayoutEffect,
  useRef,
  useMemo,
  useCallback,
  useEffect,
} from "react";

import { caseconv, deep, shallowCopy, type Destructor, toArray } from "@synnaxlabs/x";
import { type z } from "zod";

import { useSyncedRef } from "@/hooks/ref";
import { Input } from "@/input";
import { type status } from "@/status/aether";
import { componentRenderProp, type RenderProp } from "@/util/renderProp";

/** Props for the @see useField hook */
export interface UseFieldProps<I, O = I> {
  path: string;
  optional?: false;
  onChange?: (value: O, extra: ContextValue) => void;
  defaultValue?: O;
}

export interface UseNullableFieldProps<I, O = I>
  extends Omit<UseFieldProps<I, O>, "optional"> {
  optional: true;
}

/** Return type for the @see useField hook */
export interface UseFieldReturn<I extends Input.Value, O extends Input.Value = I>
  extends FieldState<I> {
  onChange: (value: O) => void;
  status: status.CrudeSpec;
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
  optional: propsOptional,
  onChange,
  defaultValue,
}: UseFieldProps<I, O>): UseFieldReturn<I, O> | null => {
  const ctx = useContext();
  const { get, bind, set } = ctx;
  const optional = defaultValue != null || (propsOptional ?? false);

  const [state, setState] = useState<FieldState<I> | null>(get<I>({ path, optional }));

  useLayoutEffect(() => {
    setState(get<I>({ path, optional }));
    return bind({ path, listener: setState, listenToChildren: false });
  }, [path, bind, setState]);

  const handleChange = useCallback(
    (value: O) => {
      onChange?.(value, ctx);
      set({ path, value });
    },
    [path, set, onChange],
  );

  if (state == null) {
    if (defaultValue != null) set({ path, value: defaultValue });
    if (!optional) throw new Error(`Field state is null: ${path}`);
    return null;
  }

  return { onChange: handleChange, ...state };
}) as UseField;

export const useFieldListener = <I extends Input.Value>(
  path: string,
  callback: (state: FieldState<I>, extra: ContextValue) => void,
): void => {
  const ctx = useContext();
  useLayoutEffect(
    () =>
      ctx.bind<I>({
        path,
        listener: (fs) => callback(fs, ctx),
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
  const [state, setState] = useState<FieldState<V> | null>(get<V>({ path, optional }));
  useLayoutEffect(() => {
    setState(get<V>({ path, optional }));
    return bind<V>({
      path,
      listener: (fs) => setState({ ...fs, value: shallowCopy(fs.value) }),
      listenToChildren: true,
    });
  }, [path, bind, get]);
  if (state == null && !optional) throw new Error("Field state is null");
  return state?.value ?? null;
}) as UseChildFieldValues;

export interface UseFieldArrayProps {
  path: string;
  updateOnChildren?: boolean;
}

export interface UseFieldArrayReturn<V extends unknown> {
  value: V[];
  push: (value: V | V[]) => void;
  remove: (index: number | number[]) => void;
}

export const useFieldArray = <V extends unknown = unknown>({
  path,
  updateOnChildren = false,
}: UseFieldArrayProps): UseFieldArrayReturn<V> => {
  const { bind, get, set } = useContext();
  const [state, setState] = useState<V[]>(get<V[]>({ path, optional: false }).value);

  useLayoutEffect(() => {
    setState(get<V[]>({ path, optional: false }).value);
    return bind<V[]>({
      path,
      listener: (fs) => setState(shallowCopy<V[]>(fs.value)),
      listenToChildren: updateOnChildren,
    });
  }, [path, bind, get, setState]);

  const push = useCallback(
    (value: V | V[]) => {
      const copy = shallowCopy(get<V[]>({ path }).value);
      copy.push(...toArray(value));
      set({ path, value: copy });
    },
    [path, get, set],
  );

  const remove = useCallback(
    (index: number | number[]) => {
      const copy = shallowCopy(get<V[]>({ path, optional: false }).value);
      const indices = toArray(index).sort((a, b) => b - a);
      indices.forEach((i) => copy.splice(i, 1));
      set({ path, value: copy });
    },
    [path, state, get],
  );

  return { value: state, push, remove };
};

export interface FieldProps<
  I extends Input.Value = string | number,
  O extends Input.Value = I,
> extends UseFieldProps<I, O>,
    Omit<Input.ItemProps, "children" | "onChange" | "defaultValue"> {
  children?: RenderProp<Input.Control<I, O>>;
  padHelpText?: boolean;
  visible?: boolean | ((state: FieldState<I>) => boolean);
  hideIfNull?: boolean;
}

const defaultInput = componentRenderProp(Input.Text);

export type FieldT<I extends Input.Value, O extends Input.Value = I> = (
  props: FieldProps<I, O>,
) => ReactElement | null;

export const Field = <
  I extends Input.Value = string | number,
  O extends Input.Value = I,
>({
  path,
  children = defaultInput as unknown as RenderProp<Input.Control<I, O>>,
  label,
  padHelpText = true,
  visible = true,
  hideIfNull = false,
  defaultValue,
  onChange,
  ...props
}: FieldProps<I, O>): ReactElement | null => {
  const field = useField<I, O>({
    path,
    optional: hideIfNull as true,
    onChange,
    defaultValue,
  });
  if (field == null) return null;
  if (path == null) throw new Error("No path provided to Form Field");
  if (label == null) label = caseconv.capitalize(deep.element(path, -1));
  visible = typeof visible === "function" ? visible(field) : visible;
  if (!visible) return null;
  const helpText = field.touched ? field.status.message : "";
  const { onChange: fieldOnChange, value } = field;
  return (
    <Input.Item
      padHelpText={padHelpText}
      helpText={helpText}
      label={label}
      required={field.required}
      {...props}
    >
      {children({ onChange: fieldOnChange, value })}
    </Input.Item>
  );
};

type Listener<V = unknown> = (state: FieldState<V>) => void;

export interface FieldState<V = unknown> {
  value: V;
  status: status.CrudeSpec;
  touched: boolean;
  required: boolean;
}

interface RequiredGetProps {
  path: string;
  optional?: boolean;
}

interface OptionalGetProps {
  path: string;
  optional: true;
}

type GetProps = RequiredGetProps | OptionalGetProps;

interface GetFunc {
  <V extends Input.Value>(props: RequiredGetProps): FieldState<V>;
  <V extends Input.Value>(props: OptionalGetProps): FieldState<V> | null;
}

interface SetProps {
  path: string;
  value: unknown;
}

type SetFunc = (props: SetProps) => void;

interface BindProps<V = unknown> {
  path: string;
  listener: Listener<V>;
  listenToChildren?: boolean;
}

type BindFunc = <V = unknown>(props: BindProps<V>) => Destructor;

export interface ContextValue<
  T extends z.ZodRawShape = z.ZodRawShape,
  Z extends z.ZodObject<T> = z.ZodObject<T>,
> {
  bind: BindFunc;
  set: SetFunc;
  get: GetFunc;
  value: () => z.output<Z>;
  validate: (path?: string) => boolean;
  has: (path: string) => boolean;
}

export const Context = createContext<ContextValue>({
  bind: () => () => {},
  set: () => {},
  get: <V extends any = unknown>(): FieldState<V> => ({
    value: undefined as unknown as V,
    status: { key: "", variant: "success", message: "" },
    touched: false,
    required: false,
  }),
  validate: () => false,
  value: () => ({}),
  has: () => false,
});

export const useContext = (): ContextValue => reactUseContext(Context);

const NO_ERROR_STATUS = (path: string): status.CrudeSpec => ({
  key: path,
  variant: "success",
  message: "",
});

interface UseRef<T extends z.ZodRawShape, Z extends z.ZodObject<T> = z.ZodObject<T>> {
  state: z.output<Z>;
  status: Map<string, status.CrudeSpec>;
  touched: Set<string>;
  listeners: Map<string, Set<Listener>>;
  parentListeners: Map<string, Set<Listener>>;
}

export interface UseProps<
  T extends z.ZodRawShape = z.ZodRawShape,
  Z extends z.ZodObject<T> = z.ZodObject<T>,
> {
  values: z.output<Z>;
  sync?: boolean;
  onChange?: (values: z.output<Z>) => void;
  schema?: Z;
}

export type UseReturn<
  T extends z.ZodRawShape = z.ZodRawShape,
  Z extends z.ZodObject<T> = z.ZodObject<T>,
> = ContextValue<T, Z>;

const fieldIsRequired = <T extends z.ZodRawShape, Z extends z.ZodObject<T>>(
  schema: Z,
  path: string,
): boolean => {
  const field = deep.get<T>(schema.shape, path, true) as z.ZodTypeAny;
  if (field == null) return false;
  return !field.isOptional();
};

export const use = <
  T extends z.ZodRawShape = z.ZodRawShape,
  Z extends z.ZodObject<T> = z.ZodObject<T>,
>({
  values,
  sync = false,
  schema,
  onChange,
}: UseProps<T, Z>): UseReturn<T, Z> => {
  const ref = useRef<UseRef<T, Z>>({
    state: values,
    status: new Map(),
    touched: new Set(),
    listeners: new Map(),
    parentListeners: new Map(),
  });
  const schemaRef = useSyncedRef(schema);
  const onChangeRef = useSyncedRef(onChange);

  const bind: BindFunc = useCallback(
    <V extends any = unknown>({
      path,
      listener: callback,
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
    <V extends any = unknown>({ path, optional }: GetProps): FieldState<V> | null => {
      const { state, status, touched } = ref.current;
      const value = deep.get(state, path, optional);
      if (value == null) return null;
      return {
        value: value as V,
        status: status.get(path) ?? NO_ERROR_STATUS(path),
        touched: touched.has(path),
        required:
          schemaRef.current != null ? fieldIsRequired(schemaRef.current, path) : false,
      };
    },
    [],
  ) as GetFunc;

  const validate = useCallback((path?: string): boolean => {
    if (schemaRef.current == null) return true;
    const { state, status, listeners } = ref.current;
    const result = schemaRef.current.safeParse(state);
    // if (path == null) status.clear();
    // else status.delete(path);
    if (result.success) {
      const keys = Array.from(status.keys());
      status.clear();
      keys.forEach((p) => {
        const fs = get({ path: p });
        if (fs == null) return;
        listeners.get(p)?.forEach((l) => l(fs));
      });
      return true;
    }
    const issueKeys = new Set(result.error.issues.map((i) => i.path.join(".")));
    result.error.issues.forEach((issue) => {
      const issuePath = issue.path.join(".");
      status.set(issuePath, {
        key: issuePath,
        variant: "error",
        message: issue.message,
      });
      const fs = get({ path: issuePath });
      listeners.get(issuePath)?.forEach((l) => l(fs));
    });
    status.forEach((_, subPath) => {
      if (!issueKeys.has(subPath)) {
        status.delete(subPath);
        const fs = get({ path: subPath });
        listeners.get(subPath)?.forEach((l) => l(fs));
      }
    });
    return false;
  }, []);

  const set: SetFunc = useCallback(({ path, value }): void => {
    const { state, touched, listeners, parentListeners } = ref.current;
    touched.add(path);
    if (path.length === 0) ref.current.state = value as z.output<Z>;
    else deep.set(state, path, value);
    validate();
    listeners.get(path)?.forEach((l) => l(get({ path })));
    parentListeners.forEach((lis, listPath) => {
      if (path.startsWith(listPath)) {
        const v = get({ path: listPath });
        lis.forEach((l) => l(v));
      }
    });
    onChangeRef.current?.(ref.current.state);
  }, []);

  const has = useCallback((path: string): boolean => {
    const { state } = ref.current;
    return deep.has(state, path);
  }, []);

  useEffect(() => {
    if (!sync) return;
    const { listeners } = ref.current;
    ref.current.state = values;
    listeners.forEach((lis, p) => {
      const v = get({ path: p });
      if (v == null) return;
      lis.forEach((l) => l(v));
    });
  }, [sync, values]);

  return useMemo(
    (): ContextValue<T, Z> => ({
      bind,
      set,
      get,
      validate,
      value: () => ref.current.state,
      has,
    }),
    [bind, set, get, validate],
  );
};

export const Form = (props: PropsWithChildren<ContextValue>): ReactElement => {
  return <Context.Provider value={props}>{props.children}</Context.Provider>;
};
