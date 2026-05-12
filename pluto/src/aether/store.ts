// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { UnexpectedError, ValidationError } from "@synnaxlabs/client";
import { type errors, TimeSpan, zod } from "@synnaxlabs/x";
import { type z } from "zod";

import {
  type CallersFromSchema,
  type EmptyMethodsSchema,
  isFireAndForget,
  type MethodsSchema,
} from "@/aether/aether/aether";
import {
  type AetherMessage,
  type MainMessage,
  type SenderHandler,
  wrapWorker,
} from "@/aether/message";
import { type state } from "@/state";

const DEFAULT_INVOKE_TIMEOUT = TimeSpan.seconds(5);

/** Sentinel used when `workerEnabled: false` — the explicit "no worker" mode.
 * All sends and handlers are silently dropped, so the {@link Store} stays
 * usable without a runtime null check on every method. Any other missing-
 * worker configuration is a constructor-time error, not a runtime fallback. */
const NOOP_WORKER: SenderHandler<MainMessage, AetherMessage> = {
  send: () => {},
  handle: () => {},
};

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

/**
 * An Entry describes a live worker component. It is created on register and
 * destroyed on unregister. Listeners are stored separately so they survive
 * across the unregister-then-register cycle React StrictMode introduces.
 *
 * Entry is generic over its schema so reads through {@link Store.getEntry}
 * recover the entry's specific state and callback types in one cast,
 * rather than re-asserting at every field access.
 */
interface Entry<S extends z.ZodType<state.State> = z.ZodType<state.State>> {
  type: string;
  path: readonly string[];
  schema: S;
  state: z.infer<S>;
  onReceiveRef: { current: ((state: z.infer<S>) => void) | undefined } | null;
  controller: AbortController;
}

/** Params for registering a component with the store. */
export interface RegisterParams<
  State extends z.ZodType<state.State>,
  Methods extends MethodsSchema = EmptyMethodsSchema,
> {
  key: string;
  type: string;
  path: readonly string[];
  schema: State;
  initialState: z.input<State>;
  initialTransfer?: Transferable[];
  methodsSchema?: Methods;
  /** Ref to a callback fired only on worker-pushed state changes. Wired at
   * register time so synchronous worker pushes are not missed. */
  onReceiveRef?: { current: ((state: z.infer<State>) => void) | undefined };
}

/** Handle returned from {@link Store.register}. Bundles the per-component
 * operations: state mutation, deletion, and method callers. */
export interface Handle<
  State extends z.ZodType<state.State>,
  Methods extends MethodsSchema = EmptyMethodsSchema,
> {
  key: string;
  path: readonly string[];
  methods: CallersFromSchema<Methods>;
  setState: (state: RawSetArg<State>, transfer?: Transferable[]) => void;
  delete: () => void;
}

/** Config for {@link Store}: pass either a ready-made `worker` (tests), or a
 * `workerURL` for the store to spawn its own worker. With neither, the store
 * has no worker and all sends are no-ops. */
export interface StoreConfig {
  worker?: SenderHandler<MainMessage, AetherMessage>;
  workerURL?: string | URL;
  workerEnabled?: boolean;
  invokeTimeout?: TimeSpan;
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
  private worker: SenderHandler<MainMessage, AetherMessage> = NOOP_WORKER;
  private invokeTracker = new InvokeTracker();
  // Errors are buffered on the store so they survive the gap between a
  // synchronous worker push and the Provider's subscription. The Provider
  // reads via getError() in render and re-renders via subscribeError().
  private currentError: Error | null = null;
  private errorListeners: Set<Listener> = new Set();
  // The raw Worker instance the store spawned, if any. Stored so dispose()
  // can terminate it. Externally-injected senders (tests) are owned by the
  // caller and not terminated here.
  private ownedWorker: Worker | null = null;
  // Config retained so the store can lazily re-attach after dispose. This
  // makes the store reusable across React StrictMode's pseudo-unmount /
  // pseudo-remount cycle, where the same fiber is reused and useRef
  // persists across the cycle.
  private config: StoreConfig;

  constructor(config: StoreConfig = {}) {
    const { worker, workerURL, workerEnabled = true } = config;
    if (workerEnabled && worker == null && workerURL == null)
      throw new ValidationError(
        "[aether.store] worker is enabled but neither `worker` nor `workerURL` was provided. Pass `workerEnabled: false` to opt out explicitly.",
      );
    this.config = config;
    this.ensureAttached();
  }

  private ensureAttached(): void {
    if (this.worker !== NOOP_WORKER) return;
    const { worker, workerURL, workerEnabled = true } = this.config;
    if (!workerEnabled) return;
    if (worker != null) this.worker = worker;
    else if (workerURL != null) {
      this.ownedWorker = new Worker(workerURL, { type: "module" });
      this.worker = wrapWorker<MainMessage, AetherMessage>(this.ownedWorker);
    }
    this.worker.handle((msg) => this.handleWorkerMessage(msg));
  }

  /** Tears down the current worker connection: detaches the handler,
   * terminates any owned Worker, and aborts in-flight invokes. The store
   * itself remains usable — a subsequent send will lazily re-attach via a
   * fresh Worker. This is what makes the store survive React StrictMode's
   * pseudo-unmount/remount cycle, where the same fiber is reused. */
  dispose(): void {
    if (this.worker === NOOP_WORKER) return;
    this.invokeTracker.abort(new Error("aether store disposed"));
    this.worker.handle(() => {});
    this.worker = NOOP_WORKER;
    this.ownedWorker?.terminate();
    this.ownedWorker = null;
  }

