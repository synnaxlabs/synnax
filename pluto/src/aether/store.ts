// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { UnexpectedError, ValidationError } from "@synnaxlabs/client";
import {
  type CrudeTimeSpan,
  type destructor,
  errors,
  state,
  TimeSpan,
  zod,
} from "@synnaxlabs/x";
import { type z } from "zod";

import { aether } from "@/aether/aether";

const DEFAULT_INVOKE_TIMEOUT = TimeSpan.seconds(5);

/** Path separator for store identities. Aether keys are dotless identifiers (nanoids,
 * UUIDs, numeric keys), so `.` joins unambiguously; `stage` asserts keys never
 * contain it. */
const PATH_SEP = ".";

/** A component's store identity: its path flattened. Collides only on a same-path
 * re-attach (e.g. a StrictMode remount), never across distinct components. */
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

/** How far a component has progressed toward existing on the worker. `staged` means the
 * caller holds it but the worker has never heard of it; `queued` means it is attached
 * and waiting for the next flush; `live` means its create message has been sent. */
type Phase = "staged" | "queued" | "live";

/** Worker component tracked by the store. Generic over its schema so {@link Store.stage}
 * and the handle it returns keep the per-entry state and callback types without
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
  phase: Phase;
  /** Set once another entry takes this path. Permanent: the handle is dead and every
   * operation on it must no-op so its stale writes never reach the successor. */
  displaced: boolean;
  /** Transferables from setStates that ran before the create message flushed. Handed to
   * that message so ownership transfers exactly once. */
  transfer: Transferable[];
  /** Invokes issued before the create message flushed. The worker cannot resolve a path
   * it has not seen, so they ride out immediately after it. */
  pendingInvokes: aether.MainInvokeRequest[];
}

/** Arguments accepted by {@link Store.stage}. */
export interface StageParams<
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

/** Per-component operations returned by {@link Store.stage}: typed setState, the
 * attach/detach pair, state reads, subscriptions, and method callers. Scoped to the
 * staging that produced it: once a same-path attach displaces that entry, every
 * operation no-ops (async invokes reject).
 *
 * Every field is built once with the handle and never replaced, so a React caller can
 * hand them straight to hooks without wrapping them in `useCallback`. */
export interface Handle<
  StateSchema extends z.ZodType<state.State, state.State>,
  Methods extends aether.MethodsSchema = aether.EmptyMethodsSchema,
