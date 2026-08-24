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

/**
 * Base payload for window-scoped actions. When windowKey is omitted, the inject-key
 * middleware fills it with the dispatching window's key.
 */
export interface OptionalKeyParams {
  windowKey?: string;
}

type RequireWindowKey<Payload extends OptionalKeyParams> = require.Require<
  Payload,
  "windowKey"
>;

/**
 * Builds a reducer wrapper that scopes a handler to a single window's state. The
 * returned wrapper resolves (and lazily creates from schema defaults) the per-window
 * state for the action's windowKey, then invokes handler with it.
 * @param schema the zod schema used to default a window's state on first access.
 * @throws {UnexpectedError} if the dispatched action has no windowKey, which should
 * already have been injected by createInjectKeyMiddleware.
 */
export const createWithKeyHandler =
  <State extends z.ZodType>(schema: State) =>
  <
    Payload extends OptionalKeyParams,
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

/** Per-window view state for a set of documents, keyed by document key. */
export type Documents<Doc> = Record<string, Doc>;

export interface DocumentParams extends OptionalKeyParams {
  key: string;
}

interface DocumentsState<Doc> {
  windows: Record<string, Documents<Doc>>;
}

/**
 * Builds a reducer wrapper that scopes a handler to one document's view state in one
 * window. A window is a viewport, so two windows showing one document each get their
 * own viewport, selection, and toolbar; the document itself lives on the Core.
 * @param schema the zod schema used to default a document's state on first access.
 * @throws {UnexpectedError} if the dispatched action has no windowKey, which should
 * already have been injected by createInjectKeyMiddleware.
 */
export const createWithDocumentHandler =
  <Doc extends z.ZodType>(schema: Doc) =>
  <
    Payload extends DocumentParams,
    SliceState extends DocumentsState<z.output<Doc>>,
    Type extends string = string,
  >(
    handler?: (
      state: z.output<Doc>,
      action: PayloadAction<RequireWindowKey<Payload>, Type>,
    ) => void,
  ): ((state: SliceState, action: PayloadAction<Payload, Type>) => void) =>
  (state, action) => {
    const { windowKey, key } = action.payload;
    if (windowKey == null)
      throw new UnexpectedError(
        `expected windowKey to be defined in ${action.type} payload`,
      );
    const win = (state.windows[windowKey] ??= {});
    win[key] ??= schema.parse({});
    handler?.(win[key], action as PayloadAction<RequireWindowKey<Payload>, Type>);
  };

/**
 * Builds the reducer for a document's create action: it starts this window's view from
 * the payload, and leaves an existing view alone so reopening a document does not
 * discard where the user had it.
 * @param schema the zod schema the initial state is parsed through.
 * @throws {UnexpectedError} if the dispatched action has no windowKey.
 */
export const createDocumentInitializer =
  <Doc extends z.ZodType>(schema: Doc) =>
  <Payload extends DocumentParams, SliceState extends DocumentsState<z.output<Doc>>>(
    state: SliceState,
    { payload, type }: PayloadAction<Payload>,
  ): void => {
    const { windowKey, key } = payload;
    if (windowKey == null)
      throw new UnexpectedError(`expected windowKey to be defined in ${type} payload`);
    const win = (state.windows[windowKey] ??= {});
    if (key in win) return;
    win[key] = schema.parse(payload);
  };

/** This window's view of the given document, if it has one. */
export const selectDocument = <Doc>(
  root: Drift.StoreState,
  slice: DocumentsState<Doc>,
  key: string,
): Doc | undefined => {
  const windowKey = Drift.selectWindowKey(root);
  if (windowKey == null) return undefined;
  return slice.windows[windowKey]?.[key];
};

/** Every document key any window holds view state for. */
export const documentKeys = <Doc>(state: DocumentsState<Doc>): string[] => [
  ...new Set(Object.values(state.windows).flatMap((win) => Object.keys(win))),
];

/** Drops the given documents from every window. */
export const removeDocuments = <Doc>(
  state: DocumentsState<Doc>,
  keys: string[],
): void => {
  Object.values(state.windows).forEach((win) => keys.forEach((key) => delete win[key]));
};

/** Runs purge over every window's copy of every document. */
export const purgeDocuments = <Doc>(
  state: DocumentsState<Doc>,
  purge: (doc: Doc) => void,
): void =>
  Object.values(state.windows).forEach((win) => Object.values(win).forEach(purge));

type KeyActionMatcher = Pick<ActionCreatorWithPayload<OptionalKeyParams>, "match">;

/**
 * Creates Redux middleware that stamps the dispatching window's key onto matching
 * actions that were dispatched without an explicit windowKey. Actions that already
 * carry a windowKey pass through unchanged; matching actions dispatched before a window
 * key is available are dropped.
 * @param actionCreators the action creator(s) whose payloads should be key-injected.
 */
export const createInjectKeyMiddleware = <StoreState>(
  actionCreators: KeyActionMatcher | KeyActionMatcher[],
): Middleware<{}, Drift.StoreState & StoreState> => {
  const creators = Array.isArray(actionCreators) ? actionCreators : [actionCreators];
  const matches = (action: unknown): action is PayloadAction<OptionalKeyParams> =>
    creators.some((creator) => creator.match(action));
  return (store) => (next) => (action) => {
    if (!matches(action) || action.payload.windowKey != null) return next(action);
    const windowKey = Drift.selectWindowKey(store.getState());
    if (windowKey == null) return;
    action.payload.windowKey = windowKey;
    return next(action);
  };
};
