// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type destructor, errors, observe, type record } from "@synnaxlabs/x";

import { type State } from "@/cache/state";
import {
  type ChannelListener,
  type InternalStore,
  ScopedUnaryStore,
  scopeStore,
  type StoreConfig,
  type Stores,
  type UnaryStore,
  type UnaryStoreConfig,
} from "@/cache/store";
import { createStreamer, type Streamer, type StreamOpener } from "@/cache/streamer";
import { type AsyncErrorHandler, type ErrorHandler } from "@/cache/types";

const checkSkip = (
  err: unknown,
  skip: errors.Matchable | errors.Matchable[] | undefined,
): boolean => {
  if (Array.isArray(skip)) return skip.some((matcher) => matcher.matches(err));
  return skip?.matches(err) ?? false;
};

const logError = (
  exc: unknown,
  message?: string,
  skip?: errors.Matchable | errors.Matchable[],
): void => {
  if (checkSkip(exc, skip)) return;
  if (message != null) console.error(message, errors.fromUnknown(exc));
  else console.error(errors.fromUnknown(exc));
};

/**
 * Default error handler used when the engine's consumer does not inject one:
 * reports to the console and continues.
 */
export const consoleErrorHandler: ErrorHandler = (excOrFunc, message, skip): void => {
  if (typeof excOrFunc !== "function") return logError(excOrFunc, message, skip);
  void (async () => {
    try {
      await excOrFunc();
    } catch (exc) {
      logError(exc, message, skip);
    }
  })();
};

/** Async counterpart of {@link consoleErrorHandler}. */
export const consoleAsyncErrorHandler: AsyncErrorHandler = async (
  excOrFunc,
  message,
  skip,
): Promise<void> => {
  if (typeof excOrFunc !== "function") return logError(excOrFunc, message, skip);
  try {
    await excOrFunc();
  } catch (exc) {
    logError(exc, message, skip);
  }
};

export interface EngineParams {
  /**
   * Opens the frame streamer used to receive change signals. Null constructs
   * a detached engine: purely local stores, no change stream ever opens, and
   * unknown store keys auto-register empty configs on first access. Used by
   * UI layers that need working stores before a cluster connection exists.
   */
  openStreamer: StreamOpener | null;
  /** Reports listener and reconciliation errors. Defaults to console logging. */
  handleError?: ErrorHandler;
  /** Async counterpart of handleError. Defaults to console logging. */
  handleAsyncError?: AsyncErrorHandler;
}

/**
 * Pure-mechanism core of the client cache: keyed stores, the change-stream
 * loop, connection epochs, and reconciliation. Holds zero domain knowledge —
 * domain clients register their store configs and expose typed sub-stores.
 *
 * The engine's change stream opens lazily: nothing touches the network until
 * the first {@link ensureStreaming} call. All stores must be registered
 * before streaming starts.
 */
export class Engine {
  private readonly configs: StoreConfig<any> = {};
  private readonly internal: InternalStore = {};
  private readonly epochObserver = new observe.Observer<number>();
  private errorSink: ErrorHandler;
  private asyncErrorSink: AsyncErrorHandler;
  // Stable delegating closures: stores capture these at registration, so a
  // later setErrorHandlers call reroutes every consumer.
  private readonly handleError: ErrorHandler = (excOrFunc: any, message?, skip?) =>
    this.errorSink(excOrFunc, message, skip);
  private readonly handleAsyncError: AsyncErrorHandler = async (
    excOrFunc: any,
    message?,
    skip?,
  ) => await this.asyncErrorSink(excOrFunc, message, skip);
  private readonly openStreamer: StreamOpener | null;
  private streamer: Streamer | null = null;
  private epochCount = 0;

  constructor({ openStreamer, handleError, handleAsyncError }: EngineParams) {
    this.openStreamer = openStreamer;
    this.errorSink = handleError ?? consoleErrorHandler;
    this.asyncErrorSink = handleAsyncError ?? consoleAsyncErrorHandler;
  }

