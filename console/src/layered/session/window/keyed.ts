// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type ActionCreatorWithPayload,
  type Middleware,
  type PayloadAction,
} from "@reduxjs/toolkit";
import { UnexpectedError } from "@synnaxlabs/client";
import { Drift } from "@synnaxlabs/drift";
import { type require } from "@synnaxlabs/x";
import type z from "zod";

/**
 * Base payload for window-scoped actions. When windowKey is omitted, the inject-key
 * middleware fills it with the dispatching window's key.
 */
export interface OptionalKeyParams {
  windowKey?: string;
}

type RequireWindowKey<Payload extends OptionalKeyParams> = require.Require<
  Payload,
  "windowKey"
>;

/**
 * Builds a reducer wrapper that scopes a handler to a single window's state. The
 * returned wrapper resolves (and lazily creates from schema defaults) the per-window
 * state for the action's windowKey, then invokes handler with it.
 *
 * @param schema the zod schema used to default a window's state on first access.
 * @throws {UnexpectedError} if the dispatched action has no windowKey, which should
 * already have been injected by createInjectKeyMiddleware.
 */
export const createWithKeyHandler =
  <State extends z.ZodType>(schema: State) =>
  <
    Payload extends OptionalKeyParams,
    SliceState extends {
      windows: Record<string, z.output<State>>;
    },
    Type extends string = string,
  >(
    handler: (
      state: z.output<State>,
      action: PayloadAction<RequireWindowKey<Payload>, Type>,
    ) => void,
  ): ((state: SliceState, action: PayloadAction<Payload, Type>) => void) =>
  (state, action) => {
    if (action.payload.windowKey == null)
      throw new UnexpectedError(
        `expected windowKey to be defined in ${action.type} payload`,
      );
    let s = state.windows[action.payload.windowKey];
    if (s == null) {
      s = schema.parse({});
      state.windows[action.payload.windowKey] = s;
    }
    handler(s, action as PayloadAction<RequireWindowKey<Payload>, Type>);
  };

type KeyActionMatcher = Pick<ActionCreatorWithPayload<OptionalKeyParams>, "match">;

/**
 * Creates Redux middleware that stamps the dispatching window's key onto matching
 * actions that were dispatched without an explicit windowKey. Actions that already
 * carry a windowKey pass through unchanged; matching actions dispatched before a window
 * key is available are dropped.
 *
 * @param actionCreators the action creator(s) whose payloads should be key-injected.
 */
export const createInjectKeyMiddleware = <StoreState>(
  actionCreators: KeyActionMatcher | KeyActionMatcher[],
): Middleware<{}, Drift.StoreState & StoreState> => {
  const creators = Array.isArray(actionCreators) ? actionCreators : [actionCreators];
  return (store) => (next) => (action) => {
    const payload = (action as PayloadAction<OptionalKeyParams>).payload;
    if (!creators.some((creator) => creator.match(action)) || payload.windowKey != null)
      return next(action);
    const windowKey = Drift.selectWindowKey(store.getState());
    if (windowKey == null) return;
    payload.windowKey = windowKey;
    return next(action);
  };
};
