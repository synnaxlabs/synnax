// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

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
  readonly type: string;
  readonly key: string;
  readonly schema: S;
  private stateHook?: (ctx: AetherContext) => void;
  private deleteHook?: () => void;
  state: z.output<S>;

  constructor(change: Update, schema: S) {
    this.type = change.type;
    this.key = change.path[0];
    this.state = schema.parse(change.state);
    this.schema = schema;
  }

  onUpdate(hook: (ctx: AetherContext) => void): void {
    this.stateHook = hook;
  }

  update({ path, ctx, state }: Update): void {
    if (state != null) {
      this.validatePath(path);
      this.state = this.schema.parse(state);
    }
    this.stateHook?.(ctx);
  }

  onDelete(hook: () => void): void {
    this.deleteHook = hook;
  }

  delete(path: string[]): void {
    this.validatePath(path);
    this.deleteHook?.();
  }

  private validatePath(path: string[]): void {
    if (path.length === 0)
      throw new Error(
        `[Leaf.setState] - ${this.type}:${this.key} received an empty path`
      );
    const key = path.pop() as string;
    if (path.length !== 0)
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
      change.ctx = change.ctx.child();
      super.update(change);
      if (change.ctx.changed) this.children.forEach((c) => c.update(change));
    }

    const childKey = subPath[0];
    const child = this.findChild(childKey);
    if (child != null) return child?.update({ ...change, path: subPath });
    if (subPath.length > 1)
      throw new Error(
        `[Composite.setState] - ${this.type}:${this.key} could not find child with key ${key}:${type}`
      );
    this.children.push(change.ctx.create(change));
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
    } else if (subPath.length > 1) child.delete(subPath.slice(1));
    else {
      child.delete(subPath);
      this.children.splice(this.children.indexOf(child), 1);
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

export type AetherComponentConstructor = new (update: Update) => AetherComponent;

export class AetherContext {
  private readonly providers: Map<string, any>;
  private readonly registry: Record<string, AetherComponentConstructor>;
  changed: boolean;

  constructor(
    registry: Record<string, AetherComponentConstructor>,
    providers: Map<string, any> = new Map()
  ) {
    this.providers = providers;
    this.registry = registry;
    this.changed = false;
  }

  child(): AetherContext {
    return new AetherContext(this.registry, this.providers);
  }

  get<P>(key: string): P {
    return this.providers.get(key);
  }

  create<C extends AetherComponent>(change: Update): C {
    return new this.registry[change.type](change) as C;
  }

  set<P>(key: string, value: P): void {
    this.providers.set(key, value);
    this.changed = true;
  }

  getOptional<P>(key: string): P {
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
    this.ctx = new AetherContext(registry);
    this.wrap = wrap;
    this.root = null;
    this.wrap.handle((msg) => this.handle(msg));
  }

  handle(msg: WorkerMessage): void {
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
