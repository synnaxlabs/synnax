// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type cache, type dispatch } from "@synnaxlabs/client";
import { type record } from "@synnaxlabs/x";

/**
 * A UnaryStore augmented with the action-dispatch surface for undoable
 * documents. The store handle comes from the cache engine; the dispatch
 * handle from the domain's controller.
 */
export type UndoableUnaryStore<
  Key extends record.Key,
  State extends cache.Data,
  Action,
> = cache.UnaryStore<Key, State> & dispatch.Handle<Key, State, Action>;

export type DispatchReducer<State extends cache.Data, Action> = dispatch.Reducer<
  State,
  Action
>;
export type ReplayResult<Action> = dispatch.ReplayResult<Action>;
export type Reversal<Action> = dispatch.Reversal<Action>;
export type Transaction<Action> = dispatch.Transaction<Action>;
export type DispatchSend<Action> = dispatch.Send<Action>;
export type DispatchFrame<Key, Action> = dispatch.Frame<Key, Action>;
export type StackEntry<Action> = dispatch.StackEntry<Action>;
