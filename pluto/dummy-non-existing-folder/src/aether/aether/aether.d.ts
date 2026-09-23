import { alamos } from "@synnaxlabs/alamos";
import { state } from "@synnaxlabs/x";
import { z } from "zod";
import { type MainInvokeRequest, type MainUpdateRequest, type Sender, type WorkerComms } from "./message";
export { Batcher, createMockPair, type MainComms, type MainMessage, type WorkerComms, type WorkerMessage, wrapWorker, wrapWorkerScope, } from "./message";
/** Factory passed to {@link Component._updateState} so a {@link Composite} can lazily
 * instantiate a missing child, wiring `parent` so the child can resolve inherited
 * context up the tree. */
interface CreateComponent {
    (parent: Node): Component;
}
export interface UpdateStateParams extends Pick<MainUpdateRequest, "path" | "type" | "state"> {
    create: CreateComponent;
}
export interface InvokeMethodParams extends Omit<MainInvokeRequest, "variant"> {
}
/** A node in the worker-side Aether tree, identified by `type` (class) and `key`
 * (instance). The `_`-prefixed methods are the internal tree protocol — implementations
 * live on {@link Leaf} and {@link Composite}; subclasses should not call them. */
export interface Component {
    type: string;
    key: string;
    toString(): string;
    _updateState: (params: UpdateStateParams) => void;
    _delete: (path: readonly string[]) => void;
    _invokeMethod: (params: InvokeMethodParams) => void;
}
/** Required constructor arguments for every aether {@link Component}. */
export interface ComponentConstructorProps {
    /** Path from the root; its last element becomes the component's `key`. */
    path: readonly string[];
    type: string;
    /** Channel for posting messages back to the main thread. */
    sender: Sender;
    instrumentation: alamos.Instrumentation;
    /** The parent node, used to resolve inherited context up the tree. `null` only at the
     * tree root. */
    parent: Node | null;
}
/** Constructor signature every entry in a {@link ComponentRegistry} must satisfy. */
export interface ComponentConstructor {
    new (props: ComponentConstructorProps): Component;
}
/** Passed to {@link Leaf.afterUpdate} / {@link Leaf.afterDelete}: read inherited
 * parent-context values and publish new ones to descendants. */
export interface Context {
    /** Returns the parent-context value at `key`, typed as `P`. The cast is unchecked.
     * Throws {@link NotFoundError} if the key is absent. */
    get<P>(key: string): P;
    /** Returns the parent-context value at `key`, typed as `P`, or `null` if absent. The
     * cast is unchecked. */
    getOptional<P>(key: string): P | null;
    /** Reports whether a parent-context value exists at `key`. Does not see values set on
     * this component via {@link Context.set}; use {@link Context.wasSetPreviously} for
     * that. */
    has(key: string): boolean;
    /** Publishes `value` at `key` to this component's descendants. The publishing
     * component itself does not see the value via {@link Context.get} — overriding a
     * parent's key keeps the parent's value visible locally. If `trigger` (default
     * `true`), descendants are notified of the change. */
    set(key: string, value: unknown, trigger?: boolean): void;
    /** Reports whether this component has previously published a value at `key`.
     * Counterpart to {@link Context.has}, which sees only parent values. */
    wasSetPreviously(key: string): boolean;
}
/** Schema describing the invocable methods of a component. Keys are method names;
 * values are the per-method `z.ZodFunction` describing inputs and output. */
export type MethodsSchema = Record<string, z.ZodFunction>;
/** Empty {@link MethodsSchema} for components that do not invoke any methods. */
export type EmptyMethodsSchema = Record<string, never>;
/** Worker-side handler signatures derived from a {@link MethodsSchema}. A schema with a
 * sync output may be implemented sync or async; a schema with a `Promise<T>` output
 * must be implemented async (no double-wrap). */
export type HandlersFromSchema<T> = {
    [K in keyof T]: T[K] extends z.ZodType<infer F> ? F extends (...params: infer A) => infer R ? R extends Promise<unknown> ? (...params: A) => R : (...params: A) => R | Promise<R> : never : never;
};
/** Reports whether `schema` describes a fire-and-forget method (output is
 * `void`/`never`/`unknown` — i.e. no response is required). */
