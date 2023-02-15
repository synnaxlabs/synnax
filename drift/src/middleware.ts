// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { Action, AnyAction, Dispatch, Middleware } from "@reduxjs/toolkit";
import type { CurriedGetDefaultMiddleware } from "@reduxjs/toolkit/dist/getDefaultMiddleware";

import { StoreState, assignKey, executeAction, isDrift, shouldEmit } from "./state";
import { desugar } from "./sugar";

import { Runtime } from "@/runtime";

import { validateAction } from "./validate";

export type Middlewares<S> = ReadonlyArray<Middleware<{}, S>>;

/**
 * Redux middleware that conditionally does two things:
 *
 *      1. Emit actions to other windows.
 *      2. Execute window lifecycle actions.
 *
 * @param runtime - The runtime of the current application window.
 * @returns a Redux middleware.
 */
export const middleware =
  <S extends StoreState, A extends Action = AnyAction>(
    runtime: Runtime<S, A>
  ): Middleware<Record<string, never>, S, Dispatch<A>> =>
  ({ getState }) =>
  (next) =>
  (action_) => {
    // eslint-disable-next-line prefer-const
    let { action, emitted, emitter } = desugar(action_);

    validateAction({ action: action_, emitted, emitter });

    // The action is recirculating from our own relay.
    if (emitter === runtime.key()) return;

    if (isDrift(action.type)) {
      const state = getState();
      if (!emitted) action.payload.key = assignKey(runtime, action, state);
      if (runtime.isMain()) executeAction(runtime, action, state);
    }

    const res = next(action);

    if (shouldEmit(emitted, action.type)) runtime.emit({ action });

    return res;
  };

/**
 * Configures the Redux middleware for the curent window's store.
 *
 * @param mw - Middleware provided by the drift user (if any).
 * @param runtime - The runtime of the current window.
 * @returns a middleware function to be passed to `configureStore`.
 */
export const configureMiddleware = <
  S extends StoreState,
  A extends Action = AnyAction,
  M extends Middlewares<S> = Middlewares<S>
>(
  mw: M | ((def: CurriedGetDefaultMiddleware<S>) => M) | undefined,
  runtime: Runtime<S, A>
): ((def: CurriedGetDefaultMiddleware<S>) => M) => {
  return (def) => {
    const base = mw != null ? (typeof mw === "function" ? mw(def) : mw) : def();
    return [...base, middleware<S, A>(runtime)] as unknown as M;
  };
};
