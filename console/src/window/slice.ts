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
    let s = state[action.payload.windowKey];
    if (s == null) {
      s = schema.parse({});
      state[action.payload.windowKey] = s;
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