export declare const isFireAndForget: <F extends z.ZodFunction>(schema: F) => boolean;
/** Main-side caller signatures derived from a {@link MethodsSchema}. Void outputs
 * become fire-and-forget; non-void outputs return `Promise<R>`. */
export type CallersFromSchema<T> = {
    [K in keyof T]: T[K] extends z.ZodType<infer F> ? F extends (...params: infer A) => infer R ? R extends void ? (...params: A) => void : (...params: A) => Promise<Awaited<R>> : never : never;
};
/**
 * Base class for every node in the worker-side Aether tree. {@link Node} owns tree
 * membership (parent, depth, deletion) and the context-propagation machinery: a node
 * publishes context values to its descendants via {@link Context.set} and subscribes to
 * values published by its ancestors via {@link Context.get}. {@link Leaf} layers typed
 * state and invocable methods on top; {@link Composite} adds a children registry. The
 * context internals are members of this shared base so a node can reach its parent's
 * and subscribers' state directly. A childless node (a pure {@link Leaf}) never gains
 * descendants, so its provider-side {@link ctxSubscribers} stays empty; only nodes that
 * own children ({@link Composite}s) ever accumulate subscribers.
 */
export declare abstract class Node implements Component {
    readonly type: string;
    /** Path from the root; this node's identity. Its last element is {@link key}. */
    protected readonly path: readonly string[];
    /** Channel for posting messages back to the main thread. */
    protected readonly sender: Sender;
    readonly instrumentation: alamos.Instrumentation;
    private _deleted;
    /** The parent node, used to resolve inherited context. `null` only at the root. */
    private _parent;
    /** Distance from the root (root is 0), derived from the parent chain at construction.
     * Providers are always strict ancestors of their subscribers, so depth is a valid
     * topological order for context propagation. */
    private readonly _depth;
    /** Context values this node has published to its descendants. A node is the provider
     * of every key in this map for the descendants nearest to it. */
    protected readonly childCtxValues: Map<string, unknown>;
    /** Keys in {@link childCtxValues} this node changed during the current
     * {@link afterUpdate}. Drives selective propagation; cleared at the start of each run. */
    private readonly childCtxChangedKeys;
    /** Provider side: for each key this node publishes, the descendants currently
     * subscribed to it (i.e. those that resolved this node as their nearest provider of
     * that key on their last {@link afterUpdate}). Empty for a childless {@link Leaf}. */
    private readonly ctxSubscribers;
    /** Consumer side: keys this node read on its last {@link afterUpdate}, mapped to the
     * provider it resolved each to. Diffed every run to keep subscriptions current. */
    private readonly ctxSubscriptions;
    /** Accumulator for the in-progress {@link afterUpdate}: keys read so far mapped to
     * their resolved provider (or `null` if unresolved). Non-`null` only while
     * {@link afterUpdate} is running, so reads outside that window are not tracked. */
    private ctxReads;
    /** {@link toString} and the span labels derived from it, built once. `type` and `path`
     * never change, and every update would otherwise rebuild these strings only for noop
     * instrumentation to discard them. */
    private readonly label;
    private readonly afterUpdateSpan;
    protected readonly updateStateSpan: string;
    private readonly deleteSpan;
    /** Built on first use and reused. The object closes over `this` alone, so a fresh one
     * per {@link afterUpdate} would be six identical allocations per update. */
    private cachedCtx;
    constructor({ path, type, sender, instrumentation, parent, }: ComponentConstructorProps);
    /** Local name: the last element of the path, unique only among siblings. */
    get key(): string;
    /** True once the node has been spliced out of the tree. Stays true for the remainder
     * of the instance's lifetime. */
    get deleted(): boolean;
    toString(): string;
    private get ctx();
    /** Resolves `key` to the nearest ancestor publishing it (self excluded) and records
     * the read against the in-progress {@link afterUpdate} so a subscription is formed.
     * Returns the provider, or `null` if unresolved. Self is excluded so a node overriding
     * a parent's key still reads the parent's value via {@link Context.get}. The read is
     * not recorded outside an {@link afterUpdate} (e.g. during {@link afterDelete}), where
     * {@link ctxReads} is `null`. */
    private readCtx;
    /** Runs {@link afterUpdate} with context-read tracking active, then reconciles the
     * resulting subscriptions. Used for both state-driven updates and context-driven
     * re-runs. */
    protected runAfterUpdate(): void;
    /** Diffs the keys read this pass against the prior subscriptions, updating both this
     * node's {@link ctxSubscriptions} and each provider's {@link ctxSubscribers}. Re-runs
     * every pass so conditional reads subscribe/unsubscribe as they appear and disappear. */
    private reconcileSubscriptions;
    /** Re-runs {@link afterUpdate} on every node subscribed to a key this node just
     * changed, cascading through their own changed keys. Subscribers run shallowest first
     * and at most once: since every provider is a strict ancestor of its subscribers,
     * depth order guarantees each node runs after all of its changed providers have
     * settled. */
    protected propagate(): void;
    /** Internal: routes delete from the tree. Subclasses other than {@link Composite}
     * must not call this. */
    _delete(path: readonly string[]): void;
    /** Removes this node from every provider's subscriber set and drops its own. For a
     * {@link Composite}, the child-first delete recursion has already torn down (and
     * unsubscribed) descendants before this runs. */
    private unsubscribeAll;
    /** Hook fired after the node's state, context, or both are updated. Override to react
     * to changes. {@link Leaf.state}, {@link Leaf.prevState}, {@link Leaf.internal}, and
     * the provided {@link Context} are all available. */
    afterUpdate(_: Context): void;
    /** Hook fired after the node is spliced out of the tree. Override for cleanup
     * (unsubscriptions, resource release, etc.). {@link deleted} is `true` here;
     * {@link Leaf.state} and {@link Leaf.prevState} are still readable. */
    afterDelete(_: Context): void;
    protected validatePath(path: readonly string[]): void;
    /** Internal: routes state updates from the tree. Subclasses other than
     * {@link Composite} must not call this. */
    abstract _updateState(params: UpdateStateParams): void;
    /** Internal: dispatches a method invocation addressed at this node. */
    abstract _invokeMethod(params: InvokeMethodParams): void;
}
/**
 * Base class for childless aether components. The corresponding React component must
 * not have descendants that use Aether — use {@link Composite} for those. Subclasses
 * define a Zod {@link schema} for their state and override {@link afterUpdate} / {@link
 * afterDelete} for lifecycle behavior. To expose invocable methods, set {@link methods}
 * to a {@link MethodsSchema} and `implements HandlersFromSchema<typeof schema>` —
 * method names on the class must match keys in the schema.
 *
 * @example
 * ```typescript
 * const buttonMethodsZ = {
 *   onMouseDown: z.function(),
 *   onMouseUp: z.function().returns(z.number()),
 * };
 *
 * class Button extends Leaf<typeof stateZ, {}, typeof buttonMethodsZ>
 *   implements HandlersFromSchema<typeof buttonMethodsZ> {
 *   schema = stateZ;
 *   methods = buttonMethodsZ;
 *   onMouseDown(): void { ... }
 *   onMouseUp(): number { return 42; }
 * }
 * ```
 */