  /**
   * Reroutes error reporting for every store, listener, and reconciliation
   * pass. Used by UI layers that surface errors through their own status
   * systems instead of the console.
   */
  setErrorHandlers(handleError: ErrorHandler, handleAsyncError: AsyncErrorHandler) {
    this.errorSink = handleError;
    this.asyncErrorSink = handleAsyncError;
  }

  /**
   * Registers a store under the given key. Must be called before streaming
   * starts; registering after {@link ensureStreaming} throws, as the new
   * store's channels would never be streamed.
   */
  registerStore<
    Key extends record.Key,
    Value extends State,
    SetExtra extends unknown | undefined = undefined,
  >(key: string, config: UnaryStoreConfig<any, Key, Value>): void {
    if (this.streamer != null)
      throw new Error(`cannot register store ${key} after streaming has started`);
    if (key in this.configs) throw new Error(`store ${key} is already registered`);
    this.configs[key] = config;
    this.internal[key] = new ScopedUnaryStore<Key, Value, SetExtra>(
      this.handleError,
      config.equal,
    );
  }

  /**
   * Registers additional channel listeners on an already-registered store.
   * Same pre-streaming constraint as {@link registerStore}. Used by machinery
   * (dispatch controllers) that needs the store to exist before it can build
   * its listener.
   */
  addListeners(key: string, ...listeners: ChannelListener<any>[]): void {
    if (this.streamer != null)
      throw new Error(`cannot add listeners to ${key} after streaming has started`);
    const config = this.configs[key];
    if (config == null) throw new Error(`store ${key} is not registered`);
    config.listeners.push(...listeners);
  }

  /**
   * Stable error handler that delegates to the current sink, so it remains
   * valid across {@link setErrorHandlers} calls. Injected into machinery
   * (dispatch controllers) built on this engine.
   */
  get errorHandler(): ErrorHandler {
    return this.handleError;
  }

  /** True when the engine has no stream source and stores are purely local. */
  get detached(): boolean {
    return this.openStreamer == null;
  }

  // Detached engines have no domain clients to register configs, so unknown
  // keys become empty local stores on first access.
  private resolve(key: string): ScopedUnaryStore | null {
    const store = this.internal[key];
    if (store != null || !this.detached) return store ?? null;
    this.registerStore(key, { listeners: [] });
    return this.internal[key];
  }

  /**
   * Returns the store registered under the given key, bound to the given
   * scope. Throws when the key was never registered.
   */
  store<
    Key extends record.Key,
    Value extends State,
    SetExtra extends unknown | undefined = undefined,
  >(key: string, scope: string = ""): UnaryStore<Key, Value, SetExtra> {
    const store = this.resolve(key);
    if (store == null) throw new Error(`store ${key} is not registered`);
    return store.scope(scope) as UnaryStore<Key, Value, SetExtra>;
  }

  /** Returns every registered store bound to the given scope. */
  scopedStores<ScopedStores extends Stores>(scope: string): ScopedStores {
    return scopeStore<ScopedStores>(this.internal, scope);
  }

  /**
   * Returns the raw, scope-free store registered under the given key. For
   * in-package machinery (dispatch controllers) that threads caller scopes
   * through its own operations; everything else uses {@link store}.
   */
  unscoped<
    Key extends record.Key,
    Value extends State,
    SetExtra extends unknown | undefined = undefined,
  >(key: string): ScopedUnaryStore<Key, Value, SetExtra> {
    const store = this.resolve(key);
    if (store == null) throw new Error(`store ${key} is not registered`);
    return store as unknown as ScopedUnaryStore<Key, Value, SetExtra>;
  }

  /**
   * Ensures the change stream is open, opening it on first call. Callers that
   * populate or read stores await this first so no change is missed between a
   * fetch and the stream opening.
   */
  async ensureStreaming(): Promise<void> {
    const opener = this.openStreamer;
    if (opener == null) return;
    this.streamer ??= createStreamer({
      openStreamer: opener,
      storeConfig: this.configs,
      handleError: this.handleAsyncError,
      store: scopeStore(this.internal, ""),
      onOpen: () => {
        this.epochCount = 1;
        this.epochObserver.notify(this.epochCount);
      },
      onReopen: () => this.bumpEpoch(),
    });
    await this.streamer.demand();
  }

