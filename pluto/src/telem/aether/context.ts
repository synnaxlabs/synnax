// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { UnexpectedError } from "@synnaxlabs/client";
import { array, deep, type destructor, id, type observe } from "@synnaxlabs/x";

import { type aether } from "@/aether/aether";
import {
  CompoundFactory,
  type CreateOptions,
  type Factory,
} from "@/telem/aether/factory";
import { PipelineFactory } from "@/telem/aether/pipeline";
import { type Sink, type Source, type Spec } from "@/telem/aether/telem";

/**
 * Provides utilities for creating and managing telemetry sources and sinks.
 */
export class Context {
  private factory: CompoundFactory;
  readonly key: string;
  readonly parent?: Context;

  constructor(factory: CompoundFactory, parent?: Context) {
    this.factory = factory;
    this.key = id.create();
    this.parent = parent;
  }

  child(factories: Factory | Factory[], parent?: Context): Context {
    const next = new CompoundFactory([
      ...this.factory.factories,
      ...array.toArray(factories),
    ]);
    next.add(new PipelineFactory(next));
    return new Context(next, parent);
  }

  create<T>(spec: Spec, options?: CreateOptions): T {
    const telem = this.factory.create(spec, options);
    if (telem == null)
      throw new UnexpectedError(
        `Telemetry service could not find a source for type ${spec.type}`,
      );
    return telem as T;
  }
}

export const CONTEXT_KEY = "pluto-telem-context";

export const useContext = (ctx: aether.Context): Context =>
  ctx.get<Context>(CONTEXT_KEY);

export const setContext = (ctx: aether.Context, prov: Context): void =>
  ctx.set(CONTEXT_KEY, prov);

export const useChildContext = (
  ctx: aether.Context,
  factories: Factory | Factory[],
  prev: Context,
): Context => {
  const tCtx = useContext(ctx);
  if (tCtx != null && prev != null && tCtx.key === prev?.parent?.key) return prev;
  const next = tCtx.child(factories, tCtx);
  ctx.set(CONTEXT_KEY, next);
  return next;
};

class Memoized<V> {
  private readonly spec: Spec;
  readonly wrapped: V;
  private readonly prevProv: Context;

  constructor(wrapped: V, prevProv: Context, prevSpec: Spec) {
    this.wrapped = wrapped;
    this.spec = prevSpec;
    this.prevProv = prevProv;
  }

  shouldUpdate(prov: Context, spec: Spec): boolean {
    return this.prevProv.key !== prov.key || !deep.equal(this.spec, spec);
  }
}

class MemoizedSource<V> extends Memoized<Source<V>> {
  value(): V {
    return this.wrapped.value();
  }

  cleanup(): void {
    this.wrapped.cleanup?.();
  }

  onChange(handler: observe.Handler<void>): destructor.Destructor {
    return this.wrapped.onChange(handler);
  }
}

class MemoizedSink<V> extends Memoized<Sink<V>> {
  set(...values: V[]): void {
    this.wrapped.set(...values);
  }

  cleanup(): void {
    this.wrapped.cleanup?.();
  }
}

export const useSource = <V>(
  ctx: aether.Context,
  spec: Spec,
  prev: Source<V>,
  options?: CreateOptions,
): MemoizedSource<V> => {
  const prov = useContext(ctx);
  if (prev instanceof MemoizedSource) {
    if (!prev.shouldUpdate(prov, spec)) return prev;
    prev.cleanup?.();
  }
  return new MemoizedSource<V>(prov.create(spec, options), prov, spec);
};

export const useSink = <V>(
  ctx: aether.Context,
  spec: Spec,
  prev: Sink<V>,
  options?: CreateOptions,
): MemoizedSink<V> => {
  const prov = useContext(ctx);
  if (prev instanceof MemoizedSink) {
    if (!prev.shouldUpdate(prov, spec)) return prev;
    prev.cleanup?.();
  }
  return new MemoizedSink<V>(prov.create(spec, options), prov, spec);
};
