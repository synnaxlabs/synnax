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

  beforeEach(() => {
    store = configureStore({
      reducer: {
        [SLICE_NAME]: reducer,
      },
      preloadedState: {
        [SLICE_NAME]: ZERO_SLICE_STATE,
      },
    });
  });

  describe("selection", () => {
    it("should set selected elements and switch to properties tab", () => {
      store.dispatch(
        actions.setSelected({ key: "s1", selected: ["node-1", "node-2"] }),
      );
      const state = store.getState()[SLICE_NAME];
      expect(state.schematics["s1"].selected).toEqual(["node-1", "node-2"]);
      expect(state.schematics["s1"].activeToolbarTab).toBe("properties");
    });

    it("should switch back to symbols tab when selection is cleared", () => {
      store.dispatch(actions.setSelected({ key: "s1", selected: ["node-1"] }));
      store.dispatch(actions.setSelected({ key: "s1", selected: [] }));
      const state = store.getState()[SLICE_NAME];
      expect(state.schematics["s1"].selected).toEqual([]);
      expect(state.schematics["s1"].activeToolbarTab).toBe("symbols");
    });

    it("should auto-create schematic state on first access", () => {
      store.dispatch(actions.setSelected({ key: "new-schematic", selected: ["a"] }));
      const state = store.getState()[SLICE_NAME];
      expect(state.schematics["new-schematic"]).toBeDefined();
      expect(state.schematics["new-schematic"].control).toBe("released");
    });
  });

  describe("control status", () => {
    it("should set control status", () => {
      store.dispatch(actions.setControlStatus({ key: "s1", control: "acquired" }));
      const state = store.getState()[SLICE_NAME];
      expect(state.schematics["s1"].control).toBe("acquired");
    });
  });

  describe("legend", () => {
    it("should toggle legend visibility", () => {
      store.dispatch(actions.setLegendVisible({ key: "s1", visible: true }));
      let state = store.getState()[SLICE_NAME];
      expect(state.schematics["s1"].legend.visible).toBe(true);

      store.dispatch(actions.setLegendVisible({ key: "s1", visible: false }));
      state = store.getState()[SLICE_NAME];
      expect(state.schematics["s1"].legend.visible).toBe(false);
    });

    it("should partially update legend", () => {
      store.dispatch(
        actions.setLegend({
          key: "s1",
          legend: { visible: true, position: { x: 100, y: 200 } },
        }),
      );
      const state = store.getState()[SLICE_NAME];
      expect(state.schematics["s1"].legend).toEqual({
        visible: true,
        position: { x: 100, y: 200 },
      });
    });
  });

  describe("toolbar", () => {
    it("should set active toolbar tab", () => {
      store.dispatch(actions.setActiveToolbarTab({ key: "s1", tab: "properties" }));
      const state = store.getState()[SLICE_NAME];
      expect(state.schematics["s1"].activeToolbarTab).toBe("properties");
    });

    it("should set selected symbol group", () => {
      store.dispatch(actions.setSelectedSymbolGroup({ key: "s1", group: "valves" }));
      const state = store.getState()[SLICE_NAME];
      expect(state.schematics["s1"].selectedSymbolGroup).toBe("valves");
    });
  });

  describe("removal", () => {
    it("should remove schematics", () => {
      store.dispatch(actions.setSelected({ key: "s1", selected: [] }));
      store.dispatch(actions.setSelected({ key: "s2", selected: [] }));
      expect(Object.keys(store.getState()[SLICE_NAME].schematics)).toHaveLength(2);

      store.dispatch(actions.remove({ keys: ["s1"] }));
      const state = store.getState()[SLICE_NAME];
      expect(Object.keys(state.schematics)).toHaveLength(1);
      expect(state.schematics["s1"]).toBeUndefined();
      expect(state.schematics["s2"]).toBeDefined();
    });

    it("should remove multiple schematics at once", () => {
      store.dispatch(actions.setSelected({ key: "s1", selected: [] }));
      store.dispatch(actions.setSelected({ key: "s2", selected: [] }));
      store.dispatch(actions.setSelected({ key: "s3", selected: [] }));

      store.dispatch(actions.remove({ keys: ["s1", "s3"] }));
      const state = store.getState()[SLICE_NAME];
      expect(Object.keys(state.schematics)).toHaveLength(1);
      expect(state.schematics["s2"]).toBeDefined();
    });
  });
});
