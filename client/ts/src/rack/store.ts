// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type cache } from "@/cache";
import { type Key, keyZ, type Payload, payloadZ } from "@/rack/types.gen";

export const SET_CHANNEL_NAME = "sy_rack_set";
export const DELETE_CHANNEL_NAME = "sy_rack_delete";

export const STORE_KEY = "racks";

/** Registers the rack table on the given cache. */
export const bindStore = (engine: cache.Cache): void => {
  const table = () => engine.table<Key, Omit<Payload, "status">>(STORE_KEY);
  const set: cache.ChannelListener<{}, typeof payloadZ> = {
    channel: SET_CHANNEL_NAME,
    schema: payloadZ,
    onChange: ({ changed: { status: _, ...rack } }) => table().set(rack.key, rack),
  };
  const del: cache.ChannelListener<{}, typeof keyZ> = {
    channel: DELETE_CHANNEL_NAME,
    schema: keyZ,
    onChange: ({ changed }) => table().delete(changed),
  };
  engine.registerTable<Key, Omit<Payload, "status">>(STORE_KEY, {
    listeners: [set, del],
  });
};
