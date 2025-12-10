// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type framer, type Synnax } from "@synnaxlabs/client";
import { type destructor } from "@synnaxlabs/x";

import {
  createStore,
  type InternalStore,
  scopeStore,
  type Store,
  type StoreConfig,
} from "@/flux/core/store";
import { openStreamer as fluxOpenStreamer } from "@/flux/core/streamer";
import { type status } from "@/status/aether";

interface ClientArgs<ScopedStore extends Store> {
  client: Synnax | null;
  openStreamer?: framer.StreamOpener;
  storeConfig: StoreConfig<ScopedStore>;
  handleError: status.ErrorHandler;
  handleAsyncError: status.AsyncErrorHandler;
}

export class Client<ScopedStore extends Store = Store> {
  private readonly store: InternalStore;
  private readonly streamCloser: Promise<destructor.Async> | null = null;
  readonly client: Synnax | null;

  constructor({
    client,
    openStreamer,
    storeConfig,
    handleError,
    handleAsyncError,
  }: ClientArgs<ScopedStore>) {
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
