// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type cache } from "@/cache";
import { type Group, groupZ, type Key, keyZ } from "@/group/types.gen";

export const SET_CHANNEL_NAME = "sy_group_set";
export const DELETE_CHANNEL_NAME = "sy_group_delete";

export const STORE_KEY = "groups";

/** Registers the group table on the given cache. */
export const bindStore = (engine: cache.Cache): void => {
  const table = () => engine.table<Key, Group>(STORE_KEY);
  const set: cache.ChannelListener<{}, typeof groupZ> = {
    channel: SET_CHANNEL_NAME,
    schema: groupZ,
    onChange: ({ changed }) => table().set(changed.key, changed),
  };
  const del: cache.ChannelListener<{}, typeof keyZ> = {
    channel: DELETE_CHANNEL_NAME,
    schema: keyZ,
    onChange: ({ changed }) => table().delete(changed),
  };
  engine.registerTable<Key, Group>(STORE_KEY, { listeners: [set, del] });
};
