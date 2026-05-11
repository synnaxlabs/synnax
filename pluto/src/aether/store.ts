// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { UnexpectedError, ValidationError } from "@synnaxlabs/client";
import { type errors, type SenderHandler, zod } from "@synnaxlabs/x";
import { type z } from "zod";

import {
  type CallersFromSchema,
  type EmptyMethodsSchema,
  isFireAndForget,
  type MethodsSchema,
} from "@/aether/aether/aether";
import { type AetherMessage, type MainMessage } from "@/aether/message";
import { type state } from "@/state";

const DEFAULT_INVOKE_TIMEOUT = 5000;

const reconstructError = (payload: errors.NativePayload): Error => {
  const err = new Error(payload.message);
  err.name = payload.name;
  err.stack = payload.stack;
  return err;
};

interface PendingRequest {
  resolve: (value: unknown) => void;
  reject: (error: Error) => void;
  controller: AbortController;
}

class InvokeTracker {
  private pending = new Map<string, PendingRequest>();
  private counters = new Map<string, number>();

  nextKey(componentKey: string): string {
    const counter = this.counters.get(componentKey) ?? 0;
    this.counters.set(componentKey, counter + 1);
    return `${componentKey}-${counter}`;
  }

  track(
    key: string,
    resolve: (value: unknown) => void,
    reject: (error: Error) => void,
    signal: AbortSignal,
  ): void {
    const controller = new AbortController();
    signal.addEventListener("abort", () => controller.abort(signal.reason), {
      signal: controller.signal,
    });
    controller.signal.addEventListener("abort", () => {
      this.pending.delete(key);
      reject(controller.signal.reason);
    });
    this.pending.set(key, { resolve, reject, controller });
  }

  resolve(key: string, result: unknown, error?: errors.NativePayload): boolean {
    const pending = this.pending.get(key);
    if (pending == null) return false;
    this.pending.delete(key);
    if (error != null) pending.reject(reconstructError(error));
    else pending.resolve(result);
    return true;
  }

  abort(reason: Error): void {
    this.pending.forEach(({ controller }) => controller.abort(reason));
    this.pending.clear();
  }

  clearCounter(key: string): void {
    this.counters.delete(key);
  }
}

/** Setter argument accepted by {@link Handle.setState}. */
export type RawSetArg<S extends z.ZodType<state.State>> =
  | (z.input<S> | z.infer<S>)
  | ((prev: z.infer<S>) => z.input<S> | z.infer<S>);

type Listener = () => void;
type WorkerPushListener = (state: unknown) => void;
type OnReceiveRef = { current: ((state: unknown) => void) | undefined };

/**
 * An Entry describes a live worker component. It is created on register and
 * destroyed on unregister. Listeners are stored separately so they survive
 * across the unregister-then-register cycle React StrictMode introduces.
 */
interface Entry {
  type: string;
  path: string[];
  schema: z.ZodType<state.State>;
  state: state.State;
  onReceiveRef: OnReceiveRef | null;
  controller: AbortController;
}

/** Params for registering a component with the store. */
export interface RegisterParams<
  S extends z.ZodType<state.State>,
  M extends MethodsSchema = EmptyMethodsSchema,
> {
  key: string;
  type: string;
  path: readonly string[];
  schema: S;
  initialState: z.input<S>;
  initialTransfer?: Transferable[];
  methodsSchema?: M;
  /** Ref to a callback fired only on worker-pushed state changes. Wired at
   * register time so synchronous worker pushes are not missed. */
  onReceiveRef?: { current: ((state: z.infer<S>) => void) | undefined };
}

/** Handle returned from {@link Store.register}. Bundles the per-component
 * operations: state mutation, deletion, and method callers. */
export interface Handle<
  S extends z.ZodType<state.State>,
  M extends MethodsSchema = EmptyMethodsSchema,
> {
  key: string;
  path: string[];
  methods: CallersFromSchema<M>;
  setState: (state: RawSetArg<S>, transfer?: Transferable[]) => void;
  delete: () => void;
}

/**
 * Store is the single source of truth for aether component state on the main
 * thread. One store is owned per Aether.Provider and shared across all
 * components in its tree. Components register with the store during render,
 * subscribe via useSyncExternalStore, and unregister on unmount. Worker pushes
 * land in the store and notify listeners outside of any React render.
 */
