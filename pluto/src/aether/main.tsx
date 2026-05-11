// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { deep, type SenderHandler } from "@synnaxlabs/x";
import {
  memo,
  type PropsWithChildren,
  type ReactElement,
  useCallback,
  useEffect,
  useLayoutEffect,
  useMemo,
  useRef,
  useState,
  useSyncExternalStore,
} from "react";
import { type z } from "zod";

import {
  type CallersFromSchema,
  type EmptyMethodsSchema,
  type MethodsSchema,
} from "@/aether/aether/aether";
import { type AetherMessage, type MainMessage } from "@/aether/message";
import { type Handle, type RawSetArg, Store } from "@/aether/store";
import { context } from "@/context";
import { useSyncedRef } from "@/hooks";
import { useUniqueKey } from "@/hooks/useUniqueKey";
import { useMemoPrimitiveArray } from "@/memo";
import { type state } from "@/state";
import { Worker } from "@/worker";

export type { CallersFromSchema, EmptyMethodsSchema, MethodsSchema };
export { Store } from "@/aether/store";

/** Value provided by the Aether context to descendant components. */
export interface ContextValue {
  /** The current path in the Aether component tree. */
  path: readonly string[];
  /** The single store shared across this provider's tree. */
  store: Store;
}

const DEFAULT_STORE = new Store();

const [Context, useContext] = context.create<ContextValue>({
  defaultValue: { store: DEFAULT_STORE, path: ["root"] },
  displayName: "Aether.Context",
});

export interface ProviderProps extends PropsWithChildren {
  workerKey: string;
  worker?: SenderHandler<MainMessage, AetherMessage>;
}

export const Provider = ({
  workerKey,
  worker: propsWorker,
  children,
}: ProviderProps): ReactElement => {
  const contextWorker = Worker.use<MainMessage, AetherMessage>(workerKey);
  const worker = propsWorker ?? contextWorker;

  const storeRef = useRef<Store | null>(null);
  storeRef.current ??= new Store();
  const store = storeRef.current;

  const [error, setError] = useState<Error | null>(null);
  if (error != null) throw error;

  const [ready, setReady] = useState(false);

  useEffect(() => {
    store.setWorker(worker ?? null);
    const unsubscribe = store.onError((err) => {
      setError((prev) => {
        if (prev != null) {
          console.error(
            "[aether] - received new error after error was already set, but before previous error was thrown.",
          );
          console.error(err);
        }
        return err;
      });
    });
    setReady(true);
    return () => {
      unsubscribe();
      store.setWorker(null);
    };
  }, [worker, store]);

  const value = useMemo<ContextValue>(() => ({ store, path: ["root"] }), [store]);

  return <Context value={value}>{ready && children}</Context>;
};

export interface UseLifecycleReturn<
  State extends z.ZodType<state.State>,
  Methods extends MethodsSchema = EmptyMethodsSchema,
> {
  path: readonly string[];
  setState: (state: RawSetArg<State>, transfer?: Transferable[]) => void;
  methods: CallersFromSchema<Methods>;
  subscribe: (listener: () => void) => () => void;
  getSnapshot: () => z.infer<State>;
}

type StateHandler<S = unknown> = (state: S) => void;

interface UseLifecycleProps<
  S extends z.ZodType,
  M extends MethodsSchema = EmptyMethodsSchema,
> {
  type: string;
  schema: S;
  aetherKey?: string;
  initialState: z.input<S>;
  initialTransfer?: Transferable[];
  onReceive?: StateHandler<z.infer<S>>;
  methods?: M;
}

export const useLifecycle = <
  State extends z.ZodType<state.State>,
  Methods extends MethodsSchema = EmptyMethodsSchema,
