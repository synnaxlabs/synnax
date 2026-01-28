// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type {
  Dispatch,
  Middleware,
  MiddlewareAPI,
  PayloadAction,
} from "@reduxjs/toolkit";

export interface MiddlewareEffectArgs<S, LP, DP> {
  store: MiddlewareAPI<Dispatch<{ payload: LP | DP; type: string }>, S>;
  action: PayloadAction<LP>;
}

export type MiddlewareEffect<S, LP, DP = LP> = (
  args: MiddlewareEffectArgs<S, LP, DP>,
) => void;

export const dispatchEffect =
  <S, I, O>(factory: (payload: I) => PayloadAction<O>): MiddlewareEffect<S, I, O> =>
  ({ store, action }) =>
    store.dispatch(factory(action.payload));

export const effectMiddleware =
  <S, LP, DP>(
    deps: string[],
    effects: Array<MiddlewareEffect<S, LP | DP, DP | LP>>,
    before: boolean = false,
  ): Middleware<Dispatch<PayloadAction<LP>>, S, Dispatch<PayloadAction<LP>>> =>
  (store) =>
  (next) =>
  (unknownAction) => {
    const action = unknownAction as PayloadAction<LP>;
    if (before && deps.includes(action.type))
      effects.forEach((factory) => factory({ store, action }));
    const state = next(unknownAction);
    if (!before && deps.includes(action.type))
      effects.forEach((factory) => factory({ store, action }));
    return state;
  };