export class Store {
  private entries: Map<string, Entry> = new Map();
  // Listeners are keyed by string and persist independently of entry
  // lifetime. This is required by useSyncExternalStore semantics under
  // React StrictMode: the subscription is wired during a commit that may
  // straddle an unregister-then-register cycle.
  private listenersByKey: Map<string, Set<Listener>> = new Map();
  private workerListenersByKey: Map<string, Set<WorkerPushListener>> = new Map();
  private worker: SenderHandler<MainMessage, AetherMessage> | null = null;
  private invokeTracker = new InvokeTracker();
  private errorListeners: Set<(err: Error) => void> = new Set();

  /** Wires the store to a worker for message exchange. Passing null detaches. */
  setWorker(worker: SenderHandler<MainMessage, AetherMessage> | null): void {
    if (this.worker === worker) return;
    if (this.worker != null) this.invokeTracker.abort(new Error("aether worker reset"));
    this.worker = worker;
    if (worker == null) return;
    worker.handle((msg) => this.handleWorkerMessage(msg));
  }

  /** Subscribes to worker-side errors. Returns an unsubscribe function. */
  onError(listener: (err: Error) => void): () => void {
    this.errorListeners.add(listener);
    return () => {
      this.errorListeners.delete(listener);
    };
  }

  /** Subscribes to any change for the component identified by key. Fires on
   * both worker pushes and local setState calls. Suitable for
   * useSyncExternalStore. The subscription persists across register-unregister
   * cycles, so listeners attached during a StrictMode pseudo-remount remain
   * live once the real register happens. */
  subscribe(key: string, listener: Listener): () => void {
    let set = this.listenersByKey.get(key);
    if (set == null) {
      set = new Set();
      this.listenersByKey.set(key, set);
    }
    set.add(listener);
    return () => {
      const s = this.listenersByKey.get(key);
      if (s == null) return;
      s.delete(listener);
      if (s.size === 0) this.listenersByKey.delete(key);
    };
  }

  /** Subscribes to worker-pushed state changes only. Local setState calls do
   * not trigger this listener. Used to wire onAetherChange-style callbacks. */
  subscribeWorkerPush(key: string, listener: WorkerPushListener): () => void {
    let set = this.workerListenersByKey.get(key);
    if (set == null) {
      set = new Set();
      this.workerListenersByKey.set(key, set);
    }
    set.add(listener);
    return () => {
      const s = this.workerListenersByKey.get(key);
      if (s == null) return;
      s.delete(listener);
      if (s.size === 0) this.workerListenersByKey.delete(key);
    };
  }

  /** Reads the latest known state for the component identified by key. */
  getSnapshot<S>(key: string): S {
    const entry = this.entries.get(key);
    if (entry == null)
      throw new UnexpectedError(`[Aether.Store] - missing entry for key ${key}`);
    return entry.state as S;
  }

  /**
   * Registers a component with the store and synchronously sends the create +
   * initial state messages to the worker. If a component with the same key
   * already exists, the prior worker component is deleted and the entry is
   * updated in place. React listeners are preserved across re-registration
   * so subscribed consumers keep their subscription.
   */
  register<S extends z.ZodType<state.State>, M extends MethodsSchema>(
    params: RegisterParams<S, M>,
  ): Handle<S, M> {
    const {
      key,
      type,
      path,
      schema,
      initialState,
      initialTransfer = [],
      methodsSchema,
    } = params;
    if (key.length === 0)
      throw new ValidationError(
        `[Aether.Store] - received zero length key when registering component of type ${type}`,
      );
    if (type.length === 0)
      console.warn(
        `[Aether.Store] - received zero length type when registering component at ${path.join(".")}. This is probably a bad idea.`,
      );

    const parsed = zod.parse(schema, initialState, { label: type });
    const existing = this.entries.get(key);
    if (existing != null) {
      this.send({
        variant: "delete",
        path: existing.path,
        type: existing.type,
      });
      existing.controller.abort(new Error("Component re-registered"));
      existing.type = type;
      existing.path = path;
      existing.schema = schema;
      existing.state = parsed;
      existing.controller = new AbortController();
      if (params.onReceiveRef != null)
        existing.onReceiveRef = params.onReceiveRef as OnReceiveRef;
    } else
      this.entries.set(key, {
        type,
        path,
        schema,
        state: parsed,
        onReceiveRef: (params.onReceiveRef ?? null) as OnReceiveRef | null,
        controller: new AbortController(),
      });
    // Notify any listeners that subscribed before the entry was (re-)created
    // — under StrictMode, useSyncExternalStore can subscribe during a
    // pseudo-remount window where the entry has not yet been created.
    this.listenersByKey.get(key)?.forEach((l) => l());
    this.send({ variant: "update", path, state: parsed, type }, initialTransfer);

    return this.buildHandle(key, methodsSchema);
  }

