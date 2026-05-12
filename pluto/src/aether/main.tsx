// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { deep, type destructor, type TimeSpan } from "@synnaxlabs/x";
import {
  memo,
  type PropsWithChildren,
  type ReactElement,
  useCallback,
  useLayoutEffect,
  useMemo,
  useRef,
  useSyncExternalStore,
} from "react";
import { type z } from "zod";

import {
  type CallersFromSchema,
  type EmptyMethodsSchema,
  type MethodsSchema,
} from "@/aether/aether/aether";
import {
  type AetherMessage,
  type MainMessage,
  type SenderHandler,
} from "@/aether/message";
import { type Handle, type RawSetArg, Store } from "@/aether/store";
import { context } from "@/context";
import { useSyncedRef } from "@/hooks";
import { useUniqueKey } from "@/hooks/useUniqueKey";
import { useMemoPrimitiveArray } from "@/memo";
import { type state } from "@/state";

export type { CallersFromSchema, EmptyMethodsSchema, MethodsSchema };
export { Store } from "@/aether/store";

/** Value provided by the Aether context to descendant components. */
export interface ContextValue {
  /** The current path in the Aether component tree. */
  path: readonly string[];
  /** The single store shared across this provider's tree. */
  store: Store;
}

const DEFAULT_STORE = new Store({ workerEnabled: false });

const [Context, useContext] = context.create<ContextValue>({
  defaultValue: { store: DEFAULT_STORE, path: ["root"] },
  displayName: "Aether.Context",
});

export interface ProviderProps extends PropsWithChildren {
  /** URL of the worker script to spawn. Ignored when {@link worker} is
   * provided. */
  workerURL?: string | URL;
  /** When false, the Aether tree mounts but no worker is spawned and all
   * worker-bound operations become no-ops. */
  workerEnabled?: boolean;
  /** Override the worker entirely. Primarily for tests that supply a mock
   * via {@link aether.createMockPair}. Bypasses workerURL/workerEnabled. */
  worker?: SenderHandler<MainMessage, AetherMessage>;
  /** Timeout for async invoke calls. Defaults to 5s. */
  invokeTimeout?: TimeSpan;
}

const ROOT_PATH = ["root"] as const;

export const Provider = ({ children, ...config }: ProviderProps): ReactElement => {
  const storeRef = useRef<Store | null>(null);
  storeRef.current ??= new Store(config);
  const store = storeRef.current;
  const subscribeError = useCallback(
    (listener: () => void) => {
      const unsubscribe = store.subscribeError(listener);
      return () => {
        unsubscribe();
        store.dispose();
      };
    },
    [store],
  );
  const getError = useCallback(() => store.getError(), [store]);
  const error = useSyncExternalStore(subscribeError, getError);
  if (error != null) throw error;

  const value = useMemo<ContextValue>(() => ({ store, path: ROOT_PATH }), [store]);

  return <Context value={value}>{children}</Context>;
};

export interface UseLifecycleReturn<
  State extends z.ZodType<state.State>,
  Methods extends MethodsSchema = EmptyMethodsSchema,
> {
  path: readonly string[];
  setState: (state: RawSetArg<State>, transfer?: Transferable[]) => void;
  methods: CallersFromSchema<Methods>;
  subscribe: (listener: () => void) => destructor.Destructor;
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
  onAetherChange?: StateHandler<z.infer<S>>;
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
  onAetherChange,
  methods: methodsSchema,
}: UseLifecycleProps<State, Methods>): UseLifecycleReturn<State, Methods> => {
  const key = useUniqueKey(aetherKey);
  const ctx = useContext();
  const path = useMemoPrimitiveArray([...ctx.path, key]);
  const onReceiveRef = useSyncedRef(onAetherChange);
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
  const { methods } = handleRef.current;

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
    () => ctx.store.getSnapshot<State>(key),
    [ctx.store, key],
  );

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

export type UseReturn<
  S extends z.ZodType<state.State>,
  M extends MethodsSchema = EmptyMethodsSchema,
> = [
  ComponentContext,
  z.infer<S>,
  (state: RawSetArg<S>, transfer?: Transferable[]) => void,
  CallersFromSchema<M>,
];

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
  const { path, setState, methods, subscribe, getSnapshot } = useLifecycle<
    State,
    Methods
  >(props);
  const state = useSyncExternalStore(subscribe, getSnapshot);
  return [{ path }, state, setState, methods];
};

export interface CompositeProps extends PropsWithChildren {
  path: readonly string[];
}

export const Composite = memo(({ children, path }: CompositeProps): ReactElement => {
  const ctx = useContext();
  const value = useMemo<ContextValue>(() => ({ ...ctx, path }), [ctx, path]);
  return <Context value={value}>{children}</Context>;
});
Composite.displayName = "Aether.Composite";
