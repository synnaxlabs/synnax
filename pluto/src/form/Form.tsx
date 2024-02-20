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
} from "react";

import { Case, deep, shallowCopy, type Destructor, toArray } from "@synnaxlabs/x";
import { type z } from "zod";

import { useSyncedRef } from "@/hooks/ref";
import { Input } from "@/input";
import { type status } from "@/status/aether";
import { componentRenderProp, type RenderProp } from "@/util/renderProp";

/** Props for the @see useField hook */
export interface UseFieldProps {
  path: string;
  allowNull?: false;
}

export interface UseNullableFieldProps {
  path: string;
  allowNull: true;
}

/** Return type for the @see useField hook */
export interface UseFieldReturn<I extends Input.Value, O extends Input.Value = I>
  extends FieldState<I> {
  onChange: (value: O) => void;
  status: status.CrudeSpec;
}

interface UseField {
  <I extends Input.Value, O extends Input.Value = I>(
    props: UseFieldProps,
  ): UseFieldReturn<I, O>;
  <I extends Input.Value, O extends Input.Value = I>(
    props: UseNullableFieldProps,
  ): UseFieldReturn<I, O> | null;
}

export const useField = (<I extends Input.Value, O extends Input.Value = I>({
  path,
  allowNull = false,
}: UseFieldProps): UseFieldReturn<I, O> | null => {
  const { bind, get, set } = useContext();
  const [state, setState] = useState<FieldState<I> | null>(get<I>(path, allowNull));
  useLayoutEffect(() => {
    setState(get<I>(path, allowNull));
    return bind(path, setState, false);
  }, [path, bind, setState]);
  if (state == null && !allowNull) throw new Error("Field state is null");
  return {
    onChange: useCallback((value: O) => set(path, value), [path, set]),
    ...(state as FieldState<I>),
  };
}) as UseField;

export const useFieldListener = <I extends Input.Value>(
  path: string,
  callback: (state: FieldState<I>, extra: FormContextValue) => void,
): void => {
  const ctx = useContext();
  useLayoutEffect(
    () => ctx.bind<I>(path, (fs) => callback(fs, ctx), false),
    [path, ctx],
  );
};

export interface UseChildFieldValuesProps {
  path: string;
  allowNull?: false;
}

export interface UseNullableChildFieldValuesProps {
  path: string;
  allowNull: true;
}

export interface UseChildFieldValues {
  <V extends unknown = unknown>(props: UseChildFieldValuesProps): V;
  <V extends unknown = unknown>(props: UseNullableChildFieldValuesProps): V | null;
}

