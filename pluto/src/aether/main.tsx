// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type CrudeTimeSpan, deep, type destructor, type state } from "@synnaxlabs/x";
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
import { type MainComms } from "@/aether/aether/message";
import { type Handle, type RawSetArg, Store } from "@/aether/store";
import { context } from "@/context";
import { useSyncedRef } from "@/hooks";
import { useUniqueKey } from "@/hooks/useUniqueKey";

/** Value supplied by the Aether context to descendants of {@link Provider}. */
export interface ContextValue {
  /** Path of the nearest enclosing {@link Composite}, or `["root"]` at the top level.
   * Components append their key to derive their own path. */
  path: readonly string[];
  /** Store shared by every component below the {@link Provider}. */
  store: Store;
}

const DEFAULT_STORE = new Store({ workerEnabled: false });

const [Context, useContext] = context.create<ContextValue>({
  defaultValue: { store: DEFAULT_STORE, path: ["root"] },
  displayName: "Aether.Context",
});

export interface ProviderProps extends PropsWithChildren {
  /** URL of the worker script to spawn. Ignored when {@link worker} is set. */
  workerURL?: string | URL;
  /** Opt out of spawning a worker. Aether mounts, but all worker-bound operations
   * no-op. */
  workerEnabled?: boolean;
  /** Pre-built comms (e.g. from {@link aether.createMockPair} in tests). Takes
   * precedence over {@link workerURL} and {@link workerEnabled}. */
  worker?: MainComms;
  /** Default timeout for async method invocations. Defaults to 5s. */
  invokeTimeout?: CrudeTimeSpan;
}

const ROOT_PATH = ["root"] as const;

/** Roots an Aether tree: owns a {@link Store}, spawns or wraps the worker, and supplies
 * the context that descendant {@link use} / {@link useLifecycle} / {@link Composite}
 * calls read. Re-throws worker-reported errors so the nearest React error boundary can
 * catch them. */
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

  // Spawn the worker on commit rather than in the Store constructor, which runs during
  // render. A tree with no aether components still connects, so a worker that fails to
  // load reports it without waiting for a first component.
  useLayoutEffect(() => store.connect(), [store]);

  const value = useMemo<ContextValue>(() => ({ store, path: ROOT_PATH }), [store]);

  return <Context value={value}>{children}</Context>;
};

/** Output of {@link useLifecycle}: the component's path, a typed setState, the methods
 * registry, and the subscribe / getSnapshot pair consumed by `useSyncExternalStore`. */
export interface UseLifecycleReturn<
  StateSchema extends z.ZodType<state.State, state.State>,
  Methods extends MethodsSchema = EmptyMethodsSchema,
> {
  path: readonly string[];
  setState: (state: RawSetArg<StateSchema>, transfer?: Transferable[]) => void;
  methods: CallersFromSchema<Methods>;
  subscribe: (listener: () => void) => destructor.Destructor;
  getSnapshot: () => z.infer<StateSchema>;
}

type StateHandler<T = unknown> = (state: T) => void;

interface UseLifecycleProps<
  StateSchema extends z.ZodType,
  Methods extends MethodsSchema = EmptyMethodsSchema,
> {
  /** Component type, matched against the worker-side registry. */
  type: string;
  /** Zod schema validating both `initialState` and worker-pushed state. */
  schema: StateSchema;
  /** Key for the component, generated if omitted. Read on the mounting render only;
   * later changes are ignored. Remount under a React `key` to get a new identity. */
  aetherKey?: string;
  initialState: z.input<StateSchema>;
  /** Optional `Transferable`s included with the initial update message. */
  initialTransfer?: Transferable[];
  /** Fired on worker-pushed state changes only (not on local setState). */
  onAetherChange?: StateHandler<z.infer<StateSchema>>;
  methods?: Methods;
}

/** Registers a component with the enclosing {@link Provider}'s {@link Store} and
 * returns the operations needed to drive it. Lower-level than {@link use} — does not
 * subscribe React to state changes. Most callers want {@link use} or
 * {@link useUnidirectional} instead. */
export const useLifecycle = <
  StateSchema extends z.ZodType<state.State, state.State>,
  Methods extends MethodsSchema = EmptyMethodsSchema,
>({
  type,
  aetherKey,
  initialState,
  schema,
  initialTransfer = [],
  onAetherChange,
  methods: methodsSchema,
}: UseLifecycleProps<StateSchema, Methods>): UseLifecycleReturn<
  StateSchema,
  Methods