  /**
   * The connection epoch: 0 before streaming starts, 1 once live, incremented
   * on every reconnect. Changes may have been missed between epochs.
   */
  get epoch(): number {
    return this.epochCount;
  }

  /** Subscribes to epoch changes. Returns a destructor that unsubscribes. */
  onEpoch(callback: (epoch: number) => void): destructor.Destructor {
    return this.epochObserver.onChange(callback);
  }

  /**
   * Re-checks every cached entry against the cluster: refreshes entries that
   * still exist and tombstones entries that vanished. Runs automatically
   * after every reconnect for stores whose config provides a refetch.
   */
  async reconcile(): Promise<void> {
    await Promise.all(
      Object.entries(this.configs).map(async ([key, config]) => {
        const { refetch } = config;
        if (refetch == null) return;
        const store = this.internal[key].scope("");
        const keys = store.list().map((value: record.Keyed<record.Key>) => value.key);
        if (keys.length === 0) return;
        await this.handleAsyncError(async () => {
          const values = await refetch(keys);
          const present = new Set(values.map(({ key: k }) => k));
          const vanished = keys.filter((k) => !present.has(k));
          if (vanished.length > 0) store.delete(vanished);
          if (values.length > 0) store.set(values);
        }, `Failed to reconcile ${key} cache`);
      }),
    );
  }

  private bumpEpoch(): void {
    this.epochCount++;
    this.epochObserver.notify(this.epochCount);
    void this.handleAsyncError(
      async () => await this.reconcile(),
      "Failed to reconcile caches after reconnect",
    );
  }

  /** Closes the change stream. A no-op when streaming never started. */
  async close(): Promise<void> {
    if (this.streamer == null) return;
    await this.streamer.close();
  }
}

/**
 * Cache control surface exposed on the client as `client.cache`. Not a data
 * access path: per-domain stores on the domain clients remain the only way to
 * read cached records.
 */
export class Handle {
  private readonly engine_: Engine | null;

  constructor(engine: Engine | null) {
    this.engine_ = engine;
  }

  /** Whether caching was enabled at client construction. */
  get enabled(): boolean {
    return this.engine_ != null;
  }

  /**
   * The connection epoch: 0 before change streaming starts, 1 once live,
   * incremented on every reconnect. Always 0 when the cache is disabled.
   */
  get epoch(): number {
    return this.engine_?.epoch ?? 0;
  }

  /**
   * Subscribes to connection epoch changes. Returns a destructor that
   * unsubscribes. A no-op when the cache is disabled.
   */
  onEpoch(callback: (epoch: number) => void): destructor.Destructor {
    if (this.engine_ == null) return () => {};
    return this.engine_.onEpoch(callback);
  }

  /**
   * The engine backing this client's caches. Intended for binding layers
   * (flux) rather than application code.
   * @throws when the cache was disabled at client construction.
   */
  get engine(): Engine {
    if (this.engine_ == null)
      throw new Error("cache is disabled on this client (cache: false)");
    return this.engine_;
  }

  /**
   * Reroutes error reporting for every store, listener, and reconciliation
   * pass. Used by UI layers that surface errors through their own status
   * systems instead of the console. A no-op when the cache is disabled.
   */
  setErrorHandlers(handleError: ErrorHandler, handleAsyncError: AsyncErrorHandler) {
    this.engine_?.setErrorHandlers(handleError, handleAsyncError);
  }

  /**
   * Ensures the change stream is open, opening it on first call. A no-op when
   * the cache is disabled.
   */
  async ensureStreaming(): Promise<void> {
    await this.engine_?.ensureStreaming();
  }

  /** Closes the change stream. A no-op when disabled or never streamed. */
  async close(): Promise<void> {
    await this.engine_?.close();
  }
}
