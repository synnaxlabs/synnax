// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Key, keyZ, type Policy, policyZ } from "@/access/policy/types.gen";
import { type cache } from "@/cache";

export const SET_CHANNEL_NAME = "sy_policy_set";
export const DELETE_CHANNEL_NAME = "sy_policy_delete";

export const STORE_KEY = "policies";

/** Registers the policy table on the given cache. */
export const bindStore = (engine: cache.Cache): void => {
  const table = () => engine.table<Key, Policy>(STORE_KEY);
  const set: cache.ChannelListener<{}, typeof policyZ> = {
    channel: SET_CHANNEL_NAME,
    schema: policyZ,
    onChange: ({ changed }) => table().set(changed.key, changed),
  };
  const del: cache.ChannelListener<{}, typeof keyZ> = {
    channel: DELETE_CHANNEL_NAME,
    schema: keyZ,
    onChange: ({ changed }) => table().delete(changed),
  };
  engine.registerTable<Key, Policy>(STORE_KEY, { listeners: [set, del] });
};
