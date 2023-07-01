// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { UnexpectedError } from "@synnaxlabs/client";
import { TypedWorker } from "@synnaxlabs/x";
import { z } from "zod";

import { PsuedoSetStateArg } from "../hooks/useStateRef";

import { MainMessage, WorkerMessage } from "@/core/aether/message";

export interface Update {
  ctx: AetherContext;
  path: string[];
  type: string;
  state: any;
}

export interface AetherComponent {
  type: string;
  key: string;
  update: (update: Update) => void;
  delete: (path: string[]) => void;
}

export class AetherLeaf<S extends z.ZodTypeAny> implements AetherComponent {
  private _ctx: AetherContext;
  readonly type: string;
  readonly key: string;
  readonly schema: S;
  _state: z.output<S>;
  _prevState: z.output<S>;

  constructor(update: Update, schema: S) {
    this.type = update.type;
    this.key = update.path[0];
    this.schema = schema;
    this._state = this.schema.parse(update.state);
    this._ctx = update.ctx;
    this._prevState = this._state;
  }

  setState(state: PsuedoSetStateArg<z.input<S>>): void {
    const nextState: z.input<S> =
      typeof state === "function" ? (state as any)(this._state) : state;
    this._state = this.schema.parse(nextState);
    this._ctx.setState(this.key, nextState);
  }

  get state(): z.output<S> {
    return this._state;
  }

  update({ path, ctx, state }: Update): z.output<S> {
    this._ctx = ctx;
    if (state != null) {
      this.validatePath(path);
      this._prevState = this._state;
      this._state = this.schema.parse(state);
    }
    this.handleUpdate(ctx);
    return this._state;
  }

  get prevState(): z.output<S> {
    return this._prevState;
  }

  handleUpdate(ctx: AetherContext): void {}

  delete(path: string[]): void {
    this.validatePath(path);
    this.handleDelete();
  }

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
    C extends AetherComponent = AetherComponent
  >
  extends AetherLeaf<S>
  implements AetherComponent
{
  children: C[];

  constructor(change: Update, schema: S) {
    super(change, schema);
    this.children = [];
  }

  update(change: Update): void {
    const { path, type, state } = change;

    // We're doing a context update.
    if (state == null) {
      super.update(change);
      this.children.forEach((c) => c.update(change));
      return;
    }

    const [key, subPath] = this.getRequiredKey(path);
    if (subPath.length === 0) {
      // Check if super altered the context. If so, we need to re-render children.
      if (key !== this.key)
        throw new UnexpectedError(
          `[Composite.update] - ${this.type}:${this.key} received a key ${key} but expected ${this.key}`
        );
      change.ctx = change.ctx.child();
      super.update(change);
      if (change.ctx.changed)
        this.children.forEach((c) => {
          c.update({
            ctx: change.ctx,
            path: [],
            type,
            state: null,
          });
        });
      return;
    }

    const childKey = subPath[0];
    const child = this.findChild(childKey);
    if (child != null) return child?.update({ ...change, path: subPath });
    if (subPath.length > 1)
      throw new Error(
        `[Composite.setState] - ${this.type}:${this.key} could not find child with key ${childKey} while updating `
      );
    this.children.push(change.ctx.create({ ...change, path: subPath }));
  }

  delete(path: string[]): void {
    const [key, subPath] = this.getRequiredKey(path);
    if (subPath.length === 0) {
      if (key !== this.key) {
        throw new Error(
          `[Composite.delete] - ${this.type}:${this.key} received a key ${key} but expected ${this.key}`
        );
      }
      const c = this.children;
      this.children = [];
      c.forEach((c) => c.delete([c.key]));
      super.delete([this.key]);
      return;
    }
    const child = this.findChild(subPath[0]);
    if (child == null) return;
    if (subPath.length > 1) child.delete(subPath);
    else {
      this.children.splice(this.children.indexOf(child), 1);
      child.delete(subPath);
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

  findChildrenOfType<T extends C = C>(key: string): T[] {
    return this.children.filter((c) => c.type === key) as T[];
  }
}

export type AetherComponentConstructor = (update: Update) => AetherComponent;

export class AetherContext {
  private readonly providers: Map<string, any>;
  private readonly registry: Record<string, AetherComponentConstructor>;
  private readonly worker: TypedWorker<WorkerMessage, MainMessage>;
  changed: boolean;

  constructor(
    worker: TypedWorker<WorkerMessage, MainMessage>,
    registry: Record<string, AetherComponentConstructor>,
    providers: Map<string, any> = new Map()
  ) {
    this.providers = providers;
    this.registry = registry;
    this.changed = false;
    this.worker = worker;
  }

  setState(key: string, state: any, transfer: Transferable[] = []): void {
    this.worker.send({ variant: "backward", key, state }, transfer);
  }

  child(): AetherContext {
    return new AetherContext(this.worker, this.registry, this.providers);
  }

  getOptional<P>(key: string): P | null {
    return this.providers.get(key) ?? null;
  }

  set<P>(key: string, value: P): void {
    this.providers.set(key, value);
    this.changed = true;
  }

  create<C extends AetherComponent>(update: Update): C {
    const factory = this.registry[update.type];
    if (factory == null)
      throw new Error(`[AetherRoot.create] - could not find component ${update.type}`);
    return factory(update) as C;
  }

  get<P>(key: string): P {
    const value = this.providers.get(key);
    if (value == null)
      throw new Error(`[AetherRoot.get] - could not find provider ${key}`);
    return value;
  }
}

export type AetherComponentRegistry = Record<string, AetherComponentConstructor>;

const aetherRootState = z.object({});

class AetherRoot extends AetherComposite<typeof aetherRootState> {
  wrap: TypedWorker<WorkerMessage, MainMessage>;
  ctx: AetherContext;

  static render(
    wrap: TypedWorker<WorkerMessage, MainMessage>,
    registry: AetherComponentRegistry
  ): AetherRoot {
    return new AetherRoot(wrap, registry);
  }

  private constructor(
    wrap: TypedWorker<WorkerMessage, MainMessage>,
    registry: Record<string, AetherComponentConstructor>
  ) {
    const ctx = new AetherContext(wrap, registry);
    super(
      {
        ctx,
        path: ["root"],
        type: "root",
        state: { ready: false },
      },
      aetherRootState
    );
    this.ctx = ctx;
    this.wrap = wrap;
    this.wrap.handle(this.handle.bind(this));
  }

  handle(msg: MainMessage): void {
    if (msg.variant === "delete") {
      this.delete(msg.path);
    } else {
      const change: Update = { ...msg, ctx: this.ctx.child() };
      this.update(change);
    }
    console.log(msg, this.children);
  }
}

export const render = AetherRoot.render;
