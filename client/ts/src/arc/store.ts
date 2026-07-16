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

/** Registers the arc store and its dispatch controller on the engine. */
export const bindStore = (
  engine: cache.Engine,
): dispatch.Controller<Key, Arc, Action> => {
  const store = () => engine.store<Key, Arc>(STORE_KEY);
  const deleteListener: cache.ChannelListener<{}, typeof keyZ> = {
    channel: DELETE_CHANNEL_NAME,
    schema: keyZ,
    onChange: ({ changed }) => store().delete(changed),
  };
  engine.registerStore<Key, Arc>(STORE_KEY, { listeners: [deleteListener] });
  const controller = new dispatch.Controller<Key, Arc, Action>({
    store: engine.unscoped(STORE_KEY),
    handleError: engine.errorHandler,
    reduce: reduceAll,
    isUndoable,
    kindOf,
  });
  engine.addListeners(STORE_KEY, controller.listener(SET_CHANNEL_NAME, scopedActionZ));
  return controller;
};
