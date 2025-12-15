// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { alamos } from "@synnaxlabs/alamos";
import { NotFoundError, UnexpectedError, ValidationError } from "@synnaxlabs/client";
import {
  deep,
  type errors,
  type record,
  type Sender,
  type SenderHandler,
  shallow,
  zod,
} from "@synnaxlabs/x";
import { z } from "zod";

import {
  type AetherMessage,
  type MainInvokeRequest,
  type MainMessage,
  type MainUpdateRequest,
} from "@/aether/message";
import { state } from "@/state";
import { prettyParse } from "@/util/zod";

const newTreeError = (e: unknown, pathOrMessage?: string): Error => {
  if (e instanceof Error) {
    e.message = `[${pathOrMessage}] - ${e.message}`;
    return e;
  }
  return new Error(pathOrMessage ?? "unknown error", { cause: e });
};

/**
 * A selected subset of the Map interface used for efficiently propagating context
 * updates through the tree.
 */
export interface ContextMap extends Pick<Map<string, unknown>, "get" | "forEach"> {}

/**
 * An internal function alias that creates a new component with the specified
 * initial parent context
 */
interface CreateComponent {
  (initialParentCtxValues: ContextMap): Component;
}

export interface UpdateStateParams
  extends Pick<MainUpdateRequest, "path" | "type" | "state"> {
  create: CreateComponent;
}

export interface InvokeMethodParams extends Omit<MainInvokeRequest, "variant"> {}

/**
 * A component in the Aether tree. Each component instance has a unique key identifying
 * it within the tree, and also has a type identifying it's class. Components
 * implementing different functionality should have different types, and these types
 * typically correlate to the corresponding name of the component in the react tree.
 */
export interface Component {
  /** The type of component. */
  type: string;
  /** A unique key identifying the component within the tree. */
  key: string;
  toString(): string;
  /**
   * Propagates a state update to the component at the given path. This is an internal
   * method, should not be called by subclasses outside of this file.
   *
   * @param path - The path of the component to update. The length of the path indicates
   * the depth of the component in the aether tree. This method should propagate the
   * state if the path has a length greater than 1.
   * @param type - the type of component being updated (or created).
   * @param state - The state to update the component with.
   * @param create - A function that creates a new component of the appropriate type if
   * it doesn't exist in the tree.
   */
  _updateState: (args: UpdateStateParams) => void;
  /**
   * Propagates a context update to the children and all of its descendants.
   */
  _updateContext: (values: ContextMap) => void;
  /**
   * Propagates a delete to the internal tree of the component, calling the handleDelete
   * component on the component itself and any children as necessary. It is up to
   * the parent component to remove the child from its internal tree.
   *
   * @param path - The path of the component to delete.
   */
  _delete: (path: string[]) => void;
  /**
   * Invokes a method on this component. This is called by the Root when
   * a MainInvokeRequest is received.
   *
   * @param key - The correlation ID for matching the response.
   * @param method - The name of the method to invoke.
   * @param args - The arguments to pass to the method (spread when calling handler).
   * @param expectsResponse - Whether to send a response back to the caller.
   */
  _invokeMethod: (params: InvokeMethodParams) => void;
}

/** Constructor props that all aether components must accept. */
export interface ComponentConstructorProps {
  /** The key of the component. */
  key: string;
  /** The type of the component. */
  type: string;
  /** A sender used to propagate messages back to the main thread. */
  sender: Sender<AetherMessage>;
  /** Instrumentation used for logging and tracing. */
  instrumentation: alamos.Instrumentation;
  /**
   * Initial parent context values for the component. These should be intentionally
   * set to null if no values exists.
   */
  parentCtxValues: ContextMap | null;
}

/** A constructor type for an AetherComponent. */
export interface ComponentConstructor {
  new (props: ComponentConstructorProps): Component;
}

/**
 * The Context interface allows for parent components to pass context values to
 * arbitrarily nested children.
 */