>({
  type,
  aetherKey,
  initialState,
  schema,
  initialTransfer = [],
  onReceive,
  methods: methodsSchema,
}: UseLifecycleProps<State, Methods>): UseLifecycleReturn<State, Methods> => {
  const key = useUniqueKey(aetherKey);
  const ctx = useContext();
  const path = useMemoPrimitiveArray([...ctx.path, key]);
  const onReceiveRef = useSyncedRef(onReceive);
  // Register synchronously on first render so parent components are created
  // before their children. Identity inputs (key, type, schema) are treated as
  // immutable per component lifetime; later renders reuse the same handle.
  const handleRef = useRef<Handle<State, Methods> | null>(null);
  handleRef.current ??= ctx.store.register({
    key,
    type,
    path,
    schema,
    initialState,
    initialTransfer,
    methodsSchema,
    onReceiveRef,
  });

  useLayoutEffect(
    () => () => {
      ctx.store.unregister(key);
      handleRef.current = null;
    },
    [ctx.store, key],
  );

  const setState = useCallback(
    (next: RawSetArg<State>, transfer: Transferable[] = []) =>
      handleRef.current?.setState(next, transfer),
    [],
  );

  const subscribe = useCallback(
    (listener: () => void) => ctx.store.subscribe(key, listener),
    [ctx.store, key],
  );

  const getSnapshot = useCallback(
    () => ctx.store.getSnapshot<z.infer<State>>(key),
    [ctx.store, key],
  );

  const methods = handleRef.current?.methods ?? ({} as CallersFromSchema<Methods>);

  return useMemo(
    () => ({ setState, path, methods, subscribe, getSnapshot }),
    [setState, path, methods, subscribe, getSnapshot],
  );
};

export interface ComponentProps {
  aetherKey?: string;
}

export interface UseProps<
  State extends z.ZodType,
  Methods extends MethodsSchema = EmptyMethodsSchema,
> extends Omit<UseLifecycleProps<State, Methods>, "onReceive"> {
  onAetherChange?: (state: z.infer<State>) => void;
}

interface ComponentContext {
  path: readonly string[];
}

/** Return tuple from {@link use}: context, current state, setter, methods. */
export type UseReturn<
  S extends z.ZodType<state.State>,
  M extends MethodsSchema = EmptyMethodsSchema,
> = [
  ComponentContext,
  z.infer<S>,
  (state: RawSetArg<S>, transfer?: Transferable[]) => void,
  CallersFromSchema<M>,
];

/** Props for {@link useUnidirectional}. */
export interface UseUnidirectionalProps<
  State extends z.ZodType,
  Methods extends MethodsSchema = EmptyMethodsSchema,
> extends Pick<UseLifecycleProps<State, Methods>, "schema" | "aetherKey" | "methods"> {
  type: string;
  state: z.input<State>;
}

export interface UseUnidirectionalReturn<
  Methods extends MethodsSchema = EmptyMethodsSchema,
> extends ComponentContext {
  methods: CallersFromSchema<Methods>;
}

export const useUnidirectional = <
  State extends z.ZodType<state.State>,
  Methods extends MethodsSchema = EmptyMethodsSchema,
>({
  state,
  ...rest
}: UseUnidirectionalProps<State, Methods>): UseUnidirectionalReturn<Methods> => {
  const { path, setState, methods } = useLifecycle<State, Methods>({
    ...rest,
    initialState: state,
  });
  const ref = useRef<z.input<State> | z.infer<State> | null>(null);
  if (!deep.equal(ref.current, state)) {
    ref.current = state;
    setState(state);
  }
  return { path, methods };
};

export const use = <
  State extends z.ZodType<state.State>,
  Methods extends MethodsSchema = EmptyMethodsSchema,
>(
  props: UseProps<State, Methods>,
): UseReturn<State, Methods> => {
  const { onAetherChange, ...rest } = props;
  const { path, setState, methods, subscribe, getSnapshot } = useLifecycle<
    State,
    Methods
  >({
    ...rest,
    onReceive: onAetherChange,
  });
  const state = useSyncExternalStore(subscribe, getSnapshot);
  return [{ path }, state, setState, methods];
};

/** Props for {@link Composite}: the path under which children are nested in
 * the Aether tree. */
export interface CompositeProps extends PropsWithChildren {
  path: readonly string[];
}

/**
 * Establishes children as nested components in the Aether tree. The provided
 * path should match the path returned by {@link use}. Composites can nest.
 */
export const Composite = memo(({ children, path }: CompositeProps): ReactElement => {
  const ctx = useContext();
  const value = useMemo<ContextValue>(() => ({ ...ctx, path }), [ctx, path]);
  return <Context value={value}>{children}</Context>;
});
Composite.displayName = "AetherComposite";
