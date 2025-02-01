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

interface ContextMap extends Pick<Map<string, any>, "get" | "forEach"> {}

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
   * Propagates an update to the internal tree of the component, updating the component
   * itself and any children as necessary.
   *
   * @param update - The update to propagate.
   */
  _updateState: (
    path: string[],
    state: UnknownRecord,
    create: () => Component,
  ) => Promise<void>;

  /**
   *
   */
  _updateContext: (values: ContextMap, triggerAfterUpdate?: boolean) => Promise<void>;
  /**
   * Propagates a delete to the internal tree of the component, calling the handleDelete
   * component on the component itself and any children as necessary. It is up to
   * the parent component to remove the child from its internal tree.
   *
   * @param path - The path of the component to delete.
   */
  _delete: (path: string[]) => Promise<void>;
}

/** A constructor type for an AetherComponent. */
export type ComponentConstructor = new (
  key: string,
  type: string,
  sender: Sender<WorkerMessage>,
  instrumentation: alamos.Instrumentation,
) => Component;

export interface Context {
  get<P>(key: string): P;
  getOptional<P>(key: string): P | null;
  has(key: string): boolean;
  setPreviously(key: string): boolean;
  set(key: string, value: any, trigger?: boolean): void;
}

/**
 * Implements an AetherComponent that does not have any children, and servers as the base
 * class for the AetherComposite type. The corresponding react component should NOT have
 * any children that use Aether functionality; for those cases, use AetherComposite instead.
 */