export interface Context {
  /**
   * Gets a context value, interpreting it as the given type, and returning it. This
   * method does no internal type checking - it is up to the caller to ensure that the
   * value exists for the given key and is of the correct type.
   * @throws a NotFoundError if the value does not exist for the given key.
   * @returns the context value interpreted as the given type.
   */
  get<P>(key: string): P;
  /**
   * Gets a context value, interpreting it as the given type, and returning it. This
   * method does no internal type checking - it is up to the caller to ensure that the
   * value is of the correct type.
   * @returns the context value, or null if no value exists for the given key.
   */
  getOptional<P>(key: string): P | null;
  /**
   * Checks if a parent context value exists for the given key. Note that this will
   * return false for all context values set by the component itself.
   * @returns true if a value exists for the given key, false otherwise.
   */
  has(key: string): boolean;
  /**
   * Sets a context value for the given key, making it available to all children of the
   * component, but NOT to the component itself. This detail is crucial, as it means
   * the component can override the value of an existing parent key and propagate it
   * to its children, while still being able to access the original parent value.
   *
   * @param key - The key of the context value to set.
   * @param value - The value to set for the given key.
   * @param trigger - If true, the component will be notified of the change.
   */
  set(key: string, value: unknown, trigger?: boolean): void;
  /**
   * Checks if the component has previously set a context value for the given key. This
   * is an alternative to {@link has} that checks whether the component has set the value
   * itself, as opposed to whether it has been set by a parent.
   * @returns true if the component has set the value itself, false otherwise.
   */
  wasSetPreviously(key: string): boolean;
}

/**
 * A schema defining multiple invokable methods using z.function().
 * No constraint - the actual schema type flows through for proper inference.
 */

export type MethodsSchema = Record<string, z.ZodFunction>;

/** Empty methods schema for components that don't use invoke. */
export type EmptyMethodsSchema = Record<string, never>;

/**
 * Unwraps Promise<T> to T, leaves non-Promise types unchanged.
 * Used to prevent double-wrapping of Promise types.
 */
type Awaited<T> = T extends Promise<infer U> ? U : T;

/**
 * Type helper to extract handler function signatures from a methods schema.
 * Uses z.infer to extract the function type, then allows async returns.
 * If the schema output is already Promise<T>, handlers return Promise<T> (not Promise<Promise<T>>).
 */
export type HandlersFromSchema<T> = {
  [K in keyof T]: T[K] extends z.ZodType<infer F>
    ? F extends (...args: infer A) => infer R
      ? R extends Promise<unknown>
        ? (...args: A) => R // Schema already specifies async, don't allow double-wrap
        : (...args: A) => R | Promise<R> // Schema is sync, allow async implementation
      : never
    : never;
};

export const isFireAndForget = <F extends z.ZodFunction>(schema: F): boolean => {
  const outputType = zod.functionOutput(schema);
  return (
    outputType instanceof z.ZodVoid ||
    outputType instanceof z.ZodNever ||
    outputType instanceof z.ZodUnknown
  );
};

/**
 * Type helper to extract caller function signatures from a methods schema.
 * Void returns become fire-and-forget (returns void), non-void returns use Promise.
 * If the schema output is already Promise<T>, callers return Promise<T> (not Promise<Promise<T>>).
 */
export type CallersFromSchema<T> = {
  [K in keyof T]: T[K] extends z.ZodType<infer F>
    ? F extends (...args: infer A) => infer R
      ? R extends void
        ? (...args: A) => void
        : (...args: A) => Promise<Awaited<R>> // Unwrap if already Promise to avoid double-wrap
      : never
    : never;
};

/**
 * Implements an AetherComponent that does not have any children, and servers as the base
 * class for the AetherComposite type. The corresponding react component should NOT have
 * any children that use Aether functionality; for those cases, use AetherComposite instead.
 *
 * For invokable methods:
 * 1. Define a methods schema using `z.function()`
 * 2. Add `implements HandlersFromSchema<typeof schema>` to get type checking
 * 3. Set `methods = schema` on the class
 * 4. Implement methods with matching names
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
 *
 *   // TypeScript enforces these match the schema signatures
 *   onMouseDown(): void { ... }
 *   onMouseUp(): number { return 42; }
 * }
 * ```
 */
export abstract class Leaf<
  StateSchema extends z.ZodType<state.State>,
  InternalState extends {} = {},
  Methods extends MethodsSchema = EmptyMethodsSchema,
