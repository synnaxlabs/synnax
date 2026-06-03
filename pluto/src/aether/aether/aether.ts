// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { alamos } from "@synnaxlabs/alamos";
import { NotFoundError, UnexpectedError, ValidationError } from "@synnaxlabs/client";
import { deep, errors, type record, shallow, zod } from "@synnaxlabs/x";
import { z } from "zod";

import {
  type MainInvokeRequest,
  type MainMessage,
  type MainUpdateRequest,
  type Sender,
  type WorkerComms,
  wrapWorkerScope,
} from "@/aether/aether/message";

export {
  createMockPair,
  type MainComms,
  type MainMessage,
  type WorkerComms,
  type WorkerMessage,
  wrapWorker,
  wrapWorkerScope,
} from "@/aether/aether/message";
import { state } from "@/state";

const newTreeError = (e: unknown, pathOrMessage?: string): Error => {
  if (e instanceof Error) {
    e.message = `[${pathOrMessage}] - ${e.message}`;
    return e;
  }
  return new Error(`[${pathOrMessage ?? "unknown error"}] - ${String(e)}`, {
    cause: e,
  });
};

/** Read-only view over context values, exposed to {@link Component.afterUpdate}
 * for propagating context to descendants. */
export interface ContextMap extends Pick<Map<string, unknown>, "get" | "forEach"> {}

/** Factory passed to {@link Component._updateState} so a {@link Composite} can lazily
 * instantiate a missing child with the right parent context. */
interface CreateComponent {
  (initialParentCtxValues: ContextMap): Component;
}

export interface UpdateStateParams extends Pick<
  MainUpdateRequest,
  "path" | "type" | "state"
> {
  create: CreateComponent;
}

export interface InvokeMethodParams extends Omit<MainInvokeRequest, "variant"> {}

/** A node in the worker-side Aether tree, identified by `type` (class) and `key`
 * (instance). The `_`-prefixed methods are the internal tree protocol — implementations
 * live on {@link Leaf} and {@link Composite}; subclasses should not call them. */
export interface Component {
  type: string;
  key: string;
  toString(): string;
  _updateState: (params: UpdateStateParams) => void;
  _updateContext: (values: ContextMap) => void;
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
  /** Parent context inherited at creation time. `null` only at the tree root. */
  parentCtxValues: ContextMap | null;
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
  [K in keyof T]: T[K] extends z.ZodType<infer F>
    ? F extends (...params: infer A) => infer R
      ? R extends Promise<unknown>
        ? (...params: A) => R
        : (...params: A) => R | Promise<R>
      : never
    : never;
};

/** Reports whether `schema` describes a fire-and-forget method (output is
 * `void`/`never`/`unknown` — i.e. no response is required). */
export const isFireAndForget = <F extends z.ZodFunction>(schema: F): boolean => {
  const outputType = schema._zod.def.output;
  return (
    outputType instanceof z.ZodVoid ||
    outputType instanceof z.ZodNever ||
    outputType instanceof z.ZodUnknown
  );
};

/** Main-side caller signatures derived from a {@link MethodsSchema}. Void outputs
 * become fire-and-forget; non-void outputs return `Promise<R>`. */
export type CallersFromSchema<T> = {
  [K in keyof T]: T[K] extends z.ZodType<infer F>
    ? F extends (...params: infer A) => infer R
      ? R extends void
        ? (...params: A) => void
        : (...params: A) => Promise<Awaited<R>>
      : never
    : never;
};

