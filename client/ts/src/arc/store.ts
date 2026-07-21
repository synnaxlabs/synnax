// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { isUndoable, kindOf, reduceAll } from "@/arc/actions";
import { type Action, scopedActionZ } from "@/arc/actions.gen";
import { type Arc, type Key, keyZ } from "@/arc/types.gen";
import { type cache } from "@/cache";
import { dispatch } from "@/dispatch";

export const SET_CHANNEL_NAME = "sy_arc_set";
export const DELETE_CHANNEL_NAME = "sy_arc_delete";

export const STORE_KEY = "arcs";

/** Registers the arc table and its dispatch controller on the cache. */
export const bindStore = (
  engine: cache.Cache,
): dispatch.Controller<Key, Arc, Action> => {
  const table = () => engine.table<Key, Arc>(STORE_KEY);
  const deleteListener: cache.ChannelListener<{}, typeof keyZ> = {
    channel: DELETE_CHANNEL_NAME,
    schema: keyZ,
    onChange: ({ changed }) => table().delete(changed),
  };
  engine.registerTable<Key, Arc>(STORE_KEY, { listeners: [deleteListener] });
  const controller = new dispatch.Controller<Key, Arc, Action>({
    store: table(),
    onError: engine.onError,
    reduce: reduceAll,
    isUndoable,
    kindOf,
  });
  engine.addListeners(STORE_KEY, controller.listener(SET_CHANNEL_NAME, scopedActionZ));
  return controller;
};
