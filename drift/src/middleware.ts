// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type Action as CoreAction,
  type configureStore,
  type Dispatch,
  type Middleware,
  type PayloadAction,
  type Tuple,
  type UnknownAction,
} from "@reduxjs/toolkit";
import { Mutex } from "async-mutex";

import { log } from "@/debug";
import { type Runtime } from "@/runtime";
import {
  type Action,
  assignLabel,
  isDriftAction,
  type LabelPayload,
  type MaybeKeyPayload,
  reloadWindow,
  runtimeSetWindowProps,
  setWindowError,
  shouldEmit,
  type SliceState,
  type StoreState,
} from "@/state";
import { desugar } from "@/sugar";
import { sync } from "@/sync";
import { validateAction } from "@/validate";

export type Middlewares<S> = ReadonlyArray<Middleware<{}, S>>;

// Used to ensure two things:
// 1. Only one set of window update operations is applied at a time, and they are applied
//    in the correct order.
// 2. Ensure that we emit actions to other windows in the correct order i.e. after synchronized
//    window operations have been applied.
const mu = new Mutex();

const EXCLUDE_SYNC_ACTIONS: string[] = [runtimeSetWindowProps.type, reloadWindow.type];

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
  <S extends StoreState, A extends CoreAction = UnknownAction>(
    runtime: Runtime<S, A | Action>,
    debug: boolean = false,
  ): Middleware<Dispatch<A | Action>, S, Dispatch<A | Action>> =>
  (store) =>
  (next) =>
  (action_) => {
    // eslint-disable-next-line prefer-const
    let { action, emitted, emitter } = desugar<A | Action>(action_ as A | Action);

    const label = runtime.label();

    validateAction({ action: action_ as A | Action, emitted, emitter });

    const isDrift = isDriftAction(action.type);
    if (isDrift)
      log(debug, "[drift] - middleware", {
        action,
        emitted,
        emitter,
        host: label,
      });

    // The action is recirculating from our own relay.
    if (emitter === runtime.label()) return;

    // If the runtime is updating its own props, no need to sync.
    const shouldSync = isDrift && !EXCLUDE_SYNC_ACTIONS.includes(action.type);

    let prevS: SliceState | null = null;
    if (isDrift) {
      prevS = store.getState().drift;
      action = assignLabel(
        action as PayloadAction<MaybeKeyPayload | LabelPayload>,
        prevS,
      );
    }

    const res = next(action);

    const nextS = shouldSync ? store.getState().drift : null;

    const shouldEmit_ = shouldEmit(emitted, action.type);

    // Run everything within a mutex locked closure to ensure that we correctly sync
    // and then propagate actions to other windows.
    void mu.runExclusive(async (): Promise<void> => {
      try {
        if (prevS !== null && nextS !== null) await sync(prevS, nextS, runtime, debug);
        if (shouldEmit_) await runtime.emit({ action });
      } catch (err) {
        log(debug, "[drift] - ERROR", {
          error: (err as Error).message,
          action,
          emitted,
          emitter,
          host: label,
        });
        store.dispatch(setWindowError({ key: label, message: (err as Error).message }));
      }
    });

    return res;
  };

/**
 * Configures the Redux middleware for the current window's store.
 *
 * @param mw - Middleware provided by the drift user (if any).
 * @param runtime - The runtime of the current window.
 * @returns a middleware function to be passed to `configureStore`.
 */
export const configureMiddleware =
  <
    S extends StoreState,
    A extends CoreAction = UnknownAction,
    M extends Middlewares<S> = Middlewares<S>,
  >(
    mw: M | ((def: GetDefaultMiddleware<S>) => M) | undefined,
    runtime: Runtime<S, A | Action>,
    debug: boolean = false,
  ): ((def: GetDefaultMiddleware<S>) => M) =>
  (def) => {
    const base = mw != null ? (typeof mw === "function" ? mw(def) : mw) : def();
    return [middleware<S, A>(runtime, debug), ...base] as unknown as M;
  };

type ConfigureStoreOptions<
  S extends StoreState,
  A extends CoreAction = UnknownAction,
  M extends Tuple<Middlewares<S>> = Tuple<Middlewares<S>>,
> = Parameters<typeof configureStore<S, A, M>>[0];

type MW<
  S extends StoreState,
  A extends CoreAction = UnknownAction,
  M extends Tuple<Middlewares<S>> = Tuple<Middlewares<S>>,
> = NonNullable<ConfigureStoreOptions<S, A, M>["middleware"]>;
export type GetDefaultMiddleware<S extends StoreState> = Parameters<MW<S>>[0];