/**
 * Base class for childless aether components. The corresponding React component must
 * not have descendants that use Aether — use {@link Composite} for those.
 *
 * Subclasses define a Zod {@link schema} for their state and override
 * {@link afterUpdate} / {@link afterDelete} for lifecycle behavior. To expose invocable
 * methods, set {@link methods} to a {@link MethodsSchema} and `implements
 * HandlersFromSchema<typeof schema>` — method names on the class must match keys in the
 * schema.
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
export abstract class Leaf<
  StateSchema extends z.ZodType<state.State>,
  InternalState extends {} = {},
  Methods extends MethodsSchema = EmptyMethodsSchema,
> implements Component {
  readonly type: string;
  /** Path from the root; this component's identity. Its last element is {@link key}. */
  private readonly path: readonly string[];
  /** Channel for posting messages back to the main thread. */
  protected readonly sender: Sender;
  private readonly _internalState: InternalState;
  private _state: z.infer<StateSchema> | undefined;
  private _prevState: z.infer<StateSchema> | undefined;
  private _deleted: boolean = false;
  /** Context values inherited from this component's parent. Read-only. */
  protected readonly parentCtxValues: Map<string, any>;
  /** Context values this component has published to its descendants. */
  protected readonly childCtxValues: Map<string, any>;
  /** Keys in {@link childCtxValues} whose changes should trigger descendants to re-run
   * {@link afterUpdate}. Cleared at the start of each update. */
  protected readonly childCtxChangedKeys: Set<string>;
  readonly instrumentation: alamos.Instrumentation;

  /** Zod schema for the component's state. Must be defined by every subclass. */
  schema: StateSchema | undefined = undefined;

  /** Optional schema enabling {@link Component._invokeMethod}. Each key must correspond
   * to a method on the class with a matching signature. */
  methods: Methods | undefined = undefined;
  private _methodImplementations: Record<
    string,
    (...args: unknown[]) => Promise<unknown>
  > | null = null;

  constructor({
    path,
    type,
    sender,
    instrumentation,
    parentCtxValues,
  }: ComponentConstructorProps) {
    this.type = type;
    this.path = path;
    this.sender = sender;
    this._internalState = {} as InternalState;
    this.instrumentation = instrumentation.child(this.toString());
    this.parentCtxValues = new Map();
    parentCtxValues?.forEach((value, key) => this.parentCtxValues.set(key, value));
    this.childCtxValues = new Map();
    this.childCtxChangedKeys = new Set();
  }

  /** Local name: the last element of the path, unique only among siblings. */
  get key(): string {
    return this.path[this.path.length - 1];
  }

  private initializeMethods() {
    if (this.methods == null || this._methodImplementations != null) return;
    this._methodImplementations = {};
    for (const [name, schema] of Object.entries(this.methods)) {
      const methodFn = this[name as keyof this];
      if (typeof methodFn !== "function")
        throw new Error(`Method ${name} is not a function`);
      this._methodImplementations[name] = isFireAndForget(schema)
        ? schema.implement(methodFn.bind(this))
        : schema.implementAsync(methodFn.bind(this));
    }
  }

  private get _schema(): StateSchema {
    if (this.schema == null)
      throw new ValidationError(
        `[AetherLeaf] - expected subclass to define component schema, but none was found.
        Make sure to define a property 'schema' on the class.`,
      );
    return this.schema;
  }

  /** Sets the state on the worker side and propagates the change to the corresponding
   * component on the main thread. Accepts either the next state value or a pure
   * function deriving it from the current state. */
  setState(next: state.SetArg<z.infer<StateSchema>>): void {
    const nextState = state.executeSetter(next, this.state);
    this._prevState = shallow.copy(this._state);
    this._state = zod.parse(this._schema, nextState, { label: this.toString() });
    this.sender.send({ variant: "update", path: this.path, state: this._state });
  }

  /** The component's current parsed state. Throws if read before the first update has
   * been applied. */
  get state(): z.infer<StateSchema> {
    if (this._state == null)
      throw new UnexpectedError(
        `[aether.leaf] state not defined in ${this.toString()}`,
      );
    return this._state;
  }

  /** Scratch space the component can use to hold derived or cached values that should
   * survive across updates without being part of the parsed state. */
  get internal(): InternalState {
    return this._internalState;
  }

  /** State prior to the most recent update. Throws when accessed before the second
   * update — first-update consumers should use {@link state} only. */
  get prevState(): z.infer<StateSchema> {
    if (this._prevState === undefined) throw new Error("prevState not defined");
    return this._prevState;
  }

  /** True once the component has been spliced out of the tree. Stays true for the
   * remainder of the instance's lifetime. */
  get deleted(): boolean {
    return this._deleted;
  }

  private get ctx(): Context {
    return {
      get: (key: string) => {
        const res = this.parentCtxValues.get(key);
        if (res === undefined)
          throw new NotFoundError(
            `Context value for ${key} not found on ${this.toString()}`,
          );
        return res;
      },
      getOptional: (key: string) => this.parentCtxValues.get(key),
      has: (key: string) => this.parentCtxValues.has(key),
      wasSetPreviously: (key: string) => this.childCtxValues.has(key),
      set: (key: string, value: unknown, trigger: boolean = true) => {
        this.childCtxValues.set(key, value);
        if (trigger) this.childCtxChangedKeys.add(key);
      },
    };
  }

  toString(): string {
    return `${this.type}(${this.key})`;
  }

  /** Internal: routes state updates from the tree. Subclasses other than
   * {@link Composite} must not call this. */
  _updateState({ path, state }: UpdateStateParams): void {
    if (this.deleted) return;
    try {
      this.initializeMethods();
      const endSpan = this.instrumentation.T.debug(`${this.toString()}:updateState`);
      this.validatePath(path);
      const state_ = zod.parse(this._schema, state, { label: this.toString() });
      if (this._state != null)
        this.instrumentation.L.debug("updating state", () => ({
          diff: deep.difference(this.state as record.Unknown, state_ as record.Unknown),
        }));
      else this.instrumentation.L.debug("setting initial state", { state, path });
      this._prevState = this._state ?? state_;
      this._state = state_;
      this.afterUpdate(this.ctx);
      endSpan();
    } catch (e) {
      throw newTreeError(e, `${this.toString()}.updateState`);
    }
  }

  _updateContext(values: ContextMap): void {
    try {
      const endSpan = this.instrumentation.T.debug(`${this.toString()}:updateContext`);
      values.forEach((value, key) => this.parentCtxValues.set(key, value));
      this.afterUpdate(this.ctx);
      endSpan();
    } catch (e) {
      throw newTreeError(e, `${this.type}.${this.key}.updateContext`);
    }
  }

  /** Internal: routes delete from the tree. Subclasses other than {@link Composite}
   * must not call this. */
  _delete(path: readonly string[]): void {
    try {
      const endSpan = this.instrumentation.T.debug(`${this.toString()}:delete`);
      this.validatePath(path);
      this._deleted = true;
      this.afterDelete(this.ctx);
      endSpan();
    } catch (e) {
      throw newTreeError(e, `[${this.toString()}:delete]`);
    }
  }

  /** Hook fired after the component's state, context, or both are updated. Override to
   * react to changes. {@link state}, {@link prevState}, {@link internal}, and the
   * provided {@link Context} are all available. */
  afterUpdate(_: Context): void {}

  /** Hook fired after the component is spliced out of the tree. Override for cleanup
   * (unsubscriptions, resource release, etc.). {@link deleted} is `true` here;
   * {@link state} and {@link prevState} are still readable. */
  afterDelete(_: Context): void {}

  private validatePath(path: readonly string[]): void {
    if (path.length === 0)
      throw new UnexpectedError(
        `[aether.leaf.setState] ${this.toString()} received an empty path`,
      );
    const key = path[path.length - 1];
    if (path.length > 1)
      throw new UnexpectedError(
        `[aether.leaf.setState] - ${this.toString()} received a subPath ${path.join(
          ".",
        )} but is a leaf`,
      );
    if (key !== this.key)
      throw new UnexpectedError(
        `[aether.leaf.setState] - ${this.toString()} received a key ${key} but expected ${this.key}`,
      );
  }

  protected handleInvokeError(
    { method, key, args }: InvokeMethodParams,
    error: unknown,
  ) {
    const expectsResponse = key != null;
    if (!expectsResponse)
      return console.error(
        `Error in fire and forget method ${method} on ${this.toString()}`,
        error,
      );

    const err =
      error instanceof Error ? error : new Error(String(error), { cause: error });
    const wrapped = new Error(
      `Failed to execute ${method}(${key}) with args ${JSON.stringify(args)} on ${this.toString()}: ${err.message}`,
      { cause: err },
    );
    // Preserve the original error's name and stack across the worker boundary so the
    // main-thread receiver sees the inner type and frames rather than the wrapper's.
    wrapped.name = err.name;
    wrapped.stack = err.stack;
    this.sender.send({
      variant: "invoke_response",
      key,
      result: undefined,
      error: errors.encode(wrapped),
    });
  }

  _invokeMethod(params: InvokeMethodParams): void {
    if (this.deleted) return;
    const { method, key, args } = params;
    const expectsResponse = key != null;
    const methodFn = this._methodImplementations?.[method];
    if (methodFn == null)
      return this.handleInvokeError(
        params,
        new Error(`Method ${method} not found on ${this.toString()}`),
      );
    // Dynamic dispatch - TypeScript can't track string → method lookup.
    try {
      const res = methodFn(...args);
      if (res instanceof Promise)
        res
          .then((r: unknown) => {
            if (expectsResponse)
              this.sender.send({ variant: "invoke_response", key, result: r });
          })
          .catch((e: unknown) => this.handleInvokeError(params, e));
    } catch (e) {
      this.handleInvokeError(params, e);
    }
  }
}

