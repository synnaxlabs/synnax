// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type cache } from "@/cache";
import { dispatch } from "@/dispatch";
import { kindOf, reduceAll } from "@/table/actions";
import { type Action, scopedActionZ } from "@/table/actions.gen";
import { type Key, keyZ, type Table } from "@/table/types.gen";

export const SET_CHANNEL_NAME = "sy_table_set";
export const DELETE_CHANNEL_NAME = "sy_table_delete";

export const STORE_KEY = "tables";

/** Registers the table's cache table and its dispatch controller. */
export const bindStore = (
  engine: cache.Cache,
): dispatch.Controller<Key, Table, Action> => {
  const table = () => engine.table<Key, Table>(STORE_KEY);
  const deleteListener: cache.ChannelListener<{}, typeof keyZ> = {
    channel: DELETE_CHANNEL_NAME,
    schema: keyZ,
    onChange: ({ changed }) => table().delete(changed),
  };
  engine.registerTable<Key, Table>(STORE_KEY, { listeners: [deleteListener] });
  const controller = new dispatch.Controller<Key, Table, Action>({
    store: table(),
    onError: engine.onError,
    reduce: reduceAll,
    kindOf,
  });
  engine.addListeners(STORE_KEY, controller.listener(SET_CHANNEL_NAME, scopedActionZ));
  return controller;
};
