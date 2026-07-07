// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createSlice, type Dispatch, type PayloadAction } from "@reduxjs/toolkit";
import { panel } from "@synnaxlabs/client";
import { Panel } from "@synnaxlabs/pluto";
import { type require } from "@synnaxlabs/x";
import { useCallback } from "react";
import { useDispatch } from "react-redux";
import z from "zod";

import { Window } from "@/session/window";

export const stateZ = z.object({
  selectedTabs: panel.tabKeyZ.array().default([]),
});

export interface State extends z.output<typeof stateZ> {}

export const ZERO_STATE = stateZ.parse({});

export const windowStateZ = z.object({
  selected: panel.keyZ.optional(),
  isOverlaid: z.boolean().optional().default(false),
  panels: z.record(panel.keyZ, stateZ).default({}),
});

export interface WindowState extends z.output<typeof windowStateZ> {}

export const ZERO_WINDOW_STATE: WindowState = windowStateZ.parse({});

export const sliceStateZ = z.object({
  windows: z.record(z.string(), windowStateZ).default({}),
});

export interface SliceState extends z.output<typeof sliceStateZ> {}

export interface StoreState {
  [SLICE_NAME]: SliceState;
}

export const SLICE_NAME = "panels";

export const ZERO_SLICE_STATE: SliceState = sliceStateZ.parse({});

export interface PanelKeyPayload extends Window.OptionalKeyParams {
  key: panel.Key;
}

export interface TabKeyPayload extends Window.OptionalKeyParams {
  tabKey: panel.TabKey;
}

export interface SetSelectedTabsPayload extends PanelKeyPayload {
  selectedTabs: panel.TabKey[];
}

export interface TabAndPanelKeyPayload extends PanelKeyPayload {
  tabKey: string;
}

interface SelectTabPayload extends TabAndPanelKeyPayload {
  otherTabKeys: panel.TabKey[];
}

const withWindowKey = Window.createWithKeyHandler(windowStateZ);

const withSelectedState = <Payload extends PanelKeyPayload>(
  handler: (
    state: State,
    action: PayloadAction<require.Require<Payload, "windowKey">>,
  ) => void,
) =>
  withWindowKey<Payload, SliceState>((win, action) => {
    const { key } = action.payload;
    let pan = win.panels[key];
    if (pan == null) {
      pan = stateZ.parse({});
      win.panels[key] = pan;
    }
    handler(pan, action);
  });

const { actions, reducer } = createSlice({
  name: SLICE_NAME,
  initialState: ZERO_SLICE_STATE,
  reducers: {
    select: withWindowKey<PanelKeyPayload, SliceState>((win, { payload: { key } }) => {
      win.selected = key;
    }),
    clearSelected: withWindowKey<Window.OptionalKeyParams, SliceState>((win) => {
      win.selected = undefined;
    }),
    remove: withWindowKey<PanelKeyPayload, SliceState>((win, { payload: { key } }) => {
      delete win.panels[key];
      if (Object.keys(win.panels).length > 0) return;
      win.selected = undefined;
    }),
    selectTab: withSelectedState<SelectTabPayload>(
      (pan, { payload: { tabKey, otherTabKeys } }) => {
        pan.selectedTabs = [
          tabKey,
          ...pan.selectedTabs.filter((k) => !otherTabKeys.includes(k)),
        ];
      },
    ),
    startOverlaying: withWindowKey<Window.OptionalKeyParams, SliceState>((win) => {
      win.isOverlaid = true;
    }),
    stopOverlaying: withWindowKey<Window.OptionalKeyParams, SliceState>((win) => {
      win.isOverlaid = false;
    }),
    reset: () => ZERO_SLICE_STATE,
  },
});

const {
  select,
  clearSelected,
  remove,
  selectTab: internalSelectTab,
  startOverlaying,
  stopOverlaying,
  reset,
} = actions;

export {
  clearSelected,
  reducer,
  remove,
  reset,
  select,
  startOverlaying,
  stopOverlaying,
};

export type Action = ReturnType<(typeof actions)[keyof typeof actions]>;
export type Payload = Action["payload"];

export const MIDDLEWARE = [
  Window.createInjectKeyMiddleware([
    select,
    clearSelected,
    remove,
    internalSelectTab,
    startOverlaying,
    stopOverlaying,
  ]),
];

export const PERSIST_EXCLUDE = [];

export const useSelectTab = (panelKey?: panel.Key) => {
  const resolvedPanelKey = Panel.useOptionalKey(panelKey);
  const getTabLeaf = Panel.useGetTabLeaf();
  const dispatch = useDispatch<Dispatch<Action>>();
  return useCallback(
    (key: panel.TabKey) => {
      if (resolvedPanelKey == null) return;
      const leaf = getTabLeaf({ tabKey: key });
      dispatch(
        internalSelectTab({
          tabKey: key,
          key: resolvedPanelKey,
          otherTabKeys: leaf.tabs.map((t) => t.key),
        }),
      );
    },
    [panelKey, dispatch],
  );
};
