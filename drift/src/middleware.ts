// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type Action as BaseAction,
  type configureStore,
  type Dispatch,
  type Middleware,
  type PayloadAction,
  type Tuple,
  type UnknownAction,
} from "@reduxjs/toolkit";
import { errors, sync as xsync } from "@synnaxlabs/x";

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

const EXCLUDE_SYNC_ACTIONS: string[] = [runtimeSetWindowProps.type, reloadWindow.type];

/**
 * Redux middleware that conditionally does two things:
 *
 *      1. Emit actions to other windows.
 *      2. Execute window lifecycle actions.
 *
 * @param runtime - The runtime of the current application window.
 * @param mu - Applies window operations one set at a time, in order, and holds each
 * emission to other windows until its window operations are applied.
 * @returns a Redux middleware.
 */
const middleware =
  <S extends StoreState, A extends BaseAction = UnknownAction>(
    runtime: Runtime<S, A | Action>,
    mu: xsync.Mutex<{}>,
    debug: boolean = false,
  ): Middleware<Dispatch<A | Action>, S, Dispatch<A | Action>> =>
  (store) =>
  (next) =>
  (action_) => {
    const incoming = action_ as A | Action;
    // eslint-disable-next-line prefer-const
    let { action, emitted, emitter } = desugar<A | Action>(incoming);

    const label = runtime.label();

    validateAction(incoming);

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

    const prevS: SliceState = store.getState().drift;
    if (isDrift)
      action = assignLabel(
        action as PayloadAction<MaybeKeyPayload | LabelPayload>,
        prevS,
      );

    const res = next(action);

    const nextS: SliceState = store.getState().drift;

    // Any action that moves the slice reaches the runtime, not just drift's own: a
    // host restoring persisted window state relies on the same reconciliation. If the
    // runtime is updating its own props, no need to sync.
    const shouldSync = prevS !== nextS && !EXCLUDE_SYNC_ACTIONS.includes(action.type);

    // A middleware that swallows the action (next never ran) vetoes it for every
    // window: broadcasting it would apply it remotely but not locally.
    const shouldEmit_ = shouldEmit(emitted, action.type) && res != null;

    // Run everything within a mutex locked closure to ensure that we correctly sync
    // and then propagate actions to other windows.
    void mu.runExclusive(async (): Promise<void> => {
      try {
        if (shouldSync) await sync(prevS, nextS, store.dispatch, runtime, debug);
        if (shouldEmit_) await runtime.emit({ action });
      } catch (err) {
        const e = errors.fromUnknown(err);
        log(debug, "[drift] - ERROR", {
          error: e.message,
          action,
          emitted,
          emitter,
          host: label,
        });
        store.dispatch(setWindowError({ key: label, message: e.message }));
      }
    });

    return res;
  };

/**
 * Configures the Redux middleware for the current window's store.
 * @param mw - Middleware provided by the drift user (if any).
 * @returns a middleware function to be passed to `configureStore`.
 */
export const configureMiddleware =
  <
    S extends StoreState,
    A extends BaseAction = UnknownAction,
    M extends Middlewares<S> = Middlewares<S>,
  >(
    mw: M | ((def: GetDefaultMiddleware<S>) => M) | undefined,
    runtime: Runtime<S, A | Action>,
    debug: boolean = false,
  ): ((def: GetDefaultMiddleware<S>) => M) =>
  (def) => {
    const base = mw != null ? (typeof mw === "function" ? mw(def) : mw) : def();
    const mu = xsync.createMutex({});
    return [middleware<S, A>(runtime, mu, debug), ...base] as unknown as M;
  };

type ConfigureStoreOptions<
  S extends StoreState,
  A extends BaseAction = UnknownAction,
  M extends Tuple<Middlewares<S>> = Tuple<Middlewares<S>>,
> = Parameters<typeof configureStore<S, A, M>>[0];

type MW<
  S extends StoreState,
  A extends BaseAction = UnknownAction,
  M extends Tuple<Middlewares<S>> = Tuple<Middlewares<S>>,
> = NonNullable<ConfigureStoreOptions<S, A, M>["middleware"]>;
export type GetDefaultMiddleware<S extends StoreState> = Parameters<MW<S>>[0];