/**
 * Base class for aether components that own children. Extends {@link Leaf} with a typed
 * children registry, descendant context propagation, and routing for
 * updates/deletes/invokes addressed at descendants. */
export abstract class Composite<
  StateSchema extends z.ZodType<state.State>,
  InternalState extends {} = {},
  ChildComponents extends Component = Component,
  M extends MethodsSchema = EmptyMethodsSchema,
>
  extends Leaf<StateSchema, InternalState, M>
  implements Component
{
  private readonly _children: Map<string, ChildComponents> = new Map();

  /** Snapshot of the children, in insertion order. */
  get children(): readonly ChildComponents[] {
    return Array.from(this._children.values());
  }

  /** Returns the child at `key`, or `null` if none. The `T` cast is unchecked — callers
   * must know which subtype they are looking up. */
  getChild<T extends ChildComponents = ChildComponents>(key: string): T | null {
    return (this._children.get(key) ?? null) as T | null;
  }

  /** Returns the children whose `type` matches one of `types`. The `T` cast is
   * unchecked — callers must align the type filter with the asserted return type. */
  childrenOfType<T extends ChildComponents = ChildComponents>(
    ...types: Array<T["type"]>
  ): readonly T[] {
    return this.children.filter((c) =>
      types.includes(c.type),
    ) as unknown as readonly T[];
  }

  _updateState(params: UpdateStateParams): void {
    const { path, type, create } = params;
    if (this.deleted) return;
    const subPath = this.parsePath(path);

    const isSelfUpdate = subPath.length === 0;
    if (isSelfUpdate) {
      this.childCtxChangedKeys.clear();
      super._updateState(params);
      if (this.childCtxChangedKeys.size == 0) return;
      this.updateChildContexts();
      return;
    }

    const childKey = subPath[0];
    const child = this.getChild(childKey);
    if (child != null) return child._updateState({ ...params, path: subPath });
    if (subPath.length > 1) {
      const childPath = path.slice(0, path.indexOf(childKey) + 1).join(".");
      const fullPath = path.join(".");
      throw new UnexpectedError(
        `Child of ${this.toString()} at path ${childPath} does not exist,
        but an extended path ${fullPath} was provided. This means that the aether
        tree is attempting to create a new child  of type ${type} (or nested children)
        on a child that does not exist.

        Children present: ${this.children.map((c) => `${c.type}:${c.key}`).join(".")}

        `,
      );
    }
    const newChild = create(this.childCtx());
    newChild._updateState({ ...params, path: subPath });
    this._children.set(childKey, newChild as ChildComponents);
  }

  _updateContext(values: ContextMap): void {
    super._updateContext(values);
    this.updateChildContexts();
  }

  private childCtx(): ContextMap {
    return {
      get: (key) => this.childCtxValues.get(key) ?? this.parentCtxValues.get(key),
      forEach: (callback) => {
        this.childCtxValues.forEach((value, key) =>
          callback(value, key, this.childCtxValues),
        );
        this.parentCtxValues.forEach((value, key) => {
          if (this.childCtxValues.has(key)) return;
          callback(value, key, this.parentCtxValues);
        });
      },
    };
  }

  private updateChildContexts(): void {
    const childCtx = this.childCtx();
    this.children.forEach((c) => c._updateContext(childCtx));
  }

  _delete(path: readonly string[]): void {
    const subPath = this.parsePath(path);
    if (subPath.length === 0) {
      for (const c of this.children) c._delete([c.key]);
      this._children.clear();
      super._delete([this.key]);
    }
    const child = this.getChild(subPath[0]);
    if (child == null) return;
    if (subPath.length > 1) return child._delete(subPath);
    this._children.delete(child.key);
    child._delete(subPath);
  }

  /** Walks down the tree by `path` (relative to this composite — i.e. excluding this
   * component's own key) and returns the component at the leaf, or `null` if any step
   * is missing. */
  findChildAtPath(path: string[]): Component | null {
    if (path.length === 0) return null;
    const [key, ...rest] = path;
    const child = this.getChild(key);
    if (child == null) return null;
    if (rest.length === 0) return child;
    if ("findChildAtPath" in child && typeof child.findChildAtPath === "function")
      return child.findChildAtPath(rest);

    return null;
  }

  private parsePath(path: readonly string[], type?: string): readonly string[] {
    const [key, ...subPath] = path;
    if (key == null)
      throw new Error(
        `Composite ${this.toString()} received an empty path${
          type != null ? ` for ${type}` : ""
        }`,
      );
    if (key !== this.key)
      throw new UnexpectedError(
        `[Composite.getRequiredKey] - ${this.toString()} received a key ${key} but expected ${this.key}`,
      );
    return subPath;
  }
}

