// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Instrumentation } from "@synnaxlabs/alamos";
import { UnexpectedError } from "@synnaxlabs/client";
import { Sender, SenderHandler } from "@synnaxlabs/x";
import { z } from "zod";

import { MainMessage, WorkerMessage } from "@/core/aether/message";
import { SetStateArg, executeStateSetter } from "@/util/state";

type AetherUpdateVariant = "state" | "context";

/** An update to an AetherComponent from the main React tree. */
export interface AetherUpdate {
  variant: AetherUpdateVariant;
  ctx: AetherContext;
  path: string[];
  type: string;
  state: any;
}

/**
 * A component in the Aether tree. Each component instance has a unique key identifying
 * it within the tree, and also has a type identifying it's class. Components
 * implementing different functionality should have different types, and these types
 * typically correlate to the corresponding name of the component in the react tree.
 */
export interface AetherComponent {
  type: string;
  key: string;
  internalUpdate: (update: AetherUpdate) => void;
  internalDelete: (path: string[]) => void;
}

export type AetherComponentConstructor = (update: AetherUpdate) => AetherComponent;

export class AetherContext {
  readonly providers: Map<string, any>;
  private readonly registry: Record<string, AetherComponentConstructor>;
  private readonly sender: Sender<WorkerMessage>;
  changed: boolean;

  constructor(
    sender: Sender<WorkerMessage>,
    registry: Record<string, AetherComponentConstructor>,
    providers: Map<string, any> = new Map()
  ) {
    this.providers = providers;
    this.registry = registry;
    this.changed = false;
    this.sender = sender;
  }

  setState(key: string, state: any, transfer: Transferable[] = []): void {
    this.sender.send({ key, state }, transfer);
  }

  copy(add?: AetherContext): AetherContext {
    const cpy = new AetherContext(this.sender, this.registry, new Map());
    this.providers.forEach((value, key) => {
      cpy.providers.set(key, value);
    });
    add?.providers.forEach((value, key) => {
      cpy.providers.set(key, value);
    });
    cpy.changed = false;
    return cpy;
  }

  getOptional<P>(key: string): P | null {
    return this.providers.get(key) ?? null;
  }

  set<P>(key: string, value: P): void {
    this.providers.set(key, value);
    this.changed = true;
  }

  create<C extends AetherComponent>(update: AetherUpdate): C {
    const factory = this.registry[update.type];
    if (factory == null)
      throw new Error(`[AetherRoot.create] - could not find component ${update.type}`);
    const c = factory(update) as C;
    c.internalUpdate(update);
    return c;
  }

  get<P>(key: string): P {
    const value = this.providers.get(key);
    if (value == null)
      throw new Error(`[AetherRoot.get] - could not find provider ${key}`);
    return value;
  }
}

/**
 * Implements an AtherComponent that does not have any children, and servers as the base
 * class for the AetherComposite type. The corresponding react component should NOT have
 * any children that use Aether functionality; for those cases, use AetherComposite instead.
 *
 * Implementing a leaf component is as easy as defining its schema as a Zod type, and
 * then subclassing AetherLeaf and defining handleUpdate and/or handleDelete.
 * @example
 *
 * export class MyEtherealButton extends AetherLeaf<typeof buttonState> {
 *  constructor(update: Update) {
 *    super(update, buttonState)
 *  }
 *
 *  handleUpdate(ctx: AtherContext) {
 *      // Do something here!
 *  }
 *
 *  handleDelete(ctx: AetherContext) {
 *      // Do something else here!
 *  }
 * }
 */
export class AetherLeaf<S extends z.ZodTypeAny, DS = void> implements AetherComponent {
  readonly type: string;
  readonly key: string;
  _derivedState: DS | undefined;

  _ctx: AetherContext;
  private _state: z.output<S> | undefined;
  private _prevState: z.output<S> | undefined;

  schema: S = undefined as any;

  constructor(u: AetherUpdate) {
    this.type = u.type;
    this.key = u.path[0];
    this._ctx = u.ctx;
  }

  /**
   * Sets the state on the component, communicating the change to the corresponding
   * React component on the main thread.
   *
   * @param state - The new state to set on the component. This can be the state object
   * or a pure function that takes in the previous state and returns the next state.
   */
  setState(state: SetStateArg<z.input<S> | z.output<S>>): void {
    const nextState: z.input<S> = executeStateSetter(state, this._state);
    this._prevState = { ...this._state };
    this._state = this.schema.parse(nextState);
    this.ctx.setState(this.key, nextState);
  }

  get ctx(): AetherContext {
    if (this._ctx == null) throw new UnexpectedError("context not defined");
    return this._ctx;
  }

  /** @returns the current state of the component. */
  get state(): z.output<S> {
    if (this._state == null) throw new UnexpectedError("state not defined");
    return this._state;
  }

  get derived(): DS {
    return this._derivedState as DS;
  }

  /** @returns the previous state of the component. */
  get prevState(): z.output<S> {
    return this._prevState;
  }

  /** @implements AetherComponent */
  internalUpdate({ variant, path, ctx, state }: AetherUpdate): void {
    this._ctx = ctx;
    if (variant === "state") {
      this.validatePath(path);
      const state_ = this.schema.parse(state);
      this._prevState = this._state ?? state_;
      this._state = state_;
    }
    this._derivedState = this.derive();
    this.afterUpdate();
  }

  /** @implements AetherComponent */
  internalDelete(path: string[]): void {
    this.validatePath(path);
    this.handleDelete();
  }

  derive(): DS {
    return undefined as DS;
  }

  afterUpdate(): void {}

  handleDelete(): void {}

