// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { alamos } from "@synnaxlabs/alamos";
import { UnexpectedError, ValidationError } from "@synnaxlabs/client";
import { type Sender, type SenderHandler, type UnknownRecord } from "@synnaxlabs/x";
import { deep } from "@synnaxlabs/x/deep";
import { Mutex } from "async-mutex";
import { z } from "zod";

import { type MainMessage, type WorkerMessage } from "@/aether/message";
import { state } from "@/state";
import { prettyParse } from "@/util/zod";

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
  /**
   * Propagates a state update to the component at the given path. This is an internal
   * method, should not be called by subclasses outside of this file.
   *
   * @param path - The path of the component to update. The length of the path indicates
   * the depth of the component in the aether tree. This method should propagate the
   * state if the path has a length greater than 1.
   * @param state - The state to update the component with.
   * @param create - A function that creates a new component of the appropriate type if
   * it doesn't exist in the tree.
   */
  _updateState: (
    path: string[],
    state: UnknownRecord,
    create: CreateComponent,
  ) => Promise<void>;
  /**
   * Propagates a context update to the children and all of its descendants.
   */
  _updateContext: (values: ContextMap) => Promise<void>;
  /**
   * Propagates a delete to the internal tree of the component, calling the handleDelete
   * component on the component itself and any children as necessary. It is up to
   * the parent component to remove the child from its internal tree.
   *
   * @param path - The path of the component to delete.
   */
  _delete: (path: string[]) => Promise<void>;
}

/** Constructor props that all aether components must accept. */
interface ComponentConstructorProps {
  /** The key of the component. */
  key: string;
  /** The type of the component. */
  type: string;
  /** A sender used to propagate messages back to the main thread. */
  sender: Sender<WorkerMessage>;
  /** Instrumentation used for logging and tracing. */
  instrumentation: alamos.Instrumentation;
  /** Initial parent context values for the component. These should be intentionally
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
  set(key: string, value: any, trigger?: boolean): void;
  /**
   * Checks if the component has previously set a context value for the given key. This
   * is an alternative to {@link has} that checks whether the component has set the value
   * itself, as opposed to whether it has been set by a parent.
   * @returns true if the component has set the value itself, false otherwise.
   */
  setPreviously(key: string): boolean;
}

/**
 * Implements an AetherComponent that does not have any children, and servers as the base
 * class for the AetherComposite type. The corresponding react component should NOT have
 * any children that use Aether functionality; for those cases, use AetherComposite instead.
 */
export abstract class Leaf<
  StateSchema extends z.ZodTypeAny,
  InternalState extends {} = {},
> implements Component
{
  readonly type: string;
  readonly key: string;

  private readonly sender: Sender<WorkerMessage>;

  private readonly _internalState: InternalState;
  private _state: z.output<StateSchema> | undefined;
  private _prevState: z.output<StateSchema> | undefined;
  private _deleted: boolean = false;
  protected readonly parentCtxValues: Map<string, any>;
  protected readonly childCtxValues: Map<string, any>;
  protected readonly childCtxChangedKeys: Set<string>;
  readonly instrumentation: alamos.Instrumentation;

  schema: StateSchema | undefined = undefined;

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
    this.instrumentation = instrumentation.child(`${this.type}(${this.key})`);
    this.parentCtxValues = new Map();
    parentCtxValues?.forEach((value, key) => this.parentCtxValues.set(key, value));
    this.childCtxValues = new Map();
    this.childCtxChangedKeys = new Set();
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
  setState(
    next: state.SetArg<
      z.input<StateSchema> | z.output<StateSchema>,
      z.output<StateSchema>
    >,
  ): void {
    const nextState: z.input<StateSchema> = state.executeSetter(next, this._state);
    this._prevState = { ...this._state };
    this._state = prettyParse(this._schema, nextState, `${this.type}:${this.key}`);
    this.sender.send({ key: this.key, state: nextState });
  }

  /** @returns the current state of the component. */
  get state(): z.output<StateSchema> {
    if (this._state == null)
      throw new UnexpectedError(
        `[AetherLeaf] - state not defined in ${this.type}:${this.key}`,
      );
    return this._state;
  }

  get internal(): InternalState {
    return this._internalState;
  }

  /** @returns the previous state of the component. */
  get prevState(): z.output<StateSchema> {
    return this._prevState;
  }

  get deleted(): boolean {
    return this._deleted;
  }

  private get ctx(): Context {
    return {
      get: (key: string) => this.parentCtxValues.get(key),
      getOptional: (key: string) => this.parentCtxValues.get(key),
      has: (key: string) => this.parentCtxValues.has(key),
      setPreviously: (key: string) => this.childCtxValues.has(key),
      set: (key: string, value: any, trigger: boolean = true) => {
        this.childCtxValues.set(key, value);
        if (trigger) this.childCtxChangedKeys.add(key);
      },
    };
  }

  /**
   * @implements AetherComponent, and should NOT be called by a subclass other than
   * AetherComposite.
   */
  async _updateState(
    path: string[],
    state: UnknownRecord,
    _: CreateComponent,
  ): Promise<void> {
    if (this.deleted) return;
    this.validatePath(path);
    const state_ = prettyParse(this._schema, state, `${this.type}:${this.key}`);
    if (this._state != null)
      this.instrumentation.L.debug("updating state", () => ({
        diff: deep.difference(this.state, state_),
      }));
    else this.instrumentation.L.debug("setting initial state", { state });
    this._prevState = this._state ?? state_;
    this._state = state_;
    await this.afterUpdate(this.ctx);
  }

  async _updateContext(values: ContextMap): Promise<void> {
    values.forEach((value, key) => this.parentCtxValues.set(key, value));
    await this.afterUpdate(this.ctx);
  }

  /**
   * @implements AetherComponent, and should NOT be called by a subclass other than
   * AetherComposite.
   */
  async _delete(path: string[]): Promise<void> {
    this.validatePath(path);
    this._deleted = true;
    await this.afterDelete(this.ctx);
  }

  /**
   * afterUpdate is optionally defined by a subclass, allowing the component to
   * perform some action after the component is updated. At this point, the current
   * state, previous state, derived state, and current context are all available to
   * the component.
   */
  async afterUpdate(_: Context): Promise<void> {}

  /**
   * Runs after the component has been spliced out of the tree. This is useful for
   * running cleanup code, such as unsubscribing from an event emitter. At this point,
   * the current state, previous state, derived state, and current context are all
   * available to the component, and this.deleted is true.
   */
  async afterDelete(_: Context): Promise<void> {}

  private validatePath(path: string[]): void {
    if (path.length === 0)
      throw new UnexpectedError(
        `[Leaf.setState] - ${this.type}:${this.key} received an empty path`,
      );
    const key = path[path.length - 1];
    if (path.length > 1)
      throw new UnexpectedError(
        `[Leaf.setState] - ${this.type}:${this.key} received a subPath ${path.join(
          ".",
        )} but is a leaf`,
      );
    if (key !== this.key)
      throw new UnexpectedError(
        `[Leaf.setState] - ${this.type}:${this.key} received a key ${key} but expected ${this.key}`,
      );
  }
}