> {
  path: readonly string[];
  methods: aether.CallersFromSchema<Methods>;
  setState: (state: RawSetArg<StateSchema>, transfer?: Transferable[]) => void;
  /** Latest state, readable in every phase. Owned by the handle rather than the store
   * so it stays stable across a detach/attach cycle. */
  getState: () => z.infer<StateSchema>;
  /** Subscribes to this component's state changes. Fires on both worker pushes and local
   * {@link Handle.setState}, and persists across detach-attach cycles. Returns an
   * unsubscribe function. */
  subscribe: (listener: Listener) => destructor.Destructor;
  /** Publishes the component to the store and queues its create message. Call from a
   * layout effect: a component staged by a render React discards must never reach the
   * worker. Idempotent. */
  attach: () => void;
  /** Withdraws the component, sending a delete only if its create message already went
   * out. The handle stays reusable, so a StrictMode remount re-attaches it. */
  detach: () => void;
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
 * Components stage on render, attach on commit, subscribe via `useSyncExternalStore`,
 * and detach on unmount; worker pushes land in the store and notify listeners outside
 * of any React render.
 */
export class Store {
  /** Attached entries keyed by component identity ({@link pathID}). */
  private entries: Map<string, Entry> = new Map();
  /** Subscribers keyed by component identity. Listeners persist across an entry's
   * detach-attach cycle so subscriptions wired during a StrictMode pseudo-remount stay
   * live once the real attach fires. */
  private listeners: Map<string, Set<Listener>> = new Map();
  /** Entries attached since the last flush, in attach order. */
  private queued: Entry[] = [];
  private readonly outbound: aether.Batcher<aether.MainMessage>;
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
  /** Config retained so {@link connect} can lazily rebuild the worker after
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
    this.outbound = new aether.Batcher<aether.MainMessage>(
      (messages, transfer) => {
        this.connect();
        this.worker.send(messages, transfer);
        // A listener notified during the drain may have attached another component.
        if (this.queued.length > 0) this.outbound.schedule();
      },
      () => this.drainCreates(),
    );
  }

  /** Spawns the worker (when configured with a `workerURL`) and starts handling its
   * messages. Called on the {@link Aether.Provider}'s first commit and again by any
   * send after {@link dispose}. Deferred out of the constructor so a Provider render
   * React discards never spawns a worker. Idempotent. */
  connect(): void {
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
    // Each message is caught on its own so one bad push reports and the rest of the
    // batch still applies.
    this.worker.handle((messages) => {
      for (const msg of messages)
        try {
          this.handleWorkerMessage(msg);
        } catch (e) {
          this.setError(errors.fromUnknown(e));
        }
    });
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

  /** Clears the worker-side tree, detaches the worker handler, terminates any owned
   * `Worker`, and aborts in-flight invokes. The store remains usable: a subsequent
   * send lazily re-attaches via a fresh `Worker`. Idempotent. */
  dispose(): void {
    this.queued = [];
    this.outbound.clear();
    if (this.worker === aether.NOOP_MAIN_COMMS) return;
    this.invokeTracker.abort(new Error("aether store disposed"));
    // In-process comms have no thread to die with, so tear the tree down explicitly
    // rather than relying on every component to have detached first.
    this.worker.send([{ variant: "clear" }]);
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
   * detach-attach cycles. Returns an unsubscribe function. */
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
    };
  }

  /** Validates and parses `params` into a handle owned by the caller. The store keeps
   * no reference and the worker is not told anything until {@link Handle.attach} runs,
   * so a component staged by a render React discards is reclaimed with the caller's
   * own reference. */
  stage<
    StateSchema extends z.ZodType<state.State, state.State>,
    Methods extends aether.MethodsSchema,
  >(params: StageParams<StateSchema, Methods>): Handle<StateSchema, Methods> {
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

    const entry: Entry<StateSchema> = {
      type,
      path,
      schema,
      state: zod.parse(schema, initialState, { label: type }),
      onReceiveRef: params.onReceiveRef ?? null,
      controller: new AbortController(),
      phase: "staged",
      displaced: false,
      transfer: [...initialTransfer],
      pendingInvokes: [],
    };
    return this.buildHandle(entry, methodsSchema);
  }

  /** Appends the creates attached since the last flush, shallowest path first: React runs
   * layout effects children before parents, but the worker rejects a child whose parent
   * it has never seen, and an ancestor's path is always shorter than its descendant's.
   * They land at the tail of the batch, so deletes buffered earlier in the commit still
   * lead and a re-parent tears down before it rebuilds. */
  private drainCreates(): void {
    if (this.queued.length === 0) return;
    const queued = this.queued;
    this.queued = [];
    queued.sort((a, b) => a.path.length - b.path.length);
    for (const entry of queued) {
      if (entry.phase !== "queued") continue;
      entry.phase = "live";
      const { path, state, type, transfer, pendingInvokes } = entry;
      entry.transfer = [];
      entry.pendingInvokes = [];
      this.outbound.send({ variant: "update", path, state, type }, transfer);
      for (const invoke of pendingInvokes) this.outbound.send(invoke);
      this.listeners.get(pathID(path))?.forEach((l) => l());
    }
  }

  private attachEntry(entry: Entry): void {
    if (entry.displaced || entry.phase !== "staged") return;
    const id = pathID(entry.path);
    const existing = this.entries.get(id);
    if (existing != null && existing !== entry) this.detachEntry(existing, true);
    entry.phase = "queued";
    this.entries.set(id, entry);
    this.queued.push(entry);
    this.outbound.schedule();
  }

  /** Returns `entry` to the staged phase. Sends a delete only when its create message
   * already went out; a component that never flushed does not exist on the worker.
   * `displaced` marks a same-path replacement, which must not clear the successor's
   * entry or invoke counter. */
  private detachEntry(entry: Entry, displaced = false): void {
    if (entry.phase === "staged") return;
    const id = pathID(entry.path);
    if (entry.phase === "live")
      this.outbound.send({ variant: "delete", path: entry.path });
    else {
      const idx = this.queued.indexOf(entry);
      if (idx !== -1) this.queued.splice(idx, 1);
    }
    entry.phase = "staged";
    entry.controller.abort(
      new Error(displaced ? "Component re-registered" : "Component deleted"),
    );
    // Drop queued invokes: the abort above already rejected their callers, so letting
    // them ride out on a later flush would run the side effect for a call the caller
    // was told had failed. `transfer` is kept, because it belongs to the state the next
    // flush still has to deliver.
    entry.pendingInvokes = [];
    // A fresh controller so a StrictMode remount can re-attach this same handle.
    entry.controller = new AbortController();
    if (displaced) {
      entry.displaced = true;
      return;
    }
    this.entries.delete(id);
    this.invokeTracker.clearCounter(id);
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
    // Drop pushes for a detached path — possible when delete/update messages cross
    // in flight, or after a StrictMode pseudo-unmount.
    if (entry == null) return;
    const parsed = zod.parse(entry.schema, state, { label: entry.type });
    entry.state = parsed;
    this.listeners.get(id)?.forEach((l) => l());
    entry.onReceiveRef?.current?.(parsed);
  }

  private buildHandle<
    StateSchema extends z.ZodType<state.State, state.State>,
    Methods extends aether.MethodsSchema,
  >(entry: Entry<StateSchema>, methodsSchema?: Methods): Handle<StateSchema, Methods> {
    const id = pathID(entry.path);

    const setState = (
      next: RawSetArg<StateSchema>,
      transfer: Transferable[] = [],
    ): void => {
      if (entry.displaced) return;
      const raw = state.executeSetter<z.input<StateSchema>, z.infer<StateSchema>>(
        next,
        entry.state,
      );
      entry.state = zod.parse(entry.schema, raw, { label: entry.type });
      // Before the create message flushes the worker has no component to update, so
      // the state rides along on that message instead and the transfer list merges
      // into it. Ownership of each Transferable still moves exactly once.
      if (entry.phase !== "live") entry.transfer.push(...transfer);
      else
        this.outbound.send(
          { variant: "update", path: entry.path, state: entry.state, type: entry.type },
          transfer,
        );
      this.listeners.get(id)?.forEach((l) => l());
    };

    // Invokes name a path the worker must already know. Before the create message
    // flushes there is nothing to call, so queue and let the flush send them in order.
    const sendInvoke = (msg: aether.MainInvokeRequest): void => {
      if (entry.phase === "live") this.outbound.send(msg);
      else entry.pendingInvokes.push(msg);
    };

    const invokeMethod = (method: string, args: unknown[]): void => {
      if (entry.displaced) return;
      sendInvoke({ variant: "invoke_request", path: entry.path, method, args });
    };

    const invokeMethodAsync = (
      method: string,
      args: unknown[],
      signal: AbortSignal = AbortSignal.timeout(
        new TimeSpan(this.config.invokeTimeout ?? DEFAULT_INVOKE_TIMEOUT).milliseconds,
      ),
    ): Promise<unknown> =>
      new Promise((resolve, reject) => {
        if (entry.displaced || entry.controller.signal.aborted)
          return reject(new Error("Component deleted"));
        const invokeKey = this.invokeTracker.nextKey(id);
        this.invokeTracker.track(
          invokeKey,
          resolve,
          reject,
          AbortSignal.any([signal, entry.controller.signal]),
        );
        sendInvoke({
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

    return {
      path: entry.path,
      methods,
      setState,
      getState: () => entry.state,
      subscribe: (listener) => this.subscribe(entry.path, listener),
      attach: () => this.attachEntry(entry),
      detach: () => this.detachEntry(entry),
    };
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
