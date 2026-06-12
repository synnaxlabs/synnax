// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, type Synnax } from "@synnaxlabs/client";
import { deep } from "@synnaxlabs/x";

import { type flux } from "@/flux/aether";

export const FLUX_STORE_KEY = "channels";

export interface FluxStore extends flux.UnaryStore<channel.Key, channel.Channel> {}

export interface FluxSubStore extends flux.Store {
  [FLUX_STORE_KEY]: FluxStore;
}

const SET_CHANNEL_LISTENER: flux.ChannelListener<
  FluxSubStore,
  typeof channel.payloadZ
> = {
  channel: channel.SET_CHANNEL_NAME,
  schema: channel.payloadZ,
  onChange: async ({ store, changed, client }) =>
    store.channels.set(client.channels.sugar(changed)),
};

const DELETE_CHANNEL_LISTENER: flux.ChannelListener<FluxSubStore, typeof channel.keyZ> =
  {
    channel: channel.DELETE_CHANNEL_NAME,
    schema: channel.keyZ,
    onChange: ({ store, changed }) => store.channels.delete(changed),
  };

export const FLUX_STORE_CONFIG: flux.UnaryStoreConfig<
  FluxSubStore,
  channel.Key,
  channel.Channel
> = {
  equal: (a, b) => deep.equal(a.payload, b.payload),
  listeners: [SET_CHANNEL_LISTENER, DELETE_CHANNEL_LISTENER],
};

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
