// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { UnexpectedError, ValidationError } from "@synnaxlabs/client";
import { type CrudeTimeSpan, errors, TimeSpan, zod } from "@synnaxlabs/x";
import { type z } from "zod";

import { aether } from "@/aether/aether";
import { state } from "@/state";

const DEFAULT_INVOKE_TIMEOUT = TimeSpan.seconds(5);

/** Path separator for store identities. Aether keys are dotless identifiers (nanoids,
 * UUIDs, numeric keys), so `.` joins unambiguously; `register` asserts keys never
 * contain it. */
const PATH_SEP = ".";

/** A component's store identity: its path flattened. Collides only on same-path
 * re-registration (e.g. a StrictMode remount), never across distinct components. */
const pathID = (path: readonly string[]): string => path.join(PATH_SEP);

interface PendingRequest {
  resolve: (value: unknown) => void;
  reject: (error: Error) => void;
  controller: AbortController;
}

class InvokeTracker {
  private pending = new Map<string, PendingRequest>();
  private counters = new Map<string, number>();

  nextKey(componentID: string): string {
    const counter = this.counters.get(componentID) ?? 0;
    this.counters.set(componentID, counter + 1);
    return `${componentID}-${counter}`;
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

  resolve(key: string, result: unknown, error?: errors.Payload): boolean {
    const pending = this.pending.get(key);
    if (pending == null) return false;
    this.pending.delete(key);
    if (error != null)
      pending.reject(
        errors.decode(error) ??
          new UnexpectedError(
            "[aether.store] worker reported an invoke error but the payload decoded to null",
          ),
      );
    else pending.resolve(result);
    return true;
  }

  abort(reason: Error): void {
    this.pending.forEach(({ controller }) => controller.abort(reason));
    this.pending.clear();
  }

  clearCounter(id: string): void {
    this.counters.delete(id);
  }
}

/** Setter argument accepted by {@link Handle.setState}: a new state value or a function
 * that derives one from the previous value. Always the schema's input type — the value
 * is parsed before being stored, so post-transform output types shouldn't be passed. */
export type RawSetArg<StateSchema extends z.ZodType<state.State, state.State>> =
  state.SetArg<z.input<StateSchema>, z.infer<StateSchema>>;

type Listener = () => void;

/** Live worker component tracked by the store. Generic over its schema so reads through
 * {@link Store.getEntry} recover the per-entry state and callback types without
 * re-asserting at every field access. */
interface Entry<
  StateSchema extends z.ZodType<state.State, state.State> = z.ZodType<
    state.State,
    state.State
  >,
> {
  type: string;
  path: readonly string[];
  schema: StateSchema;
  state: z.infer<StateSchema>;
  onReceiveRef: { current: ((state: z.infer<StateSchema>) => void) | undefined } | null;
  controller: AbortController;
}

/** Arguments accepted by {@link Store.register}. */
export interface RegisterParams<
  StateSchema extends z.ZodType<state.State, state.State>,
  Methods extends aether.MethodsSchema = aether.EmptyMethodsSchema,
> {
  /** Component type, matched against the worker-side registry. */
  type: string;
  /** Component path in the aether tree; its flattened form is the component's identity. */
  path: readonly string[];
  /** Zod schema validating both `initialState` and worker-pushed state. */
  schema: StateSchema;
  initialState: z.input<StateSchema>;
  /** Optional `Transferable`s included with the initial update message. */
  initialTransfer?: Transferable[];
  /** Optional method-call schema; powers `methods` on the returned handle. */
  methodsSchema?: Methods;
  /** Ref to a callback fired on worker-pushed state changes only. */
  onReceiveRef?: { current: ((state: z.infer<StateSchema>) => void) | undefined };
}

/** Per-component operations returned by {@link Store.register}: typed setState, delete,
 * and method callers. Scoped to the registration that produced it: once a same-path
 * re-registration displaces that entry, every operation no-ops (async invokes reject). */
export interface Handle<
  StateSchema extends z.ZodType<state.State, state.State>,
  Methods extends aether.MethodsSchema = aether.EmptyMethodsSchema,
> {
  path: readonly string[];
  methods: aether.CallersFromSchema<Methods>;
  setState: (state: RawSetArg<StateSchema>, transfer?: Transferable[]) => void;
  delete: () => void;
}

/** Configuration for a {@link Store}. Provide either a pre-built `worker` (e.g.
 * {@link createMockPair} in tests) or a `workerURL` for the store to spawn its own. Set
 * `workerEnabled: false` to explicitly opt out — any other missing-worker configuration
 * throws at construction. */
export interface StoreConfig {
  worker?: aether.MainComms;
  workerURL?: string | URL;
  workerEnabled?: boolean;
  /** Default timeout for async method invocations. Defaults to 5s. */
  invokeTimeout?: CrudeTimeSpan;
}

/**
 * Single source of truth for aether component state on the main thread. One store is
 * owned per {@link Aether.Provider} and shared across all components in its tree.
 * Components register on render, subscribe via `useSyncExternalStore`, and unregister
 * on unmount; worker pushes land in the store and notify listeners outside of any React
 * render.
 */
export class Store {
  /** Live entries keyed by component identity ({@link pathID}). */
  private entries: Map<string, Entry> = new Map();
  /** Subscribers keyed by component identity. Listeners persist across an entry's
   * register-unregister-register cycle so subscriptions wired during a StrictMode
   * pseudo-remount stay live once the real register fires. */
  private listeners: Map<string, Set<Listener>> = new Map();
  /** Last known state per component identity. Survives entry unregistration so that
   * `useSyncExternalStore`'s tearing-detection getSnapshot calls return a stable value
   * across the unregister-then-register window. Cleaned up when both the entry and all
   * subscribers for the identity are gone. */
  private snapshots: Map<string, state.State> = new Map();
  /** Active worker comms. {@link NOOP_WORKER} when `workerEnabled: false` or between
   * {@link dispose} and the next send (lazy re-attach). */
  private worker: aether.MainComms = aether.NOOP_MAIN_COMMS;
  private invokeTracker = new InvokeTracker();
  /** Most recent worker-reported error. Buffered so it survives the gap between a
   * synchronous worker push and the Provider's subscription. */
  private currentError: Error | null = null;
  private errorListeners: Set<Listener> = new Set();
  /** Raw {@link Worker} this store spawned; `null` for externally-injected comms
   * (tests) which are owned by the caller. */
  private ownedWorker: Worker | null = null;
  /** Config retained so {@link ensureAttached} can lazily rebuild the worker after
   * {@link dispose} — required for the StrictMode reused-fiber cycle. */
  private config: StoreConfig;