export declare abstract class Leaf<StateSchema extends z.ZodType<state.State>, InternalState extends {} = {}, Methods extends MethodsSchema = EmptyMethodsSchema> extends Node {
    private readonly _internalState;
    private _state;
    private _prevState;
    /** Zod schema for the component's state. Must be defined by every subclass. */
    schema: StateSchema | undefined;
    /** Optional schema enabling {@link Component._invokeMethod}. Each key must correspond
     * to a method on the class with a matching signature. */
    methods: Methods | undefined;
    private _methodImplementations;
    constructor(props: ComponentConstructorProps);
    private initializeMethods;
    private get _schema();
    /** Sets the state on the worker side and propagates the change to the corresponding
     * component on the main thread. Accepts either the next state value or a pure
     * function deriving it from the current state. */
    setState(next: state.SetArg<z.infer<StateSchema>>): void;
    /** The component's current parsed state. Throws if read before the first update has
     * been applied. */
    get state(): z.infer<StateSchema>;
    /** Scratch space the component can use to hold derived or cached values that should
     * survive across updates without being part of the parsed state. */
    get internal(): InternalState;
    /** State prior to the most recent update. Throws when accessed before the second
     * update — first-update consumers should use {@link state} only. */
    get prevState(): z.infer<StateSchema>;
    /** Internal: routes state updates from the tree. Subclasses other than
     * {@link Composite} must not call this. */
    _updateState({ path, state }: UpdateStateParams): void;
    protected handleInvokeError({ method, key, args }: InvokeMethodParams, error: unknown): void;
    _invokeMethod(params: InvokeMethodParams): void;
}
/**
 * Base class for aether components that own children. Extends {@link Leaf} with a typed
 * children registry, descendant context propagation, and routing for
 * updates/deletes/invokes addressed at descendants. */
