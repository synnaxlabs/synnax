// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type cache } from "@/cache";
import { type Key, keyZ, type View, viewZ } from "@/view/types.gen";

export const SET_CHANNEL_NAME = "sy_view_set";
export const DELETE_CHANNEL_NAME = "sy_view_delete";

export const STORE_KEY = "views";

/** Registers the view store on the given engine. */
export const bindStore = (engine: cache.Engine): void => {
  const store = () => engine.store<Key, View>(STORE_KEY);
  const setListener: cache.ChannelListener<{}, typeof viewZ> = {
    channel: SET_CHANNEL_NAME,
    schema: viewZ,
    onChange: ({ changed }) => store().set(changed.key, changed),
  };
  const deleteListener: cache.ChannelListener<{}, typeof keyZ> = {
    channel: DELETE_CHANNEL_NAME,
    schema: keyZ,
    onChange: ({ changed }) => store().delete(changed),
  };
  engine.registerStore<Key, View>(STORE_KEY, {
    listeners: [setListener, deleteListener],
  });
};
