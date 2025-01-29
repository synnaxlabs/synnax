// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { UnexpectedError } from "@synnaxlabs/client";
import { id } from "@synnaxlabs/x";
import { deep } from "@synnaxlabs/x/deep";
import { type Destructor } from "@synnaxlabs/x/destructor";
import { type observe } from "@synnaxlabs/x/observe";

import { type aether } from "@/aether/aether";
import { CompoundTelemFactory, type Factory } from "@/telem/aether/factory";
import { PipelineFactory } from "@/telem/aether/pipeline";
import { type Sink, type Source, type Spec } from "@/telem/aether/telem";

export class ContextValue {
  private wrappedFactory: CompoundTelemFactory;
  private readonly key: string;
  private readonly parent?: ContextValue;

  constructor(factory: CompoundTelemFactory, parent?: ContextValue) {
    this.wrappedFactory = factory;
    this.key = id.id();
    this.parent = parent;
  }

  child(f: Factory, parent?: ContextValue): ContextValue {
    const next = new CompoundTelemFactory([...this.wrappedFactory.factories, f]);
    next.add(new PipelineFactory(next));
    return new ContextValue(next, parent);
  }

  equals(other: ContextValue): boolean {
    if (this.key !== other.key) return false;
    if (this.parent == null && other.parent == null) return true;
    if (this.parent == null || other.parent == null) return false;
    return this.parent.equals(other.parent);
  }

  create<T>(spec: Spec): T {
    const telem = this.wrappedFactory.create(spec);
    if (telem == null)
      throw new UnexpectedError(
        `Telemetry service could not find a source for type ${spec.type}`,
      );
    return telem as T;
  }
}

const CONTEXT_KEY = "pluto-telem-context";

export const useContext = (ctx: aether.Context): ContextValue =>
  ctx.get<ContextValue>(CONTEXT_KEY);

export const setContext = (ctx: aether.Context, prov: ContextValue): void =>
  ctx.set(CONTEXT_KEY, prov);

export const useChildContext = (
  ctx: aether.Context,
  f: Factory,
  prev: ContextValue,
): ContextValue => {
  const tCtx = useContext(ctx);
  if (prev != null && prev.equals(tCtx)) return prev;
  const next = tCtx.child(f, tCtx);
  ctx.set(CONTEXT_KEY, next);
  return next;
};

class MemoizedSource<V> implements Source<V> {
  private readonly spec: Spec;
  private readonly wrapped: Source<V>;
  private readonly prevProv: ContextValue;

  constructor(wrapped: Source<V>, prevProv: ContextValue, prevSpec: Spec) {
    this.wrapped = wrapped;
    this.spec = prevSpec;
    this.prevProv = prevProv;
  }

  async value(): Promise<V> {
    return await this.wrapped.value();
  }

  async cleanup(): Promise<void> {
    await this.wrapped.cleanup?.();
  }

  onChange(handler: observe.Handler<void>): Destructor {
    return this.wrapped.onChange(handler);
  }

  shouldUpdate(prov: ContextValue, spec: Spec): boolean {
    if (!this.prevProv.equals(prov)) return true;
    if (!deep.equal(this.spec, spec)) return true;
    return false;
  }
}

class MemoizedSink<V> implements Sink<V> {
  private readonly spec: Spec;
  private readonly prov: ContextValue;
  private readonly wrapped: Sink<V>;

  constructor(wrapped: Sink<V>, prevProv: ContextValue, prevSpec: Spec) {
    this.wrapped = wrapped;
    this.spec = prevSpec;
    this.prov = prevProv;
  }

  async set(value: V): Promise<void> {
    return await this.wrapped.set(value);
  }

  async cleanup(): Promise<void> {
    await this.wrapped.cleanup?.();
  }

  shouldUpdate(prov: ContextValue, spec: Spec): boolean {
    if (!this.prov.equals(prov)) return true;
    if (!deep.equal(this.spec, spec)) return true;
    return false;
  }
}

export const useSource = async <V>(
  ctx: aether.Context,
  spec: Spec,
  prev: Source<V>,
): Promise<MemoizedSource<V>> => {
  const prov = useContext(ctx);
  if (prev instanceof MemoizedSource) {
    const shouldUpdate = prev.shouldUpdate(prov, spec);
    // console.log("shouldUpdate", shouldUpdate);
    if (!shouldUpdate) return prev;
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