> => {
  const key = useUniqueKey(aetherKey);
  const ctx = useContext();
  const onReceiveRef = useSyncedRef(onAetherChange);
  const handleRef = useRef<Handle<StateSchema, Methods> | null>(null);
  // Staging is pure: the store keeps no reference and the worker hears nothing until
  // the attach below. A render React discards is reclaimed with this ref. `??=` short
  // circuits, so the path array is built on the mounting render alone.
  handleRef.current ??= ctx.store.stage({
    type,
    path: [...ctx.path, key],
    schema,
    initialState,
    initialTransfer,
    methodsSchema,
    onReceiveRef,
  });
  const handle = handleRef.current;

  // The handle outlives the effect so StrictMode's cleanup-then-setup remount, which
  // does not re-render, re-attaches this same component.
  useLayoutEffect(() => {
    handle.attach();
    return () => handle.detach();
  }, [handle]);

  // Every field is a closure the handle built once, so this runs on the mounting render
  // and never again. Wrapping them in `useCallback` would allocate an arrow and a
  // dependency array per render to reproduce identities that are already stable.
  return useMemo(
    () => ({
      path: handle.path,
      setState: handle.setState,
      methods: handle.methods,
      subscribe: handle.subscribe,
      getSnapshot: handle.getState,
    }),
    [handle],
  );
};

/** Mixin for React props of components that participate in the Aether tree. */
export interface ComponentProps {
  /** Optional override for the component's aether key. Stable for the component's
   * lifetime; usually omitted to auto-generate. */
  aetherKey?: string;
}

export interface UseProps<
  StateSchema extends z.ZodType,
  Methods extends MethodsSchema = EmptyMethodsSchema,
> extends Omit<UseLifecycleProps<StateSchema, Methods>, "onReceive"> {
  onAetherChange?: (state: z.infer<StateSchema>) => void;
}

interface ComponentContext {
  path: readonly string[];
}

/** Tuple returned by {@link use}: `[ctx, state, setState, methods]`. */
export type UseReturn<
  StateSchema extends z.ZodType<state.State, state.State>,
  Methods extends MethodsSchema = EmptyMethodsSchema,
> = [
  ComponentContext,
  z.infer<StateSchema>,
  (state: RawSetArg<StateSchema>, transfer?: Transferable[]) => void,
  CallersFromSchema<Methods>,
];

export interface UseUnidirectionalProps<
  StateSchema extends z.ZodType,
  Methods extends MethodsSchema = EmptyMethodsSchema,
> extends Pick<
  UseLifecycleProps<StateSchema, Methods>,
  "schema" | "aetherKey" | "methods"
> {
  type: string;
  /** Source-of-truth state owned by the caller. Push-only: changes here propagate to
   * the worker, but worker pushes do not flow back. */
  state: z.input<StateSchema>;
}

export interface UseUnidirectionalReturn<
  Methods extends MethodsSchema = EmptyMethodsSchema,
> extends ComponentContext {
  methods: CallersFromSchema<Methods>;
}

/** One-way binding for components whose state lives in React. Pushes `state` to the
 * worker on every change (compared with `deep.equal`); worker-side updates do not
 * propagate back. Use {@link use} for bidirectional state. */
export const useUnidirectional = <
  StateSchema extends z.ZodType<state.State, state.State>,
  Methods extends MethodsSchema = EmptyMethodsSchema,
>({
  state,
  ...rest
}: UseUnidirectionalProps<StateSchema, Methods>): UseUnidirectionalReturn<Methods> => {
  const { path, setState, methods } = useLifecycle<StateSchema, Methods>({
    ...rest,
    initialState: state,
  });
  // Seeded with the state the component staged with, so the mounting render does not
  // re-send what the create message already carries.
  const ref = useRef<z.input<StateSchema>>(state);
  // In an effect rather than in render: a render React discards must not push state to
  // the worker. The store buffers to a microtask either way, so nothing lands later.
  useLayoutEffect(() => {
    if (deep.equal(ref.current, state)) return;
    ref.current = state;
    setState(state);
  });
  return { path, methods };
};

/** Bidirectional binding: registers the component, subscribes React to its state via
 * `useSyncExternalStore`, and returns the current state, a setter, and the methods
 * registry. */
export const use = <
  StateSchema extends z.ZodType<state.State, state.State>,
  Methods extends MethodsSchema = EmptyMethodsSchema,
>(
  props: UseProps<StateSchema, Methods>,
): UseReturn<StateSchema, Methods> => {
  const { path, setState, methods, subscribe, getSnapshot } = useLifecycle<
    StateSchema,
    Methods
  >(props);
  const state = useSyncExternalStore(subscribe, getSnapshot);
  return [{ path }, state, setState, methods];
};

export interface CompositeProps extends PropsWithChildren {
  path: readonly string[];
}

/** Pushes a new aether path into the context so descendants register as children of the
 * composite component identified by `path`. */
export const Composite = memo(({ children, path }: CompositeProps): ReactElement => {
  const ctx = useContext();
  const value = useMemo<ContextValue>(() => ({ ...ctx, path }), [ctx, path]);
  return <Context value={value}>{children}</Context>;
});
Composite.displayName = "Aether.Composite";
