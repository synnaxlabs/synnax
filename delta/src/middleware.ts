// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { PayloadAction, Middleware, Dispatch } from "@reduxjs/toolkit";

export interface MiddlewareEffectArgs<S, LP extends any, DP extends any> {
  getState: () => S;
  dispatch: Dispatch<PayloadAction<DP>>;
  action: PayloadAction<LP>;
}

export type MiddlewareEffect<S, LP extends any, DP extends any = LP> = (
  args: MiddlewareEffectArgs<S, LP, DP>
) => void;

export const dispatchEffect =
  <S, I, O extends any>(
    factory: (payload: I) => PayloadAction<O>
  ): MiddlewareEffect<S, I, O> =>
  ({ dispatch, action }) =>
    dispatch(factory(action.payload));

export const effectMiddleware =
  <S, LP extends any, DP extends any>(
    deps: string[],
    effects: Array<MiddlewareEffect<S, LP, DP>>
  ): Middleware<Record<string, never>, S, Dispatch<PayloadAction<LP>>> =>
  ({ getState, dispatch }) =>
  (next) =>
  (action) => {
    const state = next(action);
    if (deps.includes(action.type))
      effects.forEach((factory) => {
        factory({ getState, dispatch, action });
      });
    return state;
  };