export abstract class Leaf<State extends z.ZodTypeAny, InternalState extends {} = {}>
  implements Component
{
  readonly type: string;
  readonly key: string;

  private readonly sender: Sender<WorkerMessage>;

  private readonly _internalState: InternalState;
  private _state: z.output<State> | undefined;
  private _prevState: z.output<State> | undefined;
  private _deleted: boolean = false;
  protected readonly parentCtxValues: Map<string, any>;
  protected readonly childCtxValues: Map<string, any>;
  protected readonly childCtxChangedKeys: Set<string>;
  readonly instrumentation: alamos.Instrumentation;

  schema: State | undefined = undefined;

  constructor(
    key: string,
    type: string,
    sender: Sender<WorkerMessage>,
    instrumentation: alamos.Instrumentation,
  ) {
    this.type = type;
    this.key = key;
    this.sender = sender;
    this._internalState = {} as InternalState;
    this.instrumentation = instrumentation.child(`${this.type}(${this.key})`);
    this.parentCtxValues = new Map();
    this.childCtxValues = new Map();
    this.childCtxChangedKeys = new Set();
  }

  private get _schema(): State {
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
    next: state.SetArg<z.input<State> | z.output<State>, z.output<State>>,
  ): void {
    const nextState: z.input<State> = state.executeSetter(next, this._state);
    this._prevState = { ...this._state };
    this._state = prettyParse(this._schema, nextState, `${this.type}:${this.key}`);
    this.sender.send({ key: this.key, state: nextState });
  }

  /** @returns the current state of the component. */
  get state(): z.output<State> {
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
  get prevState(): z.output<State> {
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
    _: () => Component,
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

  async _updateContext(
    values: ContextMap,
    triggerAfterUpdate: boolean = true,
  ): Promise<void> {
    values.forEach((value, key) => this.parentCtxValues.set(key, value));
    if (triggerAfterUpdate) await this.afterUpdate(this.ctx);
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

const createChildContext = (
  parentValues: Map<string, any>,
  childValues: Map<string, any>,
): ContextMap => {
  const childCtx = new Map(parentValues);
  childValues.forEach((value, key) => childCtx.set(key, value));
  return childCtx;
};

/**
 * AetherComposite is an implementation of AetherComponent that allows it to maintain
 * child components. It is the base class for all composite components, and should not
 * be used directly.
 */
export class Composite<
    S extends z.ZodTypeAny,
    IS extends {} = {},
    C extends Component = Component,
  >
  extends Leaf<S, IS>
  implements Component
{
  private _children: Map<string, C>;

  constructor(
    key: string,
    type: string,
    sender: Sender<WorkerMessage>,
    instrumentation: alamos.Instrumentation,
  ) {
    super(key, type, sender, instrumentation);
    this._children = new Map();
  }

  /** @returns a readonly array of the children of the component. */
  get children(): readonly C[] {
    return Array.from(this._children.values());
  }

  async _updateState(
    path: string[],
    state: UnknownRecord,
    create: () => Component,
  ): Promise<void> {
    if (this.deleted) return;
    const isChildUpdate = path.length > 1;
    const [key, ...subPath] = path;
    if (isChildUpdate) {
      const childKey = subPath[0];
      const child = this.getChild(childKey);
      if (child != null) return child._updateState(subPath, state, create);
      const newChild = create();
      const childCtx = createChildContext(this.parentCtxValues, this.childCtxValues);
      await newChild._updateContext(childCtx, false);
      await newChild._updateState(subPath, state, create);
      this._children.set(childKey, newChild as C);
      return;
    }
    if (key !== this.key)
      throw new UnexpectedError(
        `[Composite.update] - ${this.type}:${this.key} received a key ${key} but expected ${this.key}`,
      );
    this.childCtxChangedKeys.clear();
    await super._updateState(path, state, create);
    if (this.childCtxChangedKeys.size == 0) return;
    const childCtx = createChildContext(this.parentCtxValues, this.childCtxValues);
    for (const c of this.children) await c._updateContext(childCtx);
  }

  async _updateContext(
    values: ContextMap,
    triggerAfterUpdate: boolean = true,
  ): Promise<void> {
    this.childCtxChangedKeys.clear();
    await super._updateContext(values, triggerAfterUpdate);
    const childCtx = createChildContext(this.parentCtxValues, this.childCtxValues);
    if (!triggerAfterUpdate) return;
    for (const c of this.children) await c._updateContext(childCtx);
  }

  async _delete(path: string[]): Promise<void> {
    const [key, subPath] = this.getRequiredKey(path);
    if (subPath.length === 0) {
      if (key !== this.key)
        throw new Error(
          `[Composite.delete] - ${this.type}:${this.key} received a key ${key} but expected ${this.key}`,
        );
      const children = this.children;
      this._children = new Map();
      for (const c of children) await c._delete([c.key]);
      await super._delete([this.key]);
    }
    const child = this.getChild(subPath[0]);
    if (child == null) return;
    if (subPath.length > 1) await child._delete(subPath);
    else {
      this._children.delete(child.key);
      await child._delete(subPath);
    }
  }

  getRequiredKey(path: string[], type?: string): [string, string[]] {
    const [key, ...subPath] = path;
    if (key == null)
      throw new Error(
        `Composite ${this.type}:${this.key} received an empty path${
          type != null ? ` for ${type}` : ""
        }`,
      );
    return [key, subPath];
  }

  /**
   * Finds a child component with the given key.
   *
   * @param key - the key of the child component to find.
   * @returns the child component, or null if no child component with the given key
   */
  getChild<T extends C = C>(key: string): T | null {
    return (this._children.get(key) ?? null) as T | null;
  }

  /**
   * Finds all children of the component with the given type
   *
   * @param types - the type of the children to find
   * @returns an array of all children of the component with the given type
   */
  childrenOfType<T extends C = C>(...types: Array<T["type"]>): readonly T[] {
    return this.children.filter((c) =>
      types.includes(c.type),
    ) as unknown as readonly T[];
  }
}

export type ComponentRegistry = Record<string, ComponentConstructor>;

const aetherRootState = z.object({});

export interface RootProps {
  worker: SenderHandler<WorkerMessage, MainMessage>;
  registry: ComponentRegistry;
  instrumentation?: alamos.Instrumentation;
}

const shouldNotCallCreate = () => {
  throw new Error("should not call create");
};

export class Root extends Composite<typeof aetherRootState> {
  wrap: SenderHandler<WorkerMessage, MainMessage>;

  private static readonly TYPE = "root";
  private static readonly KEY = "root";
  private readonly registry: ComponentRegistry;

  private count: number = 0;

  private readonly mu = new Mutex();

  schema = aetherRootState;

  constructor({
    worker: wrap,
    instrumentation = alamos.Instrumentation.NOOP,
    registry,
  }: RootProps) {
    super(Root.KEY, Root.TYPE, wrap, instrumentation);
    this.count = 0;
    this.wrap = wrap;
    this.registry = registry;
  }

  static async render(props: RootProps): Promise<Root> {
    const root = new Root(props);
    await root._updateState([Root.KEY], {}, shouldNotCallCreate);
    root.wrap.handle((msg) => {
      root.count++;
      void root.mu.runExclusive(async () => {
        await root.handle(msg);
      });
    });
    return root;
  }
  async handle(msg: MainMessage): Promise<void> {
    const { path, variant, type } = msg;
    if (variant === "delete") await this._delete(path);
    else
      await this._updateState(path, msg.state, () =>
        this.create(path[path.length - 1], type),
      );
  }

  create(key: string, type: string): Component {
    const Constructor = this.registry[type];
    if (Constructor == null)
      throw new UnexpectedError(`[Root.create] - ${type} not found in registry`);
    return new Constructor(key, type, this.wrap, this.instrumentation);
  }
}

export const render = (props: RootProps): void => {
  void Root.render(props);
};
