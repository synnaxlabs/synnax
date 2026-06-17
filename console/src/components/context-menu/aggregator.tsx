// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  createContext,
  type PropsWithChildren,
  type ReactElement,
  use,
  useCallback,
  useLayoutEffect,
  useRef,
  useSyncExternalStore,
} from "react";

/**
 * Handler is the behavior a registrar contributes for a verb. It runs against the
 * subset of the selection the registrar owns; the sink calls every registered handler
 * for a verb when the aggregated item is activated.
 */
export type Handler = () => void | Promise<void>;

type HandlerRef = { current: Handler };

const EMPTY: readonly HandlerRef[] = [];

class Store {
  private readonly refs = new Map<string, Set<HandlerRef>>();
  private readonly listeners = new Map<string, Set<() => void>>();
  private snapshots = new Map<string, readonly HandlerRef[]>();

  register(key: string, ref: HandlerRef): void {
    let set = this.refs.get(key);
    if (set == null) this.refs.set(key, (set = new Set()));
    set.add(ref);
    this.invalidate(key);
  }

  unregister(key: string, ref: HandlerRef): void {
    const set = this.refs.get(key);
    if (set == null) return;
    set.delete(ref);
    this.invalidate(key);
  }

  subscribe(key: string, listener: () => void): () => void {
    let set = this.listeners.get(key);
    if (set == null) this.listeners.set(key, (set = new Set()));
    set.add(listener);
    return () => {
      set.delete(listener);
    };
  }

  snapshot(key: string): readonly HandlerRef[] {
    return this.snapshots.get(key) ?? EMPTY;
  }

  private invalidate(key: string): void {
    const set = this.refs.get(key);
    this.snapshots.set(key, set == null || set.size === 0 ? EMPTY : [...set]);
    this.listeners.get(key)?.forEach((l) => l());
  }
}

const Context = createContext<Store | null>(null);
Context.displayName = "ContextMenu.Aggregator.Context";

const useStore = (caller: string): Store => {
  const store = use(Context);
  if (store == null)
    throw new Error(`${caller} must be used within ContextMenu.Aggregator.Provider`);
  return store;
};

/**
 * Provider scopes a verb aggregator to its subtree. Mount one per menu instance so the
 * registry is torn down when the menu closes. Registrars and sinks must be descendants.
 */
export const Provider = ({ children }: PropsWithChildren): ReactElement => {
  const store = useRef<Store>(null);
  store.current ??= new Store();
  return <Context value={store.current}>{children}</Context>;
};

export interface RegisterProps {
  /** When false, the handler is withheld from the verb and the sink does not count it. */
  enabled?: boolean;
  handler: Handler;
}

/**
 * useRegister contributes a handler to a verb for as long as the calling component is
 * mounted and enabled. The latest handler is always invoked, so callers may pass a fresh
 * closure each render without re-registering. Call this from a component that renders
 * nothing of its own; the aggregated item is drawn by the verb's sink.
 */
export const useRegister = (
  key: string,
  { enabled = true, handler }: RegisterProps,
): void => {
  const store = useStore("ContextMenu.Aggregator.useRegister");
  const ref = useRef<Handler>(handler);
  ref.current = handler;
  useLayoutEffect(() => {
    if (!enabled) return;
    store.register(key, ref);
    return () => store.unregister(key, ref);
  }, [store, key, enabled]);
};

export interface UseRunReturn {
  /** The number of currently registered, enabled handlers for the verb. */
  count: number;
  /** Invokes every registered handler in registration order, awaiting each. */
  run: Handler;
}

/**
 * useRun reads the live set of handlers registered for a verb. The sink renders the
 * aggregated item when count is non-zero and calls run when it is activated. count
 * reflects registrations made by descendants of the same Provider and updates before
 * paint, so the sink does not flicker on first open.
 */
export const useRun = (key: string): UseRunReturn => {
  const store = useStore("ContextMenu.Aggregator.useRun");
  const refs = useSyncExternalStore(
    useCallback((l) => store.subscribe(key, l), [store, key]),
    useCallback(() => store.snapshot(key), [store, key]),
  );
  const run = useCallback<Handler>(async () => {
    for (const ref of refs) await ref.current();
  }, [refs]);
  return { count: refs.length, run };
};
