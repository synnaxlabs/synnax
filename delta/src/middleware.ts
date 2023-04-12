// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type {
  PayloadAction,
  Middleware,
  Dispatch,
  ActionCreatorWithPayload,
} from "@reduxjs/toolkit";

export interface MiddlewareEffectArgs<S, P extends any> {
  getState: () => S;
  dispatch: Dispatch<PayloadAction<P>>;
  action: PayloadAction<P>;
}

export type MiddlewareEffect<S, P extends any> = (
  args: MiddlewareEffectArgs<S, P>
) => void;

export const dispatchEffect =
  <S, I, O extends any>(factory: ActionCreatorWithPayload<O>): MiddlewareEffect<S, I> =>
  ({ dispatch, action }) =>
    dispatch(factory(action.payload as unknown as O) as unknown as PayloadAction<I>);

export const effectMiddleware =
  <S, P extends any>(
    deps: string[],
    effects: Array<MiddlewareEffect<S, P>>
  ): Middleware<Record<string, never>, S, Dispatch<PayloadAction<P>>> =>
  ({ getState, dispatch }) =>
  (next) =>
  (action) => {
    const state = next(action);
    if (deps.includes(action.type))
      effects.forEach((factory) => factory({ getState, dispatch, action }));
    return state;
  };
