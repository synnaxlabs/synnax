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

export interface OptionalKeyPayload {
  windowKey?: string;
}

type RequireWindowKey<Payload extends OptionalKeyPayload> = require.Require<
  Payload,
  "windowKey"
>;

export const createWithKeyHandler =
  <State extends z.ZodType>(schema: State) =>
  <
    Payload extends OptionalKeyPayload,
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

export const createInjectKeyMiddleware =
  <StoreState, P extends { windowKey?: string }, T extends string = string>(
    actionCreator: ActionCreatorWithPayload<P, T>,
  ): Middleware<{}, Drift.StoreState & StoreState> =>
  (store) =>
  (next) =>
  (action) => {
    if (!actionCreator.match(action) || action.payload.windowKey != null)
      return next(action);
    const windowKey = Drift.selectWindowKey(store.getState());
    if (windowKey == null) return;
    action.payload.windowKey = windowKey;
    return next(action);
  };
