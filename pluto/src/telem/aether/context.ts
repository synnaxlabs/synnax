// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Destructor, deep } from "@synnaxlabs/x";
import { type Handler } from "@synnaxlabs/x/dist/observe/observe";

import { type aether } from "@/aether/aether";
import { type telem } from "@/telem/aether";
import { type Factory } from "@/telem/aether/factory";
import { type Spec } from "@/telem/aether/telem";

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

class MemoizedSource<V> implements telem.Source<V> {
  private readonly spec: Spec;
  private readonly prov: Provider;
  private readonly wrapped: telem.Source<V>;
  private readonly prevKey: string;

  constructor(wrapped: telem.Source<V>, prevProv: Provider, prevSpec: Spec) {
    this.wrapped = wrapped;
    this.spec = prevSpec;
    this.prov = prevProv;
    this.prevKey = prevProv.clusterKey;
  }

  async value(): Promise<V> {
    return await this.wrapped.value();
  }

  async cleanup(): Promise<void> {
    await this.wrapped.cleanup?.();
  }

  onChange(handler: Handler<void>): Destructor {
    return this.wrapped.onChange(handler);
  }

  shouldUpdate(prov: Provider, spec: Spec): boolean {
    return this.prevKey !== prov.clusterKey || !deep.equal(this.spec, spec);
  }
}

class MemoizedSink<V> implements telem.Sink<V> {
  private readonly spec: Spec;
  private readonly prov: Provider;
  private readonly prevKey: string;
  private readonly wrapped: telem.Sink<V>;

  constructor(wrapped: telem.Sink<V>, prevProv: Provider, prevSpec: Spec) {
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
  prev: telem.Source<V>,
): Promise<telem.Source<V>> => {
  const prov = useProvider(ctx);
  if (prev instanceof MemoizedSource) {
    if (!prev.shouldUpdate(prov, spec)) return prev;
    await prev.cleanup?.();
  }
  return new MemoizedSource<V>(prov.create(spec), prov, spec);
};

export const useSink = async <V>(
  ctx: aether.Context,
  spec: Spec,
  prev: telem.Sink<V>,
): Promise<telem.Sink<V>> => {
  const prov = useProvider(ctx);
  if (prev instanceof MemoizedSink) {
    if (!prev.shouldUpdate(prov, spec)) return prev;
    await prev.cleanup?.();
  }
  return new MemoizedSink<V>(prov.create(spec), prov, spec);
};