  /** Removes a component from the store, sends a delete message to the worker,
   * and aborts any pending invokes scoped to this component. */
  unregister(key: string): void {
    const entry = this.entries.get(key);
    if (entry == null) return;
    this.send({ variant: "delete", path: entry.path, type: entry.type });
    entry.controller.abort(new Error("Component deleted"));
    this.invokeTracker.clearCounter(key);
    this.entries.delete(key);
  }

  private send(msg: MainMessage, transfer: Transferable[] = []): void {
    if (this.worker == null) {
      console.warn("aether - no worker");
      return;
    }
    this.worker.send(msg, transfer);
  }

  private handleWorkerMessage(msg: AetherMessage): void {
    const { variant } = msg;
    if (variant === "error") {
      const err = reconstructError(msg.error);
      this.errorListeners.forEach((l) => l(err));
      return;
    }
    if (variant === "invoke_response") {
      this.invokeTracker.resolve(msg.key, msg.result, msg.error);
      return;
    }
    const { key, state } = msg;
    const entry = this.entries.get(key);
    // A worker push can arrive for a key that has been unregistered on the
    // main side (e.g. delete and update messages crossing in flight, or
    // late-arriving echoes after StrictMode pseudo-unmount). Drop it.
    if (entry == null) return;
    const parsed = zod.parse(entry.schema, state, { label: entry.type });
    entry.state = parsed;
    this.listenersByKey.get(key)?.forEach((l) => l());
    this.workerListenersByKey.get(key)?.forEach((l) => l(parsed));
    entry.onReceiveRef?.current?.(parsed);
  }

  private buildHandle<S extends z.ZodType<state.State>, M extends MethodsSchema>(
    key: string,
    methodsSchema?: M,
  ): Handle<S, M> {
    const entry = this.entries.get(key);
    if (entry == null)
      throw new UnexpectedError(`[Aether.Store] - missing entry for key ${key}`);

    const setState = (next: RawSetArg<S>, transfer: Transferable[] = []): void => {
      const e = this.entries.get(key);
      if (e == null) return;
      const prev = e.state as z.infer<S>;
      const raw =
        typeof next === "function"
          ? (next as (p: z.infer<S>) => z.input<S> | z.infer<S>)(prev)
          : next;
      const parsed = zod.parse(e.schema, raw, { label: e.type });
      e.state = parsed;
      this.send(
        { variant: "update", path: e.path, state: parsed, type: e.type },
        transfer,
      );
      this.listenersByKey.get(key)?.forEach((l) => l());
    };

    const handleDelete = () => this.unregister(key);

    const invokeMethod = (method: string, args: unknown[]): void => {
      const e = this.entries.get(key);
      if (e == null) return;
      this.send({ variant: "invoke_request", path: e.path, method, args });
    };

    const invokeMethodAsync = <R>(
      method: string,
      args: unknown[],
      signal: AbortSignal = AbortSignal.timeout(DEFAULT_INVOKE_TIMEOUT),
    ): Promise<R> =>
      new Promise<R>((resolve, reject) => {
        const e = this.entries.get(key);
        if (e == null) return reject(new Error("Component deleted"));
        if (e.controller.signal.aborted) return reject(new Error("Component deleted"));
        if (this.worker == null) return reject(new Error("aether - no worker"));
        const invokeKey = this.invokeTracker.nextKey(key);
        this.invokeTracker.track(
          invokeKey,
          resolve as (v: unknown) => void,
          reject,
          AbortSignal.any([signal, e.controller.signal]),
        );
        this.worker.send({
          variant: "invoke_request",
          key: invokeKey,
          path: e.path,
          method,
          args,
        });
      });

    const methods = buildMethods<M>(invokeMethod, invokeMethodAsync, methodsSchema);

    return {
      key,
      path: entry.path,
      methods,
      setState,
      delete: handleDelete,
    };
  }
}

const buildMethods = <M extends MethodsSchema>(
  invokeMethod: (method: string, args: unknown[]) => void,
  invokeMethodAsync: (method: string, args: unknown[]) => Promise<unknown>,
  methodsSchema?: M,
): CallersFromSchema<M> => {
  if (methodsSchema == null) return {} as CallersFromSchema<M>;
  const callers: Record<string, (...args: unknown[]) => unknown> = {};
  for (const method of Object.keys(methodsSchema))
    callers[method] = isFireAndForget(methodsSchema[method])
      ? (...args: unknown[]) => invokeMethod(method, args)
      : (...args: unknown[]) => invokeMethodAsync(method, args);
  return callers as CallersFromSchema<M>;
};
