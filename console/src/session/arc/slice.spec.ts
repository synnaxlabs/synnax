// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { MAIN_WINDOW } from "@synnaxlabs/drift";
import { beforeEach, describe, expect, it } from "vitest";

import { Arc } from "@/session/arc";
import {
  createSliceStore,
  documentIn,
  DRIFT_STATE,
  inWindow,
} from "@/session/window/testutil";

const storeWith = (slice: Arc.SliceState) =>
  createSliceStore({
    name: Arc.SLICE_NAME,
    reducer: Arc.reducer,
    preloadedState: slice,
    middleware: Arc.MIDDLEWARE,
  });

const KEY = "arc-1";
const AUX_WINDOW = "aux-window";

describe("Arc Slice", () => {
  let store: ReturnType<typeof storeWith>;

  beforeEach(() => {
    store = storeWith(Arc.ZERO_SLICE_STATE);
  });

  const doc = (key: string = KEY): Arc.State | undefined =>
    documentIn(store.getState()[Arc.SLICE_NAME], key);

  // A window is a viewport: an edit in one window must not reach another window's
  // view of the same document.
  it("should keep each window's view of a document independent", () => {
    store.dispatch(Arc.setSelected({ key: KEY, selected: ["a"] }));
    store.dispatch(
      Arc.setEditable({ key: KEY, editable: false, windowKey: AUX_WINDOW }),
    );
    const slice = store.getState()[Arc.SLICE_NAME];
    expect(documentIn(slice, KEY, AUX_WINDOW)?.graph.editable).toBe(false);
    expect(documentIn(slice, KEY)?.graph.editable).toBe(true);
  });

  describe("setSelected", () => {
    it("should set the per-arc selection", () => {
      store.dispatch(Arc.setSelected({ key: KEY, selected: ["a", "b"] }));
      expect(doc()?.graph.selected).toEqual(["a", "b"]);
    });

    it("should switch the toolbar to properties when selecting", () => {
      store.dispatch(Arc.setSelected({ key: KEY, selected: ["a"] }));
      expect(doc()?.toolbar.selectedTab).toBe("properties");
    });

    it("should switch the toolbar back to stages when clearing", () => {
      store.dispatch(Arc.setSelected({ key: KEY, selected: ["a"] }));
      store.dispatch(Arc.setSelected({ key: KEY, selected: [] }));
      expect(doc()?.toolbar.selectedTab).toBe("stages");
    });

    it("should track selection and toolbar per arc independently", () => {
      store.dispatch(Arc.setSelected({ key: "arc-1", selected: ["a"] }));
      store.dispatch(Arc.setSelected({ key: "arc-2", selected: ["b"] }));
      store.dispatch(Arc.selectToolbarTab({ key: "arc-1", tab: "stages" }));
      expect(doc("arc-1")?.graph.selected).toEqual(["a"]);
      expect(doc("arc-2")?.graph.selected).toEqual(["b"]);
      expect(doc("arc-2")?.toolbar.selectedTab).toBe("properties");
    });
  });

  describe("selectToolbarTab", () => {
    it("should set the active toolbar tab", () => {
      store.dispatch(Arc.selectToolbarTab({ key: KEY, tab: "properties" }));
      expect(doc()?.toolbar.selectedTab).toBe("properties");
    });
  });

  describe("setViewport", () => {
    it("should merge the viewport over the existing one", () => {
      store.dispatch(
        Arc.setViewport({ key: KEY, viewport: { position: { x: 5, y: 6 }, zoom: 2 } }),
      );
      const viewport = doc()?.graph.viewport;
      expect(viewport?.position).toEqual({ x: 5, y: 6 });
      expect(viewport?.zoom).toBe(2);
      expect(viewport?.mode).toBe(Arc.ZERO_STATE.graph.viewport.mode);
    });
  });

  describe("setViewportMode", () => {
    it("should set the per-arc viewport mode without touching position or zoom", () => {
      store.dispatch(Arc.setViewportMode({ key: KEY, mode: "pan" }));
      const viewport = doc()?.graph.viewport;
      expect(viewport?.mode).toBe("pan");
      expect(viewport?.position).toEqual(Arc.ZERO_STATE.graph.viewport.position);
    });
  });

  describe("setEditable", () => {
    it("should set the editable flag", () => {
      store.dispatch(Arc.setEditable({ key: KEY, editable: false }));
      expect(doc()?.graph.editable).toBe(false);
      store.dispatch(Arc.setEditable({ key: KEY, editable: true }));
      expect(doc()?.graph.editable).toBe(true);
    });

    it("should clear the selection when toggled", () => {
      store.dispatch(Arc.setSelected({ key: KEY, selected: ["a"] }));
      store.dispatch(Arc.setEditable({ key: KEY, editable: false }));
      expect(doc()?.graph.selected).toEqual([]);
    });
  });

  describe("setFitViewOnResize", () => {
    it("should set the fit-view-on-resize flag", () => {
      store.dispatch(Arc.setFitViewOnResize({ key: KEY, fitViewOnResize: true }));
      expect(doc()?.graph.fitViewOnResize).toBe(true);
    });
  });

  describe("remove", () => {
    it("should remove multiple arcs at once", () => {
      store.dispatch(Arc.setEditable({ key: "arc-1", editable: false }));
      store.dispatch(Arc.setEditable({ key: "arc-2", editable: false }));
      store.dispatch(Arc.remove({ keys: ["arc-1", "arc-2"] }));
      expect(doc("arc-1")).toBeUndefined();
      expect(doc("arc-2")).toBeUndefined();
    });

    it("should ignore keys that do not exist", () => {
      store.dispatch(Arc.setEditable({ key: KEY, editable: false }));
      store.dispatch(Arc.remove({ keys: ["absent"] }));
      expect(doc()?.graph.editable).toBe(false);
    });
  });

  describe("zero state", () => {
    it("should hold the defaults for a new arc", () => {
      expect(Arc.ZERO_STATE).toEqual({
        graph: {
          editable: true,
          fitViewOnResize: false,
          selected: [],
          viewport: { position: { x: 0, y: 0 }, zoom: 1, mode: "select" },
        },
        toolbar: { selectedTab: "stages" },
      });
    });
  });

  describe("sliceStateZ schema", () => {
    it("should default the slice version to 0", () => {
      expect(Arc.sliceStateZ.parse({}).version).toBe(0);
      expect(Arc.ZERO_SLICE_STATE.version).toBe(0);
    });

    it("should default the window map to empty", () => {
      expect(Arc.sliceStateZ.parse({}).windows).toEqual({});
    });
  });
});

describe("persistence", () => {
  const persist = (state: Arc.State): Arc.State | undefined => {
    const stored = {
      ...DRIFT_STATE,
      [Arc.SLICE_NAME]: { version: 0 as const, windows: inWindow({ [KEY]: state }) },
    };
    Arc.PERSIST_EXCLUDE.forEach((exclude) => exclude(stored));
    return documentIn(stored[Arc.SLICE_NAME], KEY, MAIN_WINDOW);
  };

  // A selection is the state of an edit in progress, so it does not outlive the
  // session that made it.
  it("should clear the selection on the way to disk", () => {
    const state = {
      ...Arc.ZERO_STATE,
      graph: { ...Arc.ZERO_STATE.graph, selected: ["a", "b"] },
    };
    expect(persist(state)?.graph.selected).toEqual([]);
  });

  it("should leave the rest of the document alone", () => {
    const purged = persist({
      graph: { ...Arc.ZERO_STATE.graph, selected: ["a"], editable: false },
      toolbar: { selectedTab: "properties" },
    });
    expect(purged?.graph.editable).toBe(false);
    expect(purged?.toolbar.selectedTab).toBe("properties");
  });
});
