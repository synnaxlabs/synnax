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

import { WorkerMessage } from "@/core/aether/message";

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

  constructor(update: Update, schema: S) {
    this.type = update.type;
    this.key = update.path[0];
    this.schema = schema;
    this._state = this.schema.parse(update.state);
    this._ctx = update.ctx;
  }

  setState(state: z.input<S>): void {
    this._state = this.schema.parse(state);
    this._ctx.setState(this.key, this._state);
  }

  get state(): z.output<S> {
    return this._state;
  }

  update({ path, ctx, state }: Update): z.output<S> {
    this._ctx = ctx;
    if (state != null) {
      this.validatePath(path);
      this._state = this.schema.parse(state);
    }
    this.handleUpdate(ctx);
    return this._state;
  }

  handleUpdate(ctx: AetherContext): void {}

  delete(path: string[]): void {
    this.validatePath(path);
    this.handleDelete();
  }

  handleDelete(): void {}

  private validatePath(path: string[]): void {
    if (path.length === 0)
      throw new Error(
        `[Leaf.setState] - ${this.type}:${this.key} received an empty path`
      );
    const key = path[path.length - 1];
    if (path.length > 1)
      throw new Error(
        `[Leaf.setState] - ${this.type}:${this.key} received a subPath ${path.join(
          "."
        )} but is a leaf`
      );
    if (key !== this.key)
      throw new Error(
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
  readonly children: C[];

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
        throw new Error(
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
        `[Composite.setState] - ${this.type}:${this.key} could not find child with key ${type}:${childKey}`
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
      return;
    }
    const child = this.findChild(subPath[0]);
    if (child == null) {
      throw new Error(
        `[Composite.delete] - ${this.type}:${this.key} could not find child with key ${key}`
      );
    } else if (subPath.length > 1) child.delete(subPath);
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
  private readonly worker: TypedWorker<WorkerMessage>;
  changed: boolean;

  constructor(
    worker: TypedWorker<WorkerMessage>,
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

class AetherRoot {
  wrap: TypedWorker<WorkerMessage>;
  root: AetherComponent | null;
  ctx: AetherContext;

  static render(
    wrap: TypedWorker<WorkerMessage>,
    registry: Record<string, AetherComponentConstructor>
  ): AetherRoot {
    return new AetherRoot(wrap, registry);
  }

  private constructor(
    wrap: TypedWorker<WorkerMessage>,
    registry: Record<string, AetherComponentConstructor>
  ) {
    this.ctx = new AetherContext(wrap, registry);
    this.wrap = wrap;
    this.root = null;
    this.wrap.handle(this.handle.bind(this));
  }

  handle(msg: WorkerMessage): void {
    if (msg.variant === "backward")
      throw new UnexpectedError(
        `[AetherRoot.handle] - received a backward update in worker`
      );
    if (msg.variant === "delete") {
      if (this.root == null)
        throw new Error(
          `[AetherRoot.handle] - received a delete message but no root is set`
        );
      return this.root.delete(msg.path);
    }

    if (this.root == null && msg.path.length > 1) {
      console.warn(
        `[AetherRoot.handle] - received a path ${msg.path.join(".")} but no root is set`
      );
      return;
    }
    const change: Update = { ...msg, ctx: this.ctx.child() };
    if (this.root == null) this.root = this.ctx.create(change);
    else this.root.update(change);
  }
}

export const render = AetherRoot.render;