> implements Component
{
  readonly type: string;
  readonly key: string;

  protected readonly sender: Sender<AetherMessage>;

  private readonly _internalState: InternalState;
  private _state: z.infer<StateSchema> | undefined;
  private _prevState: z.infer<StateSchema> | undefined;
  private _deleted: boolean = false;
  protected readonly parentCtxValues: Map<string, any>;
  protected readonly childCtxValues: Map<string, any>;
  protected readonly childCtxChangedKeys: Set<string>;
  readonly instrumentation: alamos.Instrumentation;

  schema: StateSchema | undefined = undefined;

  /**
   * Methods schema for invoke. Define this to enable invokable methods.
   * Method names in the schema must match method names on the class.
   */
  methods: Methods | undefined = undefined as Methods | undefined;
  private _methodImplementations: Record<
    string,
    (...args: unknown[]) => Promise<unknown>
  > | null = null;

  constructor({
    key,
    type,
    sender,
    instrumentation,
    parentCtxValues,
  }: ComponentConstructorProps) {
    this.type = type;
    this.key = key;
    this.sender = sender;
    this._internalState = {} as InternalState;
    this.instrumentation = instrumentation.child(this.toString());
    this.parentCtxValues = new Map();
    parentCtxValues?.forEach((value, key) => this.parentCtxValues.set(key, value));
    this.childCtxValues = new Map();
    this.childCtxChangedKeys = new Set();
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

  /**
   * Sets the state on the component, communicating the change to the corresponding
   * React component on the main thread.
   *
   * @param next - The new state to set on the component. This can be the state object
   * or a pure function that takes in the previous state and returns the next state.
   */
  setState(next: state.SetArg<z.infer<StateSchema>>): void {
    const nextState = state.executeSetter(next, this.state);
    this._prevState = shallow.copy(this._state);
    this._state = prettyParse(this._schema, nextState, `${this.toString()}`);
    this.sender.send({ variant: "update", key: this.key, state: this._state });
  }

  /** @returns the current state of the component. */
  get state(): z.infer<StateSchema> {
    if (this._state == null)
      throw new UnexpectedError(
        `[AetherLeaf] - state not defined in ${this.toString()}`,
      );
    return this._state;
  }

  get internal(): InternalState {
    return this._internalState;
  }

  /** @returns the previous state of the component. */
  get prevState(): z.infer<StateSchema> {
    if (this._prevState === undefined) throw new Error("prevState not defined");
    return this._prevState;
  }

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

  /**
   * @implements AetherComponent, and should NOT be called by a subclass other than
   * AetherComposite.
   */
  _updateState({ path, state }: UpdateStateParams): void {
    if (this.deleted) return;
    try {
      this.initializeMethods();
      const endSpan = this.instrumentation.T.debug(`${this.toString()}:updateState`);
      this.validatePath(path);
      const state_ = prettyParse(this._schema, state, `${this.toString()}`);
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

  /**
   * @implements AetherComponent, and should NOT be called by a subclass other than
   * AetherComposite.
   */
  _delete(path: string[]): void {
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

  /**
   * afterUpdate is optionally defined by a subclass, allowing the component to
   * perform some action after the component is updated. At this point, the current
   * state, previous state, derived state, and current context are all available to
   * the component.
   */
  afterUpdate(_: Context): void {}

  /**
   * Runs after the component has been spliced out of the tree. This is useful for
   * running cleanup code, such as unsubscribing from an event emitter. At this point,
   * the current state, previous state, derived state, and current context are all
   * available to the component, and this.deleted is true.
   */
  afterDelete(_: Context): void {}

  private validatePath(path: string[]): void {
    if (path.length === 0)
      throw new UnexpectedError(
        `[Leaf.setState] - ${this.toString()} received an empty path`,
      );
    const key = path[path.length - 1];
    if (path.length > 1)
      throw new UnexpectedError(
        `[Leaf.setState] - ${this.toString()} received a subPath ${path.join(
          ".",
        )} but is a leaf`,
      );
    if (key !== this.key)
      throw new UnexpectedError(
        `[Leaf.setState] - ${this.toString()} received a key ${key} but expected ${this.key}`,
      );
  }

  protected handleInvokeError(
    { expectsResponse, method, key, args }: InvokeMethodParams,
    error: unknown,
  ) {
    if (!expectsResponse)
      return console.error(
        `Error in fire and forget method ${method} on ${this.toString()}`,
        error,
      );

    const err = error instanceof Error ? error : new Error(String(error));
    this.sender.send({
      variant: "invoke_response",
      key,
      result: undefined,
      error: {
        name: err.name,
        message: `Failed to execute ${method}(${key}) with args ${JSON.stringify(args)} on ${this.toString()}: ${err.message}`,
        stack: err.stack ?? "",
      },
    });
  }

  _invokeMethod(params: InvokeMethodParams): void {
    if (this.deleted) return;
    const { method, key, args, expectsResponse } = params;
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
          .catch((e) => this.handleInvokeError(params, e));
    } catch (e) {
      this.handleInvokeError(params, e);
    }
  }
}

/**
 * AetherComposite is an implementation of AetherComponent that allows it to maintain
 * child components. It is the base class for all composite components, and should not
 * be used directly.
 */
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

  /** @returns a readonly array of the children of the component. */
  get children(): readonly ChildComponents[] {
    return Array.from(this._children.values());
  }

  /**
   * Finds a child component with the given key.
   *
   * @param key - the key of the child component to find.
   * @returns the child component, or null if no child component with the given key
   */
  getChild<T extends ChildComponents = ChildComponents>(key: string): T | null {
    return (this._children.get(key) ?? null) as T | null;
  }

  /**
   * Finds all children of the component with the given type
   *
   * @param types - the type of the children to find
   * @returns an array of all children of the component with the given type
   */
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

  _delete(path: string[]): void {
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

  /**
   * Find a child component at the given path for method invocation.
   * @param path - The path to the child component (excluding this component's key).
   * @returns The component at the path, or null if not found.
   */
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

  private parsePath(path: string[], type?: string): string[] {
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

export type ComponentRegistry = Record<string, ComponentConstructor>;

const aetherRootState = z.object({});

/**
 * The props for creating the root of the Aether tree.
 */
export interface RootProps {
  /** A communication mechanism for sending messages to the main thread. */
  comms: SenderHandler<AetherMessage, MainMessage>;
  /** A registry of available components that can be used to mirror those on the main thread. */
  registry: ComponentRegistry;
  /** Instrumentation used for logging and tracing. */
  instrumentation?: alamos.Instrumentation;
}

const shouldNotCallCreate = () => {
  throw new Error("should not call create");
};

export class Root extends Composite<typeof aetherRootState> {
  /** Key of the root component. */
  private static readonly TYPE = "root";
  /** Type of the root component. */
  private static readonly KEY = "root";

  /** A communication mechanism for sending messages to the main thread. */
  private readonly comms: SenderHandler<AetherMessage, MainMessage>;
  /** A registry used for creating new components in the tree. */
  private readonly registry: ComponentRegistry;

  schema = aetherRootState;

  constructor({
    comms,
    instrumentation = alamos.Instrumentation.NOOP,
    registry,
  }: RootProps) {
    super({
      key: Root.KEY,
      type: Root.TYPE,
      sender: comms,
      instrumentation,
      parentCtxValues: null,
    });
    this.comms = comms;
    this.registry = registry;
  }

  /**
   * Creates a new aether tree with the provided props, and starts listing for state
   * updates on the provided comms.
   */
  static render(props: RootProps): Root {
    const root = new Root(props);
    root._updateState({
      path: [Root.KEY],
      type: "",
      state: {},
      create: shouldNotCallCreate,
    });
    /**
     * Unfortunately we get a bunch of nasty race conditions whenever component updates
     * are not serialized, so we need to lock the entire component tree when making
     * updates.
     */
    root.comms.handle((msg) => {
      try {
        root.handle(msg);
      } catch (e) {
        const errorObj: errors.NativePayload = {
          name: "unknown",
          message: JSON.stringify(e),
          stack: "unknown",
        };
        if (e instanceof Error) {
          errorObj.name = e.name;
          errorObj.message = e.message;
          errorObj.stack = e.stack;
        }
        root.comms.send({ variant: "error", error: errorObj });
      }
    });
    return root;
  }

  /**
   * Handles messages from the main thread and applies them as updates in the
   * aether tree.
   */
  private handle(msg: MainMessage): void {
    const { variant } = msg;

    if (variant === "invoke_request") {
      this.invokeAtPath(msg);
      return;
    }

    const { path, type } = msg;
    if (variant === "delete") {
      this._delete(path);
      return;
    }

    const { state } = msg;
    this._updateState({
      path,
      type,
      state,
      create: (parentCtxValues) => {
        const key = path[path.length - 1];
        return this.create({ key, type, parentCtxValues });
      },
    });
  }

  private invokeAtPath(params: InvokeMethodParams): void {
    const { path, expectsResponse } = params;
    const [rootKey, ...childPath] = path;
    if (rootKey !== Root.KEY) {
      if (expectsResponse)
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
      if (expectsResponse)
        this.handleInvokeError(
          params,
          new NotFoundError(`Component at path ${path.join(".")} not found`),
        );
      return;
    }

    component._invokeMethod(params);
  }

  /** Creates a new component from the registry */
  private create({
    key,
    type,
    parentCtxValues,
  }: Omit<ComponentConstructorProps, "sender" | "instrumentation">): Component {
    const Constructor = this.registry[type];
    if (Constructor == null)
      throw new UnexpectedError(`[Root.create] - ${type} not found in registry`);
    return new Constructor({
      key,
      type,
      sender: this.comms,
      instrumentation: this.instrumentation,
      parentCtxValues,
    });
  }
}

/**
 * Creates a new aether tree with the provided props.
 *
 * @param props - The props for the root component.
 * @param props.comms - A communication mechanism for sending messages to the main thread.
 * Typically this is implemented by a web worker.
 * @param props.registry - A registry of available components that can be used to mirror
 * those on the main thread.
 */
export const render = Root.render.bind(Root);
