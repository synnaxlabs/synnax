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

import { Arc } from "@/layered/session/arc";

const storeWith = (slice: Arc.SliceState) =>
  configureStore({
    reducer: { [Arc.SLICE_NAME]: Arc.reducer },
    preloadedState: { [Arc.SLICE_NAME]: slice },
  });

const KEY = "arc-1";

describe("Arc Slice", () => {
  let store: ReturnType<typeof storeWith>;

  beforeEach(() => {
    store = storeWith(Arc.ZERO_SLICE_STATE);
  });

  const select = <R>(
    selector: (params: Arc.KeyedSelectorParams) => R,
    key: string = KEY,
  ): R => selector({ state: store.getState(), key });

  describe("create", () => {
    it("should bootstrap session state from ZERO_STATE for the key", () => {
      store.dispatch(Arc.internalCreate({ key: KEY }));
      expect(select(Arc.selectState)).toEqual(Arc.ZERO_STATE);
    });

    it("should reset the active toolbar tab to stages", () => {
      store.dispatch(Arc.setActiveToolbarTab({ tab: "properties" }));
      store.dispatch(Arc.internalCreate({ key: KEY }));
      expect(Arc.selectToolbar(store.getState()).activeTab).toBe("stages");
    });

    it("should create multiple arcs independently", () => {
      store.dispatch(Arc.internalCreate({ key: "arc-1" }));
      store.dispatch(Arc.internalCreate({ key: "arc-2" }));
      expect(Object.keys(Arc.selectSliceState(store.getState()).arcs)).toHaveLength(2);
    });

    it("should apply provided graph fields over the defaults", () => {
      store.dispatch(Arc.internalCreate({ key: KEY, graph: { editable: false } }));
      expect(select(Arc.selectEditable)).toBe(false);
    });

    it("should not overwrite an existing entry", () => {
      store.dispatch(Arc.internalCreate({ key: KEY }));
      store.dispatch(Arc.setEditable({ key: KEY, editable: false }));
      store.dispatch(Arc.internalCreate({ key: KEY }));
      expect(select(Arc.selectEditable)).toBe(false);
    });
  });

  describe("setSelected", () => {
    it("should set the per-arc selection", () => {
      store.dispatch(Arc.internalCreate({ key: KEY }));
      store.dispatch(Arc.setSelected({ key: KEY, selected: ["a", "b"] }));
      expect(select(Arc.selectSelected)).toEqual(["a", "b"]);
    });

    it("should switch the toolbar to properties when selecting", () => {
      store.dispatch(Arc.internalCreate({ key: KEY }));
      store.dispatch(Arc.setSelected({ key: KEY, selected: ["a"] }));
      expect(Arc.selectToolbar(store.getState()).activeTab).toBe("properties");
    });

    it("should switch the toolbar back to stages when clearing", () => {
      store.dispatch(Arc.internalCreate({ key: KEY }));
      store.dispatch(Arc.setSelected({ key: KEY, selected: ["a"] }));
      store.dispatch(Arc.setSelected({ key: KEY, selected: [] }));
      expect(Arc.selectToolbar(store.getState()).activeTab).toBe("stages");
    });

    it("should clear other arcs' selections when entering the properties tab", () => {
      store.dispatch(Arc.internalCreate({ key: "arc-1", graph: { selected: ["a"] } }));
      store.dispatch(Arc.internalCreate({ key: "arc-2" }));
      store.dispatch(Arc.setSelected({ key: "arc-2", selected: ["b"] }));
      expect(select(Arc.selectSelected, "arc-1")).toEqual([]);
      expect(select(Arc.selectSelected, "arc-2")).toEqual(["b"]);
    });

    it("should lazily create the entry when the key does not exist", () => {
      store.dispatch(Arc.setSelected({ key: KEY, selected: ["a"] }));
      expect(select(Arc.selectSelected)).toEqual(["a"]);
    });
  });

  describe("setViewport", () => {
    it("should set the per-arc viewport", () => {
      store.dispatch(Arc.internalCreate({ key: KEY }));
      const viewport = { position: { x: 5, y: 6 }, zoom: 2 };
      store.dispatch(Arc.setViewport({ key: KEY, viewport }));
      expect(select(Arc.selectViewport)).toEqual(viewport);
    });

    it("should lazily create the entry when the key does not exist", () => {
      const viewport = { position: { x: 1, y: 2 }, zoom: 3 };
      store.dispatch(Arc.setViewport({ key: KEY, viewport }));
      expect(select(Arc.selectViewport)).toEqual(viewport);
    });
  });

  describe("setEditable", () => {
    it("should set the editable flag", () => {
      store.dispatch(Arc.internalCreate({ key: KEY, graph: { editable: false } }));
      store.dispatch(Arc.setEditable({ key: KEY, editable: true }));
      expect(select(Arc.selectEditable)).toBe(true);
    });

    it("should clear the selection when toggled", () => {
      store.dispatch(Arc.internalCreate({ key: KEY }));
      store.dispatch(Arc.setSelected({ key: KEY, selected: ["a"] }));
      store.dispatch(Arc.setEditable({ key: KEY, editable: false }));
      expect(select(Arc.selectSelected)).toEqual([]);
    });
  });

  describe("setFitViewOnResize", () => {
    it("should set the fit-view-on-resize flag", () => {
      store.dispatch(Arc.internalCreate({ key: KEY }));
      store.dispatch(Arc.setFitViewOnResize({ key: KEY, fitViewOnResize: true }));
      expect(select(Arc.selectState).graph.fitViewOnResize).toBe(true);
    });
  });

  describe("setViewportMode", () => {
    it("should set the global viewport mode", () => {
      store.dispatch(Arc.setViewportMode({ mode: "pan" }));
      expect(Arc.selectViewportMode(store.getState())).toBe("pan");
    });
  });

  describe("remove", () => {
    it("should remove an arc by key", () => {
      store.dispatch(Arc.internalCreate({ key: KEY }));
      store.dispatch(Arc.remove({ keys: [KEY] }));
      expect(Arc.selectSliceState(store.getState()).arcs[KEY]).toBeUndefined();
    });

    it("should remove multiple arcs at once", () => {
      store.dispatch(Arc.internalCreate({ key: "arc-1" }));
      store.dispatch(Arc.internalCreate({ key: "arc-2" }));
      store.dispatch(Arc.remove({ keys: ["arc-1", "arc-2"] }));
      expect(Object.keys(Arc.selectSliceState(store.getState()).arcs)).toHaveLength(0);
    });

    it("should ignore keys that do not exist", () => {
      store.dispatch(Arc.internalCreate({ key: KEY }));
      store.dispatch(Arc.remove({ keys: ["absent"] }));
      expect(select(Arc.selectState)).toEqual(Arc.ZERO_STATE);
    });
  });

  describe("selectors", () => {
    it("should fall back to ZERO_STATE for an unknown key", () => {
      expect(select(Arc.selectState)).toEqual(Arc.ZERO_STATE);
    });
  });

  describe("stateZ schema", () => {
    it("should accept the zero state", () => {
      expect(() => Arc.stateZ.parse(Arc.ZERO_STATE)).not.toThrow();
    });

    it("should apply defaults when fields are missing", () => {
      const parsed = Arc.stateZ.parse({});
      expect(parsed.graph.editable).toBe(true);
      expect(parsed.graph.fitViewOnResize).toBe(false);
      expect(parsed.graph.selected).toEqual([]);
      expect(parsed.graph.viewport).toEqual({ position: { x: 0, y: 0 }, zoom: 1 });
    });
  });

  describe("sliceStateZ schema", () => {
    it("should default the slice version to 0", () => {
      expect(Arc.sliceStateZ.parse({}).version).toBe(0);
      expect(Arc.ZERO_SLICE_STATE.version).toBe(0);
    });

    it("should default the global mode and toolbar tab", () => {
      const parsed = Arc.sliceStateZ.parse({});
      expect(parsed.mode).toBe("select");
      expect(parsed.toolbar.activeTab).toBe("stages");
    });
  });
});
