import {
  type ActionCreatorWithPayload,
  type Middleware,
  type PayloadAction,
} from "@reduxjs/toolkit";
import { UnexpectedError } from "@synnaxlabs/client";
import { Drift } from "@synnaxlabs/drift";
import { type require } from "@synnaxlabs/x";

export interface OptionalWindowKeyPayload {
  windowKey?: string;
}

type RequireWindowKey<Payload extends OptionalWindowKeyPayload> = require.Require<
  Payload,
  "windowKey"
>;

export const withWindowKey =
  <
    Payload extends OptionalWindowKeyPayload,
    SliceState extends {},
    Type extends string = string,
  >(
    handler: (
      state: SliceState,
      action: PayloadAction<RequireWindowKey<Payload>, Type>,
    ) => void,
  ): ((state: SliceState, action: PayloadAction<Payload, Type>) => void) =>
  (state, action) => {
    if (action.payload.windowKey == null)
      throw new UnexpectedError(
        `expected windowKey to be defined in ${action.type} payload`,
      );
    handler(state, action as PayloadAction<RequireWindowKey<Payload>, Type>);
  };

export const createInjectWindowKeyMiddleware =
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
