import { Action, AnyAction, Dispatch, Middleware } from "@reduxjs/toolkit";
import { CurriedGetDefaultMiddleware } from "@reduxjs/toolkit/dist/getDefaultMiddleware";

import { Runtime } from "./runtime";
import {
  StoreState,
  assignKey,
  executeAction,
  isDrift,
  shouldEmit,
} from "./state";
import { desugar } from "./sugar";

// eslint-disable-next-line @typescript-eslint/ban-types
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
export const middleware = <S extends StoreState, A extends Action = AnyAction>(
  runtime: Runtime<S, A>
): Middleware<Record<string, never>, S, Dispatch<A>> => {
  return ({ getState }) =>
    (next) =>
    (action_) => {
      // eslint-disable-next-line prefer-const
      let { action, emitted, emitter } = desugar(action_);

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
    const base = mw ? (typeof mw === "function" ? mw(def) : mw) : def();
    return [...base, middleware<S, A>(runtime)] as unknown as M;
  };
};
