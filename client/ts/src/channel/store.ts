// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { deep } from "@synnaxlabs/x";

import { type cache } from "@/cache";
import { type Key, keyZ, type Payload, payloadZ } from "@/channel/types.gen";

export const SET_CHANNEL_NAME = "sy_channel_set";
export const DELETE_CHANNEL_NAME = "sy_channel_delete";

export const STORE_KEY = "channels";

/**
 * Registers the channel store on the given engine. Generic over the sugared
 * channel type so this file stays independent of the Channel class.
 */
export const bindStore = <C extends { key: Key; payload: Payload }>(
  engine: cache.Engine,
  sugar: (payload: Payload) => C,
  retrieve: (keys: Key[]) => Promise<C[]>,
): void => {
  const store = () => engine.store<Key, C>(STORE_KEY);
  const set: cache.ChannelListener<{}, typeof payloadZ> = {
    channel: SET_CHANNEL_NAME,
    schema: payloadZ,
    onChange: ({ changed }) => store().set(sugar(changed)),
  };
  const del: cache.ChannelListener<{}, typeof keyZ> = {
    channel: DELETE_CHANNEL_NAME,
    schema: keyZ,
    onChange: ({ changed }) => store().delete(changed),
  };
  engine.registerStore<Key, C>(STORE_KEY, {
    equal: (a, b) => deep.equal(a.payload, b.payload),
    listeners: [set, del],
    refetch: retrieve,
  });
};