/**
 * AetherComposite is an implementation of AetherComponent that allows it to maintain
 * child components. It is the base class for all composite components, and should not
 * be used directly.
 */
export abstract class Composite<
    StateSchema extends z.ZodTypeAny,
    InternalState extends {} = {},
    ChildComponents extends Component = Component,
  >
  extends Leaf<StateSchema, InternalState>
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

  async _updateState(
    path: string[],
    state: UnknownRecord,
    create: CreateComponent,
  ): Promise<void> {
    if (this.deleted) return;
    const subPath = this.parsePath(path);

    const isSelfUpdate = subPath.length === 0;
    if (isSelfUpdate) {
      this.childCtxChangedKeys.clear();
      await super._updateState(path, state, create);
      if (this.childCtxChangedKeys.size == 0) return;
      await this.updateChildContexts();
      return;
    }

    const childKey = subPath[0];
    const child = this.getChild(childKey);
    if (child != null) return child._updateState(subPath, state, create);
    if (subPath.length > 1)
      throw new UnexpectedError(
        `[Composite.update] - ${this.type}:${this.key} received a subPath ${subPath.join(
          ".",
        )} but found now child with key ${childKey}`,
      );
    const newChild = create(this.childCtx());
    await newChild._updateState(subPath, state, create);
    this._children.set(childKey, newChild as ChildComponents);
  }

  async _updateContext(values: ContextMap): Promise<void> {
    await super._updateContext(values);
    await this.updateChildContexts();
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

  private async updateChildContexts(): Promise<void> {
    for (const c of this.children) await c._updateContext(this.childCtx());
  }

  async _delete(path: string[]): Promise<void> {
    const subPath = this.parsePath(path);
    if (subPath.length === 0) {
      for (const c of this.children) await c._delete([c.key]);
      this._children.clear();
      await super._delete([this.key]);
    }
    const child = this.getChild(subPath[0]);
    if (child == null) return;
    if (subPath.length > 1) return await child._delete(subPath);
    this._children.delete(child.key);
    await child._delete(subPath);
  }

  private parsePath(path: string[], type?: string): string[] {
    const [key, ...subPath] = path;
    if (key == null)
      throw new Error(
        `Composite ${this.type}:${this.key} received an empty path${
          type != null ? ` for ${type}` : ""
        }`,
      );
    if (key !== this.key)
      throw new UnexpectedError(
        `[Composite.getRequiredKey] - ${this.type}:${this.key} received a key ${key} but expected ${this.key}`,
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
  comms: SenderHandler<WorkerMessage, MainMessage>;
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
  private readonly comms: SenderHandler<WorkerMessage, MainMessage>;
  /** A registry used for creating new components in the tree. */
  private readonly registry: ComponentRegistry;
  /** Mutex used for serializing state updates to the tree. */
  private readonly mu = new Mutex();

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
  static async render(props: RootProps): Promise<Root> {
    const root = new Root(props);
    await root._updateState([Root.KEY], {}, shouldNotCallCreate);
    /**
     * Unfortunately we get a bunch of nasty race conditions whenever component updates
     * are not serialized, so we need to lock the entire component tree when making
     * updates.
     */
    root.comms.handle((msg) => {
      void root.mu.runExclusive(async () => {
        await root.handle(msg);
      });
    });
    return root;
  }

  /**
   * Handles messages from the worker thread and applies them as updates in the
   * aether tree.
   */
  private async handle(msg: MainMessage): Promise<void> {
    const { path, variant, type } = msg;
    if (variant === "delete") await this._delete(path);
    else
      await this._updateState(path, msg.state, (parentCtxValues) => {
        const key = path[path.length - 1];
        return this.create({ key, type, parentCtxValues });
      });
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
