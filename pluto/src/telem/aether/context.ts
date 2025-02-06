// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { UnexpectedError } from "@synnaxlabs/client";
import { id, toArray } from "@synnaxlabs/x";
import { deep } from "@synnaxlabs/x/deep";
import { type Destructor } from "@synnaxlabs/x/destructor";
import { type observe } from "@synnaxlabs/x/observe";

import { type aether } from "@/aether/aether";
import { CompoundTelemFactory, type Factory } from "@/telem/aether/factory";
import { PipelineFactory } from "@/telem/aether/pipeline";
import { type Sink, type Source, type Spec } from "@/telem/aether/telem";

/**
 * Provides utilities for creating and managing telemetry sources and sinks.
 */
export class Context {
  private factory: CompoundTelemFactory;
  readonly key: string;
  readonly parent?: Context;

  constructor(factory: CompoundTelemFactory, parent?: Context) {
    this.factory = factory;
    this.key = id.id();
    this.parent = parent;
  }

  child(factories: Factory | Factory[], parent?: Context): Context {
    const next = new CompoundTelemFactory([
      ...this.factory.factories,
      ...toArray(factories),
    ]);
    next.add(new PipelineFactory(next));
    return new Context(next, parent);
  }

  create<T>(spec: Spec): T {
    const telem = this.factory.create(spec);
    if (telem == null)
      throw new UnexpectedError(
        `Telemetry service could not find a source for type ${spec.type}`,
      );
    return telem as T;
  }
}

const CONTEXT_KEY = "pluto-telem-context";

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
  async value(): Promise<V> {
    return await this.wrapped.value();
  }

  async cleanup(): Promise<void> {
    await this.wrapped.cleanup?.();
  }

  onChange(handler: observe.Handler<void>): Destructor {
    return this.wrapped.onChange(handler);
  }
}

class MemoizedSink<V> extends Memoized<Sink<V>> {
  async set(value: V): Promise<void> {
    return await this.wrapped.set(value);
  }

  async cleanup(): Promise<void> {
    await this.wrapped.cleanup?.();
  }
}

export const useSource = async <V>(
  ctx: aether.Context,
  spec: Spec,
  prev: Source<V>,
): Promise<MemoizedSource<V>> => {
  const prov = useContext(ctx);
  if (prev instanceof MemoizedSource) {
    if (!prev.shouldUpdate(prov, spec)) return prev;
    await prev.cleanup?.();
  }
  return new MemoizedSource<V>(prov.create(spec), prov, spec);
};

export const useSink = async <V>(
  ctx: aether.Context,
  spec: Spec,
  prev: Sink<V>,
): Promise<MemoizedSink<V>> => {
  const prov = useContext(ctx);
  if (prev instanceof MemoizedSink) {
    if (!prev.shouldUpdate(prov, spec)) return prev;
    await prev.cleanup?.();
  }
  return new MemoizedSink<V>(prov.create(spec), prov, spec);
};