export const useChildFieldValues = (<V extends unknown = unknown>({
  path,
  allowNull = false,
}: UseChildFieldValuesProps): V | null => {
  const { bind, get } = useContext();
  const [state, setState] = useState<FieldState<V> | null>(get<V>(path, allowNull));
  useLayoutEffect(() => {
    setState(get<V>(path, allowNull));
    return bind<V>(
      path,
      (fs) => setState({ ...fs, value: shallowCopy(fs.value) }),
      true,
    );
  }, [path, bind, get]);
  if (state == null && !allowNull) throw new Error("Field state is null");
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
  const [state, setState] = useState<V[]>(get<V[]>(path, false).value);

  useLayoutEffect(() => {
    setState(get<V[]>(path, false).value);
    return bind<V[]>(
      path,
      (fs) => setState(shallowCopy<V[]>(fs.value)),
      updateOnChildren,
    );
  }, [path, bind, get, setState]);

  const push = useCallback(
    (value: V | V[]) => {
      const copy = shallowCopy(get<V[]>(path, false).value);
      copy.push(...toArray(value));
      set(path, copy);
    },
    [path, get, set],
  );

  const remove = useCallback(
    (index: number | number[]) => {
      const copy = shallowCopy(get<V[]>(path, false).value);
      const indices = toArray(index).sort((a, b) => b - a);
      indices.forEach((i) => copy.splice(i, 1));
      set(path, copy);
    },
    [path, state, get],
  );

  return { value: state, push, remove };
};

export interface FieldProps<
  I extends Input.Value = string | number,
  O extends Input.Value = I,
> extends UseFieldProps,
    Omit<Input.ItemProps, "children"> {
  children?: RenderProp<Input.Control<I, O>>;
  padHelpText?: boolean;
  visible?: boolean | ((state: FieldState<I>) => boolean);
}

const defaultInput = componentRenderProp(Input.Text);

export const Field = <
  I extends Input.Value = string | number,
  O extends Input.Value = I,
>({
  path,
  children = defaultInput as unknown as RenderProp<Input.Control<I, O>>,
  label,
  padHelpText = true,
  visible = true,
  ...props
}: FieldProps<I, O>): ReactElement | null => {
  const field = useField<I, O>({ path });
  if (path == null || path.length === 0) throw new Error("Path is required");
  if (label == null) label = Case.capitalize(deep.element(path, -1));
  visible = typeof visible === "function" ? visible(field) : visible;
  if (!visible) return null;
  const helpText = field.touched ? field.status.message : "";
  return (
    <Input.Item padHelpText={padHelpText} helpText={helpText} label={label} {...props}>
      {children(field)}
    </Input.Item>
  );
};

type Listener<V = unknown> = (state: FieldState<V>) => void;

interface FieldState<V = unknown> {
  value: V;
  status: status.CrudeSpec;
  touched: boolean;
}

interface Get {
  <V extends Input.Value>(path: string, allowNull: false): FieldState<V>;
  <V extends Input.Value>(path: string, allowNull: true): FieldState<V> | null;
}

export interface FormContextValue<Z extends z.ZodTypeAny = z.ZodTypeAny> {
  bind: <V>(
    path: string,
    callback: Listener<V>,
    listenToChildren: boolean,
  ) => Destructor;
  set: (path: string, value: unknown) => void;
  get: Get;
  value: () => z.output<Z>;
  validate: (path?: string) => boolean;
  has: (path: string) => boolean;
}

export const Context = createContext<FormContextValue>({
  bind: () => () => {},
  set: () => {},
  get: <V extends any = unknown>(): FieldState<V> => ({
    value: undefined as unknown as V,
    status: { key: "", variant: "success", message: "" },
    touched: false,
  }),
  validate: () => false,
  value: () => ({}),
  has: () => false,
});

export const useContext = (): FormContextValue => reactUseContext(Context);

const NO_ERROR_STATUS = (path: string): status.CrudeSpec => ({
  key: path,
  variant: "success",
  message: "",
});

interface UseRef<Z extends z.ZodTypeAny> {
  state: z.output<Z>;
  status: Map<string, status.CrudeSpec>;
  touched: Set<string>;
  listeners: Map<string, Set<Listener>>;
  parentListeners: Map<string, Set<Listener>>;
}

export interface UseProps<Z extends z.ZodTypeAny> {
  schema: Z;
  initialValues: z.output<Z>;
}

export const use = <Z extends z.ZodTypeAny>({
  initialValues,
  schema,
}: UseProps<Z>): FormContextValue<Z> => {
  const ref = useRef<UseRef<Z>>({
    state: initialValues,
    status: new Map(),
    touched: new Set(),
    listeners: new Map(),
    parentListeners: new Map(),
  });
  const schemaRef = useSyncedRef(schema);

  const bind = useCallback(
    <V extends any = unknown>(
      path: string,
      callback: Listener<V>,
      listenToChildren: boolean,
    ): Destructor => {
      const { parentListeners, listeners } = ref.current;
      const lis = listenToChildren ? parentListeners : listeners;
      if (!lis.has(path)) lis.set(path, new Set());
      lis.get(path)?.add(callback as Listener);
      return () => lis.get(path)?.delete(callback as Listener);
    },
    [],
  );

  const get = useCallback(
    <V extends any = unknown>(
      path: string,
      allowNull: boolean,
    ): FieldState<V> | null => {
      const { state, status, touched } = ref.current;
      const value = deep.get(state, path, allowNull);
      if (value == null) return null;
      return {
        value: value as V,
        status: status.get(path) ?? NO_ERROR_STATUS(path),
        touched: touched.has(path),
      };
    },
    [],
  ) as Get;

  const validate = useCallback((path?: string): boolean => {
    const { state, status, listeners } = ref.current;
    console.log(state);
    const result = schemaRef.current.safeParse(state);
    console.log(result.success);
    // if (path == null) status.clear();
    // else status.delete(path);
    if (result.success) {
      const keys = Array.from(status.keys());
      status.clear();
      keys.forEach((key) => {
        const fs = get(key, true);
        if (fs == null) return;
        listeners.get(key)?.forEach((l) => l(fs));
      });
      return true;
    }
    const issueKeys = new Set(result.error.issues.map((i) => i.path.join(".")));
    result.error.issues.forEach((issue) => {
      const key = issue.path.join(".");
      status.set(key, {
        key,
        variant: "error",
        message: issue.message,
      });
      const fs = get(key, false);
      listeners.get(key)?.forEach((l) => l(fs));
    });
    status.forEach((_, key) => {
      if (!issueKeys.has(key)) {
        status.delete(key);
        const fs = get(key, false);
        listeners.get(key)?.forEach((l) => l(fs));
      }
    });
    return false;
  }, []);

  const set = useCallback((path: string, value: unknown): void => {
    const { state, touched, listeners, parentListeners } = ref.current;
    touched.add(path);
    deep.set(state, path, value);
    validate();
    listeners.get(path)?.forEach((l) => l(get(path, false)));
    parentListeners.forEach((lis, key) => {
      if (path.startsWith(key)) {
        lis.forEach((l) => l(get(key, false)));
      }
    });
  }, []);

  const has = useCallback((path: string): boolean => {
    const { state } = ref.current;
    return deep.has(state, path);
  }, []);

  return useMemo(
    (): FormContextValue<Z> => ({
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

export const Form = (props: PropsWithChildren<FormContextValue>): ReactElement => {
  return <Context.Provider value={props}>{props.children}</Context.Provider>;
};
