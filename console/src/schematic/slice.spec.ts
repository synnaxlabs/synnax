// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { combineReducers, configureStore } from "@reduxjs/toolkit";
import { Drift, MAIN_WINDOW } from "@synnaxlabs/drift";
import { beforeEach, describe, expect, it } from "vitest";

import { Layout } from "@/layout";
import { MIDDLEWARE } from "@/schematic/middleware";
import {
  actions,
  purgeState,
  reducer,
  SLICE_NAME,
  type StoreState,
  ZERO_SLICE_STATE,
  ZERO_STATE,
} from "@/schematic/slice";

describe("Schematic Slice", () => {
  let store: ReturnType<typeof configureStore<StoreState>>;
  const layoutKey = "schematic-1";

  beforeEach(() => {
    store = configureStore({
      reducer: { [SLICE_NAME]: reducer },
      preloadedState: { [SLICE_NAME]: ZERO_SLICE_STATE },
    });
    store.dispatch(actions.create({ key: layoutKey }));
  });

  it("should seed a default UI shell on create", () => {
    const s = store.getState()[SLICE_NAME].schematics[layoutKey];
    expect(s).toBeDefined();
    expect(s.selected).toEqual([]);
    expect(s.toolbar.activeTab).toBe("symbols");
    expect(s.editable).toBe(false);
  });

  it("should honor an editable override on create", () => {
    const key = "schematic-2";
    store.dispatch(actions.create({ key, editable: true }));
    const s = store.getState()[SLICE_NAME].schematics[key];
    expect(s.editable).toBe(true);
  });

  it("should track selection and switch toolbar tab", () => {
    store.dispatch(actions.setSelected({ key: layoutKey, selected: ["n1"] }));
    let s = store.getState()[SLICE_NAME].schematics[layoutKey];
    expect(s.selected).toEqual(["n1"]);
    expect(s.toolbar.activeTab).toBe("properties");
    store.dispatch(actions.setSelected({ key: layoutKey, selected: [] }));
    s = store.getState()[SLICE_NAME].schematics[layoutKey];
    expect(s.toolbar.activeTab).toBe("symbols");
  });

  it("should toggle editable and clear selection on disable", () => {
    store.dispatch(actions.setEditable({ key: layoutKey, editable: true }));
    store.dispatch(actions.setSelected({ key: layoutKey, selected: ["n1"] }));
    store.dispatch(actions.setEditable({ key: layoutKey, editable: false }));
    const s = store.getState()[SLICE_NAME].schematics[layoutKey];
    expect(s.editable).toBe(false);
    expect(s.selected).toEqual([]);
  });

  it("should set legend visibility independently of position", () => {
    store.dispatch(actions.setLegendVisible({ key: layoutKey, visible: true }));
    const s = store.getState()[SLICE_NAME].schematics[layoutKey];
    expect(s.legend.visible).toBe(true);
    expect(s.legend.position).toBeDefined();
  });

  it("should clear pendingUpload", () => {
    store.dispatch(actions.clearPendingUpload({ key: layoutKey }));
    const s = store.getState()[SLICE_NAME].schematics[layoutKey];
    expect(s.pendingUpload).toBeUndefined();
  });

  it("should remove a schematic", () => {
    store.dispatch(actions.remove({ keys: [layoutKey] }));
    expect(store.getState()[SLICE_NAME].schematics[layoutKey]).toBeUndefined();
  });

  it("should reset transient fields on purge", () => {
    const state = structuredClone(ZERO_STATE);
    state.controlStatus = "acquired";
    state.selected = ["n1"];
    state.toolbar = { ...state.toolbar, activeTab: "properties" };
    purgeState(state);
    expect(state.controlStatus).toBe("released");
    expect(state.selected).toEqual([]);
    expect(state.toolbar.activeTab).toBe("symbols");
  });
});

describe("Schematic Middleware", () => {
  const layoutKey = "schematic-1";

  const buildStore = (
    layouts: Record<string, Layout.State>,
    activeTab: string | null,
  ) => {
    const store = configureStore({
      reducer: combineReducers({
        [SLICE_NAME]: reducer,
        [Layout.SLICE_NAME]: Layout.reducer,
        drift: Drift.reducer,
      }),
      preloadedState: {
        [SLICE_NAME]: ZERO_SLICE_STATE,
        [Layout.SLICE_NAME]: {
          ...Layout.ZERO_SLICE_STATE,
          layouts: { ...Layout.ZERO_SLICE_STATE.layouts, ...layouts },
          windowPanels: {
            ...Layout.ZERO_SLICE_STATE.windowPanels,
            [MAIN_WINDOW]: {
              ...Layout.ZERO_SLICE_STATE.windowPanels[MAIN_WINDOW],
              activeTab,
            },
          },
        },
      },
      middleware: (getDefault) => getDefault().concat(MIDDLEWARE),
    });
    store.dispatch(actions.create({ key: layoutKey }));
    return store;
  };

  const mosaicLayout = (key: string): Layout.State => ({
    key,
    windowKey: MAIN_WINDOW,
    type: "schematic",
    name: key,
    location: "mosaic",
  });

  it("should apply setSelected when the schematic is the active mosaic tab", () => {
    const store = buildStore({ [layoutKey]: mosaicLayout(layoutKey) }, layoutKey);
    store.dispatch(actions.setSelected({ key: layoutKey, selected: ["n1"] }));
    expect(store.getState()[SLICE_NAME].schematics[layoutKey].selected).toEqual(["n1"]);
  });
});