  /** Throws {@link ValidationError} if `workerEnabled` is true (the default) and
   * neither `worker` nor `workerURL` is provided. */
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
    if (this.worker !== aether.NOOP_MAIN_COMMS) return;
    const { worker, workerURL, workerEnabled = true } = this.config;
    if (!workerEnabled) return;
    if (worker != null) this.worker = worker;
    else if (workerURL != null) {
      this.ownedWorker = new Worker(workerURL, { type: "module" });
      // Transport-level failures (script load, syntax error, CSP rejection,
      // structured-clone deserialization) don't reach `onmessage`. Route them into the
      // same error pipeline as worker-reported errors so the Provider can rethrow them
      // into an error boundary.
      this.ownedWorker.onerror = (e) => {
        const location =
          e.filename != null && e.filename.length > 0
            ? ` (${e.filename}:${e.lineno}:${e.colno})`
            : "";
        const message =
          e.message != null && e.message.length > 0 ? e.message : "unknown";
        this.setError(new Error(`[aether] worker error: ${message}${location}`));
      };
      this.ownedWorker.onmessageerror = () =>
        this.setError(new Error("[aether] failed to deserialize message from worker"));
      this.worker = aether.wrapWorker(this.ownedWorker);
    }
    this.worker.handle((msg) => this.handleWorkerMessage(msg));
  }

  private setError(err: Error): void {
    if (this.currentError != null)
      console.error(
        "[aether] received new error after error was already set, but before previous error was thrown.",
        err,
      );
    this.currentError = err;
    this.errorListeners.forEach((l) => l());
  }

  /** Detaches the worker handler, terminates any owned `Worker`, and aborts in-flight
   * invokes. The store remains usable: a subsequent send lazily re-attaches via a fresh
   * `Worker`. Idempotent. */
  dispose(): void {
    if (this.worker === aether.NOOP_MAIN_COMMS) return;
    this.invokeTracker.abort(new Error("aether store disposed"));
    this.worker.handle(() => {});
    this.worker = aether.NOOP_MAIN_COMMS;
    this.ownedWorker?.terminate();
    this.ownedWorker = null;
  }

  /** Returns the most recent worker-reported error, or `null` if none. Suitable for
   * `useSyncExternalStore`. */
  getError(): Error | null {
    return this.currentError;
  }

  /** Subscribes to worker-error notifications. The listener fires whenever a new error
   * is buffered; re-read via {@link getError}. Returns an unsubscribe function. */
  subscribeError(listener: Listener): () => void {
    this.errorListeners.add(listener);
    return () => {
      this.errorListeners.delete(listener);
    };
  }

