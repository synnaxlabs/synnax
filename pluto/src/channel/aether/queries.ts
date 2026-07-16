// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, type Synnax } from "@synnaxlabs/client";

import { type flux } from "@/flux/aether";

export const FLUX_STORE_KEY = "channels";

export interface FluxStore extends flux.UnaryStore<channel.Key, channel.Channel> {}

export interface FluxSubStore extends flux.Store {
  [FLUX_STORE_KEY]: FluxStore;
}

/**
 * Resolves the channel with the given key, returning it from the store if cached and
 * otherwise retrieving it from the cluster and populating the store. Subsequent calls
 * for the same key, on any consumer sharing the store, are served from cache.
 *
 * @throws QueryError if the channel does not exist.
 */
export const retrieveCached = async (
  client: Synnax,
  store: FluxSubStore,
  key: channel.Key,
): Promise<channel.Channel> => {
  const cached = store.channels.get(key);
  if (cached != null) return cached;
  const ch = await client.channels.retrieve(key);
  store.channels.set(ch.key, ch);
  return ch;
};
