// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { configureStore } from "@reduxjs/toolkit";
import { beforeEach, describe, expect, it } from "vitest";

import {
  actions,
  reducer,
  SLICE_NAME,
  type StoreState,
  ZERO_SLICE_STATE,
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
    expect(s.activeToolbarTab).toBe("symbols");
    expect(s.editable).toBe(true);
  });

  it("should track selection and switch toolbar tab", () => {
    store.dispatch(actions.setSelected({ key: layoutKey, selected: ["n1"] }));
    let s = store.getState()[SLICE_NAME].schematics[layoutKey];
    expect(s.selected).toEqual(["n1"]);
    expect(s.activeToolbarTab).toBe("properties");
    store.dispatch(actions.setSelected({ key: layoutKey, selected: [] }));
    s = store.getState()[SLICE_NAME].schematics[layoutKey];
    expect(s.activeToolbarTab).toBe("symbols");
  });

  it("should toggle editable and clear selection on disable", () => {
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
});
