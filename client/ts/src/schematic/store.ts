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

/** Registers the schematic store and its dispatch controller on the engine. */
export const bindStore = (
  engine: cache.Engine,
): dispatch.Controller<Key, Schematic, Action> => {
  const store = () => engine.store<Key, Schematic>(STORE_KEY);
  const deleteListener: cache.ChannelListener<{}, typeof keyZ> = {
    channel: DELETE_CHANNEL_NAME,
    schema: keyZ,
    onChange: ({ changed }) => store().delete(changed),
  };
  engine.registerStore<Key, Schematic>(STORE_KEY, { listeners: [deleteListener] });
  const controller = new dispatch.Controller<Key, Schematic, Action>({
    store: engine.unscoped(STORE_KEY),
    handleError: engine.errorHandler,
    reduce: reduceAll,
    kindOf,
  });
  engine.addListeners(STORE_KEY, controller.listener(SET_CHANNEL_NAME, scopedActionZ));
  return controller;
};