  /** Reads the latest buffered worker error. Returns null if none has been
   * reported since the store was created. Suitable for useSyncExternalStore. */
  getError(): Error | null {
    return this.currentError;
  }

  /** Subscribes to worker error notifications. Listeners are invoked whenever
   * a new error is buffered; consumers re-read via getError(). */
  subscribeError(listener: Listener): () => void {
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

  /** Reads the latest known state for the component identified by key. */
  getSnapshot<S extends z.ZodType<state.State>>(key: string): z.infer<S> {
    const entry = this.getEntry<S>(key);
    if (entry == null)
      throw new UnexpectedError(`[aether.store] missing entry for key ${key}`);
    return entry.state;
  }

  /** Centralized typed read of an entry. The single cast here recovers the
   * schema-specific types of `state` and `onReceiveRef` from the erased
   * storage map, so call sites can read those fields without further casts. */
  private getEntry<S extends z.ZodType<state.State>>(
    key: string,
  ): Entry<S> | undefined {
    return this.entries.get(key) as Entry<S> | undefined;
  }

  /** Centralized typed write. The single cast here erases the schema-specific
   * types of the Entry's fields so it can live in the uniform storage map. */
  private setEntry<S extends z.ZodType<state.State>>(
    key: string,
    entry: Entry<S>,
  ): void {
    this.entries.set(key, entry as Entry);
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
        `[aether.store] received zero length key when registering component of type ${type}`,
      );
    if (type.length === 0)
      console.warn(
        `[aether.store] received zero length type when registering component at ${path.join(".")}. This is probably a bad idea.`,
      );

    const parsed = zod.parse(schema, initialState, { label: type });
    const existing = this.entries.get(key);
    if (existing != null) {
      this.send({ variant: "delete", path: existing.path, type: existing.type });
      existing.controller.abort(new Error("Component re-registered"));
    }
    this.setEntry<S>(key, {
      type,
      path,
      schema,
      state: parsed,
      onReceiveRef: params.onReceiveRef ?? null,
      controller: new AbortController(),
    });
    // Notify any listeners that subscribed before the entry was (re-)created.
    // Deferred to a microtask so the notification cannot fire inside the
    // current React render — register is called during the render phase of
    // useLifecycle, and uSES listeners are setStates that React refuses to
    // accept mid-render. Under StrictMode the pseudo-remount triggers this
    // path with a persisted listener still attached.
    const listeners = this.listenersByKey.get(key);
    if (listeners != null && listeners.size > 0)
      queueMicrotask(() => listeners.forEach((l) => l()));
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
    this.ensureAttached();
    this.worker.send(msg, transfer);
  }

  private handleWorkerMessage(msg: AetherMessage): void {
    const { variant } = msg;
    if (variant === "error") {
      const err = reconstructError(msg.error);
      if (this.currentError != null) {
        console.error(
          "[aether] received new error after error was already set, but before previous error was thrown.",
        );
        console.error(err);
      }
      this.currentError = err;
      this.errorListeners.forEach((l) => l());
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
    entry.onReceiveRef?.current?.(parsed);
  }

  private buildHandle<S extends z.ZodType<state.State>, M extends MethodsSchema>(
    key: string,
    methodsSchema?: M,
  ): Handle<S, M> {
    const entry = this.getEntry<S>(key);
    if (entry == null)
      throw new UnexpectedError(`[aether.store] missing entry for key ${key}`);

    const setState = (next: RawSetArg<S>, transfer: Transferable[] = []): void => {
      const e = this.getEntry<S>(key);
      if (e == null) return;
      const raw = typeof next === "function" ? next(e.state) : next;
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

    const invokeMethodAsync = (
      method: string,
      args: unknown[],
      signal: AbortSignal = AbortSignal.timeout(
        (this.config.invokeTimeout ?? DEFAULT_INVOKE_TIMEOUT).milliseconds,
      ),
    ): Promise<unknown> =>
      new Promise((resolve, reject) => {
        const e = this.entries.get(key);
        if (e == null || e.controller.signal.aborted)
          return reject(new Error("Component deleted"));
        const invokeKey = this.invokeTracker.nextKey(key);
        this.invokeTracker.track(
          invokeKey,
          resolve,
          reject,
          AbortSignal.any([signal, e.controller.signal]),
        );
        this.send({
          variant: "invoke_request",
          key: invokeKey,
          path: e.path,
          method,
          args,
        });
      });

    const methods = buildMethods<M>(invokeMethod, invokeMethodAsync, methodsSchema);

    return { key, path: entry.path, methods, setState, delete: handleDelete };
  }
}

const buildMethods = <Methods extends MethodsSchema>(
  invokeMethod: (method: string, args: unknown[]) => void,
  invokeMethodAsync: (method: string, args: unknown[]) => Promise<unknown>,
  methodsSchema?: Methods,
): CallersFromSchema<Methods> => {
  const callers: Record<string, (...args: unknown[]) => unknown> = {};
  if (methodsSchema != null)
    for (const [method, schema] of Object.entries(methodsSchema)) {
      const base = isFireAndForget(schema) ? invokeMethod : invokeMethodAsync;
      callers[method] = (...args: unknown[]) => base(method, args);
    }
  return callers as CallersFromSchema<Methods>;
};
