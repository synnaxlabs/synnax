// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { deep } from "@synnaxlabs/x/deep";
import { type Destructor } from "@synnaxlabs/x/destructor";
import { type observe } from "@synnaxlabs/x/observe";

import { type aether } from "@/aether/aether";
import { type Factory } from "@/telem/aether/factory";
import { type Sink, type Source, type Spec } from "@/telem/aether/telem";

export interface Provider {
  clusterKey: string;
  key: string;
  equals: (other: Provider) => boolean;
  registerFactory: (f: Factory) => void;
  create: <T>(props: Spec) => T;
}

const CONTEXT_KEY = "pluto-telem-context";

export const useProvider = (ctx: aether.Context): Provider =>
  ctx.get<Provider>(CONTEXT_KEY);

export const setProvider = (ctx: aether.Context, prov: Provider): void =>
  ctx.set(CONTEXT_KEY, prov);

export const registerFactory = (ctx: aether.Context, f: Factory): void =>
  useProvider(ctx).registerFactory(f);

class MemoizedSource<V> implements Source<V> {
  private readonly spec: Spec;
  private readonly wrapped: Source<V>;
  private readonly prevKey: string;

  constructor(wrapped: Source<V>, prevProv: Provider, prevSpec: Spec) {
    this.wrapped = wrapped;
    this.spec = prevSpec;
    this.prevKey = prevProv.clusterKey;
  }

  get cache(): unknown {
    return this.wrapped.cache;
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

  shouldUpdate(prov: Provider, spec: Spec): boolean {
    return this.prevKey !== prov.clusterKey || !deep.equal(this.spec, spec);
  }
}

class MemoizedSink<V> implements Sink<V> {
  private readonly spec: Spec;
  private readonly prov: Provider;
  private readonly prevKey: string;
  private readonly wrapped: Sink<V>;

  constructor(wrapped: Sink<V>, prevProv: Provider, prevSpec: Spec) {
    this.wrapped = wrapped;
    this.spec = prevSpec;
    this.prov = prevProv;
    this.prevKey = prevProv.clusterKey;
  }

  async set(value: V): Promise<void> {
    return await this.wrapped.set(value);
  }

  async cleanup(): Promise<void> {
    await this.wrapped.cleanup?.();
  }

  shouldUpdate(prov: Provider, spec: Spec): boolean {
    return this.prevKey !== prov.clusterKey || !deep.equal(this.spec, spec);
  }
}

export const useSource = async <V>(
  ctx: aether.Context,
  spec: Spec,
  prev: Source<V>,
): Promise<MemoizedSource<V>> => {
  const prov = useProvider(ctx);
  if (prev instanceof MemoizedSource) {
    if (!prev.shouldUpdate(prov, spec)) return prev;
    spec.cache = prev.cache;
    await prev.cleanup?.();
  }
  return new MemoizedSource<V>(prov.create(spec), prov, spec);
};

export const useSink = async <V>(
  ctx: aether.Context,
  spec: Spec,
  prev: Sink<V>,
): Promise<MemoizedSink<V>> => {
  const prov = useProvider(ctx);
  if (prev instanceof MemoizedSink) {
    if (!prev.shouldUpdate(prov, spec)) return prev;
    await prev.cleanup?.();
  }
  return new MemoizedSink<V>(prov.create(spec), prov, spec);
};