  private validatePath(path: string[]): void {
    if (path.length === 0)
      throw new UnexpectedError(
        `[Leaf.setState] - ${this.type}:${this.key} received an empty path`
      );
    const key = path[path.length - 1];
    if (path.length > 1)
      throw new UnexpectedError(
        `[Leaf.setState] - ${this.type}:${this.key} received a subPath ${path.join(
          "."
        )} but is a leaf`
      );
    if (key !== this.key)
      throw new UnexpectedError(
        `[Leaf.setState] - ${this.type}:${this.key} received a key ${key} but expected ${this.key}`
      );
  }
}

export class AetherComposite<
    S extends z.ZodTypeAny,
    DS = void,
    C extends AetherComponent = AetherComponent
  >
  extends AetherLeaf<S, DS>
  implements AetherComponent
{
  children: C[];

  constructor(u: AetherUpdate) {
    super(u);
    this.children = [];
  }

  internalUpdate(u: AetherUpdate): void {
    const { variant, path } = u;

    // We're doing a context update.
    if (variant === "context") {
      this._ctx = u.ctx.copy(this._ctx);
      this.updateContext({ ...u, ctx: this._ctx });
      return;
    }

    const [key, subPath] = this.getRequiredKey(path);
    if (subPath.length === 0) return this.updateThis(key, { ...u, ctx: this.ctx });
    return this.updateChild(key, subPath, { ...u, ctx: this.ctx });
  }

  private updateContext(u: AetherUpdate): void {
    super.internalUpdate(u);
    this.children.forEach((c) => c.internalUpdate(u));
  }

  private updateChild(key: string, subPath: string[], u: AetherUpdate): void {
    const childKey = subPath[0];
    const child = this.findChild(childKey);
    if (child != null) return child.internalUpdate({ ...u, path: subPath });
    if (subPath.length > 1)
      throw new Error(
        `[Composite.setState] - ${this.type}:${this.key} could not find child with key ${childKey} while updating `
      );
    this.children.push(u.ctx.create({ ...u, ctx: u.ctx.copy(), path: subPath }));
  }

  private updateThis(key: string, u: AetherUpdate): void {
    const ctx = u.ctx.copy(this._ctx);
    // Check if super altered the context. If so, we need to re-render children.
    if (key !== this.key)
      throw new UnexpectedError(
        `[Composite.update] - ${this.type}:${this.key} received a key ${key} but expected ${this.key}`
      );
    super.internalUpdate({ ...u, ctx });
    if (!ctx.changed) return;
    this._ctx = ctx;
    this.children.forEach((c) =>
      c.internalUpdate({ ...u, ctx: this.ctx, variant: "context" })
    );
  }

  internalDelete(path: string[]): void {
    const [key, subPath] = this.getRequiredKey(path);
    if (subPath.length === 0) {
      if (key !== this.key) {
        throw new Error(
          `[Composite.delete] - ${this.type}:${this.key} received a key ${key} but expected ${this.key}`
        );
      }
      const c = this.children;
      this.children = [];
      c.forEach((c) => c.internalDelete([c.key]));
      super.internalDelete([this.key]);
      return;
    }
    const child = this.findChild(subPath[0]);
    if (child == null) return;
    if (subPath.length > 1) child.internalDelete(subPath);
    else {
      this.children.splice(this.children.indexOf(child), 1);
      child.internalDelete(subPath);
    }
  }

  getRequiredKey(path: string[], type?: string): [string, string[]] {
    const [key, ...subPath] = path;
    if (key == null)
      throw new Error(
        `Composite ${this.type}:${this.key} received an empty path` +
          (type != null ? ` for ${type}` : "")
      );
    return [key, subPath];
  }

  findChild<T extends C = C>(key: string): T | null {
    return (this.children.find((c) => c.key === key) ?? null) as T | null;
  }

  childrenOfType<T extends C = C>(key: string): T[] {
    return this.children.filter((c) => c.type === key) as T[];
  }
}

export type AetherComponentRegistry = Record<string, AetherComponentConstructor>;

const aetherRootState = z.object({});

export interface AetherRootProps {
  worker: SenderHandler<WorkerMessage, MainMessage>;
  registry: AetherComponentRegistry;
  instrumentation?: Instrumentation;
}

export class AetherRoot extends AetherComposite<typeof aetherRootState> {
  wrap: SenderHandler<WorkerMessage, MainMessage>;
  instrumentation: Instrumentation;

  private static readonly TYPE = "root";
  private static readonly KEY = "root";

  private static readonly ZERO_UPDATE: Omit<AetherUpdate, "ctx"> = {
    path: [AetherRoot.KEY],
    type: AetherRoot.TYPE,
    variant: "state",
    state: {},
  };

  static readonly schema = aetherRootState;
  schema = AetherRoot.schema;

  static render(props: AetherRootProps): AetherRoot {
    return new AetherRoot(props);
  }

  private constructor({ worker: wrap, registry, instrumentation }: AetherRootProps) {
    const ctx = new AetherContext(wrap, registry, new Map());
    const u = { ctx, ...AetherRoot.ZERO_UPDATE };
    super(u);
    this.internalUpdate(u);
    this.wrap = wrap;
    this.wrap.handle(this.handle.bind(this));
    this.instrumentation = instrumentation ?? Instrumentation.NOOP;
  }

  handle(msg: MainMessage): void {
    try {
      console.log(msg);
      if (msg.variant === "delete") this.internalDelete(msg.path);
      else {
        const u: AetherUpdate = {
          ...msg,
          variant: "state",
          ctx: this.ctx,
        };
        console.log(u);
        this.internalUpdate(u);
      }
      console.log(msg);
      console.log(this.children);
    } catch (e) {
      console.log(msg);
      console.log(this.children);
      console.error(e);
    }
  }
}

export const render = AetherRoot.render;
