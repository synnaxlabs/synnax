// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createSlice, type PayloadAction } from "@reduxjs/toolkit";
import { channel, type log, UnexpectedError } from "@synnaxlabs/client";
import { type Drift } from "@synnaxlabs/drift";
import { Log } from "@synnaxlabs/pluto";
import z from "zod";

import { Window } from "@/session/window";

export const toolbarTabZ = z.enum(["channels", "properties"]);
export type ToolbarTab = z.infer<typeof toolbarTabZ>;

const toolbarStateZ = z.object({
  selectedTab: toolbarTabZ.default("channels"),
});

const searchZ = Log.search.searchZ.extend({
  /** Shows only the lines that match. */
  filtered: z.boolean().default(false),
});
export interface SearchState extends z.infer<typeof searchZ> {}

const stateZ = z.object({
  hold: z.boolean().default(false),
  /** Where the log stood when it last unmounted. A remount reads it while on hold. */
  position: z.string().default(""),
  toolbar: toolbarStateZ.prefault({}),
  hiddenChannels: z.array(channel.keyZ).default([]),
  search: searchZ.prefault({}),
});
export interface State extends z.infer<typeof stateZ> {}
export interface NewState extends z.input<typeof stateZ> {}

export const ZERO_STATE = stateZ.parse({});

const windowStateZ = z.record(z.string(), stateZ).default({});

export const sliceStateZ = z.object({
  version: z.literal(0).default(0),
  /** A window is a viewport, so each holds its own view of every document. */
  windows: z.record(z.string(), windowStateZ).default({}),
});
export interface SliceState extends z.infer<typeof sliceStateZ> {}

export const ZERO_SLICE_STATE = sliceStateZ.parse({});

export const SLICE_NAME = "log";

export interface StoreState extends Drift.StoreState {
  [SLICE_NAME]: SliceState;
}

export interface KeyedPayload extends Window.OptionalKeyParams {
  key: log.Key;
}

export interface CreatePayload extends KeyedPayload, NewState {}

export interface SetActiveToolbarTabPayload extends KeyedPayload {
  tab: ToolbarTab;
}

export interface SetHoldPayload extends KeyedPayload {
  hold?: boolean;
}

export interface SetPositionPayload extends KeyedPayload {
  position: string;
}

export interface SetChannelVisiblePayload extends KeyedPayload {
  channel: channel.Key;
  visible: boolean;
}

export interface SetSearchPayload extends KeyedPayload, Partial<SearchState> {}

export interface RemovePayload {
  keys: string[];
}

const withSelectedState = Window.createWithDocumentHandler(stateZ);
const initializeDocument = Window.createDocumentInitializer(stateZ);

export const { actions, reducer } = createSlice({
  name: SLICE_NAME,
  initialState: ZERO_SLICE_STATE,
  reducers: {
    create: initializeDocument<CreatePayload, SliceState>,
    setSelectedToolbarTab: withSelectedState<SetActiveToolbarTabPayload, SliceState>(
      (state, { payload: { tab } }) => {
        state.toolbar.selectedTab = tab;
      },
    ),
    setHold: withSelectedState<SetHoldPayload, SliceState>(
      (state, { payload: { hold } }) => {
        state.hold = hold ?? !state.hold;
        if (!state.hold) state.position = "";
      },
    ),
    // A deleted log unmounts after its document goes, which must not bring it back. A
    // log on hold always has a document.
    setPosition: (state, { payload, type }: PayloadAction<SetPositionPayload>) => {
      const { windowKey, key, position } = payload;
      if (windowKey == null)
        throw new UnexpectedError(
          `expected windowKey to be defined in ${type} payload`,
        );
      const doc = state.windows[windowKey]?.[key];
      if (doc != null) doc.position = position;
    },
    setChannelVisible: withSelectedState<SetChannelVisiblePayload, SliceState>(
      (state, { payload: { channel, visible } }) => {
        const hidden = new Set(state.hiddenChannels);
        if (visible) hidden.delete(channel);
        else hidden.add(channel);
        state.hiddenChannels = Array.from(hidden);
      },
    ),
    setSearch: withSelectedState<SetSearchPayload, SliceState>(
      (state, { payload: { key: _key, windowKey: _windowKey, ...search } }) => {
        state.search = { ...state.search, ...search };
      },
    ),
    remove: (state, { payload }: PayloadAction<RemovePayload>) => {
      Window.removeDocuments(state, payload.keys);
    },
  },
  extraReducers: Window.handleRemoved,
});

export const {
  create,
  setSelectedToolbarTab,
  setHold,
  setPosition,
  setChannelVisible,
  setSearch,
  remove,
} = actions;

export type Action = ReturnType<(typeof actions)[keyof typeof actions]>;

export const MIDDLEWARE = [
  Window.createInjectKeyMiddleware([
    create,
    setSelectedToolbarTab,
    setHold,
    setPosition,
    setChannelVisible,
    setSearch,
  ]),
];
