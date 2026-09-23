import { type CrudeTimeSpan, type destructor, state } from "@synnaxlabs/x";
import { type z } from "zod";
import { aether } from "./aether";
/** Setter argument accepted by {@link Handle.setState}: a new state value or a function
 * that derives one from the previous value. Always the schema's input type — the value
 * is parsed before being stored, so post-transform output types shouldn't be passed. */
export type RawSetArg<StateSchema extends z.ZodType<state.State, state.State>> = state.SetArg<z.input<StateSchema>, z.infer<StateSchema>>;
type Listener = () => void;
/** Arguments accepted by {@link Store.stage}. */
export interface StageParams<StateSchema extends z.ZodType<state.State, state.State>, Methods extends aether.MethodsSchema = aether.EmptyMethodsSchema> {
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
    onReceiveRef?: {
        current: ((state: z.infer<StateSchema>) => void) | undefined;
    };
}
/** Per-component operations returned by {@link Store.stage}: typed setState, the
 * attach/detach pair, state reads, subscriptions, and method callers. Scoped to the
 * staging that produced it: once a same-path attach displaces that entry, every
 * operation no-ops (async invokes reject).
 *
 * Every field is built once with the handle and never replaced, so a React caller can
 * hand them straight to hooks without wrapping them in `useCallback`. */
export interface Handle<StateSchema extends z.ZodType<state.State, state.State>, Methods extends aether.MethodsSchema = aether.EmptyMethodsSchema> {
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
export declare class Store {
    /** Attached entries keyed by component identity ({@link pathID}). */
    private entries;
    /** Subscribers keyed by component identity. Listeners persist across an entry's
     * detach-attach cycle so subscriptions wired during a StrictMode pseudo-remount stay
     * live once the real attach fires. */
    private listeners;
    /** Entries attached since the last flush, in attach order. */
    private queued;
    private readonly outbound;
    /** Active worker comms. {@link NOOP_WORKER} when `workerEnabled: false` or between
     * {@link dispose} and the next send (lazy re-attach). */
    private worker;
    private invokeTracker;
    /** Most recent worker-reported error. Buffered so it survives the gap between a
     * synchronous worker push and the Provider's subscription. */
    private currentError;
    private errorListeners;
    /** Raw {@link Worker} this store spawned; `null` for externally-injected comms
     * (tests) which are owned by the caller. */
    private ownedWorker;
    /** Config retained so {@link connect} can lazily rebuild the worker after
     * {@link dispose} — required for the StrictMode reused-fiber cycle. */
    private config;
    /** Throws {@link ValidationError} if `workerEnabled` is true (the default) and
     * neither `worker` nor `workerURL` is provided. */
    constructor(config?: StoreConfig);
    /** Spawns the worker (when configured with a `workerURL`) and starts handling its
     * messages. Called on the {@link Aether.Provider}'s first commit and again by any
     * send after {@link dispose}. Deferred out of the constructor so a Provider render
     * React discards never spawns a worker. Idempotent. */
    connect(): void;
    private setError;
    /** Clears the worker-side tree, detaches the worker handler, terminates any owned
     * `Worker`, and aborts in-flight invokes. The store remains usable: a subsequent
     * send lazily re-attaches via a fresh `Worker`. Idempotent. */
    dispose(): void;
    /** Returns the most recent worker-reported error, or `null` if none. Suitable for
     * `useSyncExternalStore`. */
    getError(): Error | null;
    /** Subscribes to worker-error notifications. The listener fires whenever a new error
     * is buffered; re-read via {@link getError}. Returns an unsubscribe function. */
    subscribeError(listener: Listener): () => void;
    /** Subscribes to state changes for the component at `path`. Fires on both worker
     * pushes and local {@link Handle.setState} calls. Subscriptions persist across
     * detach-attach cycles. Returns an unsubscribe function. */
    subscribe(path: readonly string[], listener: Listener): () => void;
    /** Validates and parses `params` into a handle owned by the caller. The store keeps
     * no reference and the worker is not told anything until {@link Handle.attach} runs,
     * so a component staged by a render React discards is reclaimed with the caller's
     * own reference. */
    stage<StateSchema extends z.ZodType<state.State, state.State>, Methods extends aether.MethodsSchema>(params: StageParams<StateSchema, Methods>): Handle<StateSchema, Methods>;
    /** Appends the creates attached since the last flush, shallowest path first: React runs
     * layout effects children before parents, but the worker rejects a child whose parent
     * it has never seen, and an ancestor's path is always shorter than its descendant's.
     * They land at the tail of the batch, so deletes buffered earlier in the commit still
     * lead and a re-parent tears down before it rebuilds. */
    private drainCreates;
    private attachEntry;
    /** Returns `entry` to the staged phase. Sends a delete only when its create message
     * already went out; a component that never flushed does not exist on the worker.
     * `displaced` marks a same-path replacement, which must not clear the successor's
     * entry or invoke counter. */
    private detachEntry;
    private handleWorkerMessage;
    private buildHandle;
}
export {};
//# sourceMappingURL=store.d.ts.map