  /** Subscribes to state changes for the component at `path`. Fires on both worker
   * pushes and local {@link Handle.setState} calls. Subscriptions persist across
   * register-unregister cycles. Returns an unsubscribe function. */
  subscribe(path: readonly string[], listener: Listener): () => void {
    const id = pathID(path);
    let set = this.listeners.get(id);
    if (set == null) {
      set = new Set();
      this.listeners.set(id, set);
    }
    set.add(listener);
    return () => {
      const s = this.listeners.get(id);
      if (s == null) return;
      s.delete(listener);
      if (s.size > 0) return;
      this.listeners.delete(id);
      if (!this.entries.has(id)) this.snapshots.delete(id);
    };
  }

  /** Returns the latest known state for the component at `path`. Falls back to the
   * cached last-known snapshot when the entry has been unregistered but subscribers
   * remain (e.g. StrictMode's unregister-then-register window). Throws
   * {@link UnexpectedError} only when neither is available. */
  getSnapshot<StateSchema extends z.ZodType<state.State, state.State>>(
    path: readonly string[],
  ): z.infer<StateSchema> {
    const id = pathID(path);
    const entry = this.getEntry<StateSchema>(id);
    if (entry != null) return entry.state;
    const cached = this.snapshots.get(id);
    if (cached != null) return cached as z.infer<StateSchema>;
    throw new UnexpectedError(
      `[aether.store] missing entry for path ${path.join(".")}`,
    );
  }

  /** Single seam where the schema-specific types of `state` and `onReceiveRef` are
   * recovered from the erased storage map. */
  private getEntry<StateSchema extends z.ZodType<state.State, state.State>>(
    id: string,
  ): Entry<StateSchema> | undefined {
    return this.entries.get(id) as Entry<StateSchema> | undefined;
  }

  /** Single seam where an Entry's schema-specific types are erased on insert into the
   * uniform storage map. */
  private setEntry<StateSchema extends z.ZodType<state.State, state.State>>(
    id: string,
    entry: Entry<StateSchema>,
  ): void {
    this.entries.set(id, entry);
  }

  /** Registers the component described by `params` and sends its initial state to the
   * worker. If a component is already registered at the same path (e.g. a StrictMode
   * remount), the prior worker component is deleted and the entry is replaced;
   * subscribers keep their subscriptions. */
  register<
    StateSchema extends z.ZodType<state.State, state.State>,
    Methods extends aether.MethodsSchema,
  >(params: RegisterParams<StateSchema, Methods>): Handle<StateSchema, Methods> {
    const {
      type,
      path,
      schema,
      initialState,
      initialTransfer = [],
      methodsSchema,
    } = params;
    if (path.length === 0 || path[path.length - 1].length === 0)
      throw new ValidationError(
        `[aether.store] received empty path or leaf key when registering component of type ${type}`,
      );
    if (path.some((segment) => segment.includes(PATH_SEP)))
      throw new ValidationError(
        `[aether.store] aether key may not contain the reserved path separator "${PATH_SEP}" when registering component of type ${type} at ${path.join(" / ")}`,
      );
    if (type.length === 0)
      console.warn(
        `[aether.store] received zero length type when registering component at ${path.join(".")}. This is probably a bad idea.`,
      );

    const id = pathID(path);
    const parsed = zod.parse(schema, initialState, { label: type });
    const existing = this.entries.get(id);
    if (existing != null) {
      this.send({ variant: "delete", path: existing.path });
      existing.controller.abort(new Error("Component re-registered"));
    }
    this.setEntry<StateSchema>(id, {
      type,
      path,
      schema,
      state: parsed,
      onReceiveRef: params.onReceiveRef ?? null,
      controller: new AbortController(),
    });
    this.snapshots.set(id, parsed);
    // Deferred: register runs during a React render and uses listeners are setStates
    // React refuses to accept mid-render. StrictMode's pseudo- remount path can hit
    // this with a persisted listener still attached.
    const listeners = this.listeners.get(id);
    if (listeners != null && listeners.size > 0)
      queueMicrotask(() => listeners.forEach((l) => l()));
    this.send({ variant: "update", path, state: parsed, type }, initialTransfer);

    return this.buildHandle(id, methodsSchema);
  }

  /** Deletes the component at `path`: sends a delete message to the worker and aborts
   * any pending invokes scoped to the component. No-op if nothing is registered at
   * `path`. */
  unregister(path: readonly string[]): void {
    const id = pathID(path);
    const entry = this.entries.get(id);
    if (entry == null) return;
    this.send({ variant: "delete", path: entry.path });
    entry.controller.abort(new Error("Component deleted"));
    this.invokeTracker.clearCounter(id);
    this.entries.delete(id);
    if (!this.listeners.has(id)) this.snapshots.delete(id);
  }