/** Map from component `type` to constructor, consulted by {@link Root} to instantiate
 * new components on demand. */
export type ComponentRegistry = Record<string, ComponentConstructor>;

const aetherRootState = z.object({});

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

const shouldNotCallCreate = () => {
  throw new Error("should not call create");
};

/** Top-level node of the worker-side Aether tree. Owns the {@link comms} channel and
 * the component {@link registry}; routes messages from the main thread to the
 * appropriate descendant. Construct via {@link Root.render}. */
export class Root extends Composite<typeof aetherRootState> {
  private static readonly TYPE = "root";
  private static readonly KEY = "root";

  private readonly comms: WorkerComms;
  private readonly registry: ComponentRegistry;

  schema = aetherRootState;

  constructor({
    worker = wrapWorkerScope(),
    instrumentation = alamos.Instrumentation.NOOP,
    registry,
  }: RootProps) {
    super({
      path: [Root.KEY],
      type: Root.TYPE,
      sender: worker,
      instrumentation,
      parentCtxValues: null,
    });
    this.comms = worker;
    this.registry = registry;
  }

  /** Constructs a new aether tree and starts handling messages on `comms`. Any
   * synchronous error thrown while processing a message is caught and surfaced on the
   * main thread as a {@link WorkerNotifyErrorRequest}. */
  static render(props: RootProps): Root {
    const root = new Root(props);
    root._updateState({
      path: [Root.KEY],
      type: "",
      state: {},
      create: shouldNotCallCreate,
    });
    // Messages are handled sequentially: concurrent component updates lead to races, so
    // the tree is implicitly serialized via the message queue.
    root.comms.handle((msg) => {
      try {
        root.handle(msg);
      } catch (e) {
        root.comms.send({ variant: "error", error: errors.encode(e) });
      }
    });
    return root;
  }

