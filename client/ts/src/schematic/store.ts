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
import { kindOf, reduceAll } from "@/schematic/actions";
import { type Action, scopedActionZ } from "@/schematic/actions.gen";
import { type Key, keyZ, type Schematic } from "@/schematic/types.gen";

export const SET_CHANNEL_NAME = "sy_schematic_set";
export const DELETE_CHANNEL_NAME = "sy_schematic_delete";

export const STORE_KEY = "schematics";

/** Registers the schematic table and its dispatch controller on the cache. */
export const bindStore = (
  engine: cache.Cache,
): dispatch.Controller<Key, Schematic, Action> => {
  const table = () => engine.table<Key, Schematic>(STORE_KEY);
  const deleteListener: cache.ChannelListener<{}, typeof keyZ> = {
    channel: DELETE_CHANNEL_NAME,
    schema: keyZ,
    onChange: ({ changed }) => table().delete(changed),
  };
  engine.registerTable<Key, Schematic>(STORE_KEY, { listeners: [deleteListener] });
  const controller = new dispatch.Controller<Key, Schematic, Action>({
    store: table(),
    onError: engine.onError,
    reduce: reduceAll,
    kindOf,
  });
  engine.addListeners(STORE_KEY, controller.listener(SET_CHANNEL_NAME, scopedActionZ));
  return controller;
};
