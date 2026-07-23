// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type framer, type Synnax } from "@synnaxlabs/client";
import { type destructor } from "@synnaxlabs/x";

import { QueryCache } from "@/flux/base/queryCache";
import {
  createStore,
  type InternalStore,
  scopeStore,
  type Store,
  type StoreConfig,
} from "@/flux/base/store";
import { openStreamer as fluxOpenStreamer } from "@/flux/base/streamer";
import { type Query } from "@/flux/base/types";
import { type state } from "@/state";
import { type status } from "@/status/aether";

interface ClientParams<ScopedStore extends Store> {
  client: Synnax | null;
  openStreamer?: framer.StreamOpener;
  storeConfig: StoreConfig<ScopedStore>;
  handleError: status.ErrorHandler;
  handleAsyncError: status.AsyncErrorHandler;
}

export class Client<ScopedStore extends Store = Store> {
  private readonly store: InternalStore;
  private readonly streamCloser: Promise<destructor.Async> | null = null;
  // Lazy registry of per-`createRetrieve` query caches, keyed by a unique
  // cache key generated at createRetrieve construction. Stored at the
  // erased type so the map is homogeneous; `getCache` reifies the typed
  // view, which is sound because each key is bound to one (Q, D) pair at
  // the call to `new QueryCache<Q, D>()`.
  private readonly caches = new Map<string, QueryCache<Query, state.State>>();
  readonly client: Synnax | null;

  constructor({
    client,
    openStreamer,
    storeConfig,
    handleError,
    handleAsyncError,
  }: ClientParams<ScopedStore>) {
    this.store = createStore(storeConfig, handleError);
    this.client = client;
    if (client == null) return;
    openStreamer ??= client?.openStreamer.bind(client);
    this.streamCloser = fluxOpenStreamer({
      client,
      storeConfig,
      handleError: handleAsyncError,
      store: scopeStore<ScopedStore>(this.store, ""),
      openStreamer,
    });
  }

  /// Returns the query cache for the given key, creating one on first use.
  /// Each createRetrieve operation owns a unique key; reads and writes
  /// against the returned instance are typed concretely.
  getCache<Q extends Query, D extends state.State>(key: string): QueryCache<Q, D> {
    let cache = this.caches.get(key);
    if (cache == null) {
      cache = new QueryCache<Q, D>();
      this.caches.set(key, cache);
    }
    return cache as unknown as QueryCache<Q, D>;
  }

  async awaitInitialized() {
    if (this.streamCloser == null) return;
    await this.streamCloser;
  }

  scopedStore<ScopedStore extends Store>(scope: string): ScopedStore {
    return scopeStore<ScopedStore>(this.store, scope);
  }

  async close() {
    if (this.streamCloser == null) return;
    const destructor = await this.streamCloser;
    await destructor();
  }
}