  private handle(msg: MainMessage): void {
    const { variant } = msg;

    if (variant === "invoke_request") {
      this.invokeAtPath(msg);
      return;
    }

    if (variant === "delete") {
      this._delete(msg.path);
      return;
    }

    const { path, type, state } = msg;
    this._updateState({
      path,
      type,
      state,
      create: (parentCtxValues) => this.create({ path, type, parentCtxValues }),
    });
  }

  private invokeAtPath(params: InvokeMethodParams): void {
    const { path } = params;
    const [rootKey, ...childPath] = path;
    if (rootKey !== Root.KEY) {
      this.handleInvokeError(
        params,
        new Error(`Invalid path: expected root key '${Root.KEY}', got '${rootKey}'`),
      );
      return;
    }

    if (childPath.length === 0) {
      this._invokeMethod(params);
      return;
    }

    const component = this.findChildAtPath(childPath);
    if (component == null) {
      this.handleInvokeError(
        params,
        new NotFoundError(`Component at path ${path.join(".")} not found`),
      );
      return;
    }

    component._invokeMethod(params);
  }

  private create({
    path,
    type,
    parentCtxValues,
  }: Omit<ComponentConstructorProps, "sender" | "instrumentation">): Component {
    const Constructor = this.registry[type];
    if (Constructor == null)
      throw new UnexpectedError(`[Root.create] - ${type} not found in registry`);
    return new Constructor({
      path,
      type,
      sender: this.comms,
      instrumentation: this.instrumentation,
      parentCtxValues,
    });
  }
}

/** Convenience binding of {@link Root.render}: constructs an aether tree and binds it
 * to the provided `comms`. */
export const render = Root.render.bind(Root);