  private send(msg: aether.MainMessage, transfer: Transferable[] = []): void {
    this.ensureAttached();
    this.worker.send(msg, transfer);
  }

  private handleWorkerMessage(msg: aether.WorkerMessage): void {
    const { variant } = msg;
    if (variant === "error") {
      this.setError(
        errors.decode(msg.error) ??
          new UnexpectedError(
            "[aether.store] worker error message decoded to null; the error payload contract requires a non-nil payload",
          ),
      );
      return;
    }
    if (variant === "invoke_response") {
      this.invokeTracker.resolve(msg.key, msg.result, msg.error);
      return;
    }
    const { path, state } = msg;
    const id = pathID(path);
    const entry = this.entries.get(id);
    // Drop pushes for an unregistered path — possible when delete/update messages cross
    // in flight, or after a StrictMode pseudo-unmount.
    if (entry == null) return;
    const parsed = zod.parse(entry.schema, state, { label: entry.type });
    entry.state = parsed;
    this.snapshots.set(id, parsed);
    this.listeners.get(id)?.forEach((l) => l());
    entry.onReceiveRef?.current?.(parsed);
  }

  private buildHandle<
    StateSchema extends z.ZodType<state.State, state.State>,
    Methods extends aether.MethodsSchema,
  >(id: string, methodsSchema?: Methods): Handle<StateSchema, Methods> {
    const entry = this.getEntry<StateSchema>(id);
    if (entry == null)
      throw new UnexpectedError(`[aether.store] missing entry for id ${id}`);

    // Guard every operation on entry identity: a same-path re-registration
    // (StrictMode remount, single-commit re-parent) replaces the entry, and the
    // displaced instance's stale delete must not tear down its successor.
    const setState = (
      next: RawSetArg<StateSchema>,
      transfer: Transferable[] = [],
    ): void => {
      if (this.entries.get(id) !== entry) return;
      const raw = state.executeSetter<z.input<StateSchema>, z.infer<StateSchema>>(
        next,
        entry.state,
      );
      const parsed = zod.parse(entry.schema, raw, { label: entry.type });
      entry.state = parsed;
      this.snapshots.set(id, parsed);
      this.send(
        { variant: "update", path: entry.path, state: parsed, type: entry.type },
        transfer,
      );
      this.listeners.get(id)?.forEach((l) => l());
    };

    const handleDelete = () => {
      if (this.entries.get(id) !== entry) return;
      this.unregister(entry.path);
    };

    const invokeMethod = (method: string, args: unknown[]): void => {
      if (this.entries.get(id) !== entry) return;
      this.send({ variant: "invoke_request", path: entry.path, method, args });
    };

    const invokeMethodAsync = (
      method: string,
      args: unknown[],
      signal: AbortSignal = AbortSignal.timeout(
        new TimeSpan(this.config.invokeTimeout ?? DEFAULT_INVOKE_TIMEOUT).milliseconds,
      ),
    ): Promise<unknown> =>
      new Promise((resolve, reject) => {
        if (this.entries.get(id) !== entry || entry.controller.signal.aborted)
          return reject(new Error("Component deleted"));
        const invokeKey = this.invokeTracker.nextKey(id);
        this.invokeTracker.track(
          invokeKey,
          resolve,
          reject,
          AbortSignal.any([signal, entry.controller.signal]),
        );
        this.send({
          variant: "invoke_request",
          key: invokeKey,
          path: entry.path,
          method,
          args,
        });
      });

    const methods = buildMethods<Methods>(
      invokeMethod,
      invokeMethodAsync,
      methodsSchema,
    );

    return { path: entry.path, methods, setState, delete: handleDelete };
  }
}

const buildMethods = <Methods extends aether.MethodsSchema>(
  invokeMethod: (method: string, args: unknown[]) => void,
  invokeMethodAsync: (method: string, args: unknown[]) => Promise<unknown>,
  methodsSchema?: Methods,
): aether.CallersFromSchema<Methods> => {
  const callers: Record<string, (...args: unknown[]) => unknown> = {};
  if (methodsSchema != null)
    for (const [method, schema] of Object.entries(methodsSchema)) {
      const base = aether.isFireAndForget(schema) ? invokeMethod : invokeMethodAsync;
      callers[method] = (...args: unknown[]) => base(method, args);
    }
  return callers as aether.CallersFromSchema<Methods>;
};