export declare abstract class Composite<StateSchema extends z.ZodType<state.State>, InternalState extends {} = {}, ChildComponents extends Component = Component, M extends MethodsSchema = EmptyMethodsSchema> extends Leaf<StateSchema, InternalState, M> implements Component {
    private readonly _children;
    /** Snapshot of the children, in insertion order. */
    get children(): readonly ChildComponents[];
    /** Returns the child at `key`, or `null` if none. The `T` cast is unchecked — callers
     * must know which subtype they are looking up. */
    getChild<T extends ChildComponents = ChildComponents>(key: string): T | null;
    /** Returns the children whose `type` matches one of `types`. The `T` cast is
     * unchecked — callers must align the type filter with the asserted return type. */
    protected childrenOfType<T extends ChildComponents = ChildComponents>(...types: Array<T["type"]>): readonly T[];
    _updateState(params: UpdateStateParams): void;
    /** Deletes every child, running each full delete lifecycle. The component itself
     * stays alive, so the tree can be rebuilt by later updates. */
    clearChildren(): void;
    _delete(path: readonly string[]): void;
    /** Walks down the tree by `path` (relative to this composite — i.e. excluding this
     * component's own key) and returns the component at the leaf, or `null` if any step
     * is missing. */
    findChildAtPath(path: string[]): Component | null;
    private parsePath;
}
/** Map from component `type` to constructor, consulted by {@link Root} to instantiate
 * new components on demand. */
export type ComponentRegistry = Record<string, ComponentConstructor>;
declare const aetherRootState: z.ZodObject<{}, z.core.$strip>;
/** Arguments for {@link Root.render}. */
export interface RootProps {
    /** Map of component types this tree can instantiate. */
    registry: ComponentRegistry;
    /** Bidirectional channel with the main thread; usually {@link wrapWorkerScope} in
     * production or {@link createMockPair} in tests. */
    worker?: WorkerComms;
    /** Instrumentation used for logging, tracing, etc. */
    instrumentation?: alamos.Instrumentation;
}
/** Top-level node of the worker-side Aether tree. Owns the {@link comms} channel and
 * the component {@link registry}; routes messages from the main thread to the
 * appropriate descendant. Construct via {@link Root.render}. */
export declare class Root extends Composite<typeof aetherRootState> {
    private static readonly TYPE;
    private static readonly KEY;
    private readonly comms;
    private readonly registry;
    schema: z.ZodObject<{}, z.core.$strip>;
    constructor({ worker, instrumentation, registry, }: RootProps);
    /** Constructs a new aether tree and starts handling messages on `comms`. Any
     * synchronous error thrown while processing a message is caught and surfaced on the
     * main thread as a {@link WorkerNotifyErrorRequest}. */
    static render(props: RootProps): Root;
    private handle;
    private invokeAtPath;
    private create;
}
/** Convenience binding of {@link Root.render}: constructs an aether tree and binds it
 * to the provided `comms`. */
export declare const render: typeof Root.render;
//# sourceMappingURL=aether.d.ts.map