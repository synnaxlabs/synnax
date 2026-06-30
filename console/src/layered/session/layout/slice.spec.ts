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
import { Color, type Haul, Mosaic } from "@synnaxlabs/pluto";
import { beforeEach, describe, expect, it } from "vitest";

import {
  select,
  selectActiveMosaicTabState,
  selectAltKey,
  selectArgs,
  selectColorContext,
  selectFocused,
  selectHauling,
  selectMosaic,
  selectSliceState,
} from "@/layered/session/layout/selectors";
import {
  clearProject,
  moveMosaicTab,
  place,
  reducer,
  remove,
  rename,
  resizeMosaicTab,
  selectMosaicTab,
  setAltKey,
  setArgs,
  setColorContext,
  setFocus,
  setHauled,
  setProject,
  setUnsavedChanges,
  SLICE_NAME,
  splitMosaicNode,
  type State,
  ZERO_SLICE_STATE,
} from "@/layered/session/layout/slice";

const rootReducer = combineReducers({
  [SLICE_NAME]: reducer,
  [Drift.SLICE_NAME]: Drift.reducer,
});

type TestState = ReturnType<typeof rootReducer>;

const mosaicLayout = (key: string, overrides: Partial<State> = {}): State => ({
  key,
  windowKey: MAIN_WINDOW,
  type: "schematic",
  name: key,
  location: "mosaic",
  ...overrides,
});

const windowLayout = (key: string, overrides: Partial<State> = {}): State => ({
  key,
  windowKey: MAIN_WINDOW,
  type: "schematic",
  name: key,
  location: "window",
  ...overrides,
});

describe("Layout Slice", () => {
  let store: ReturnType<typeof configureStore<TestState>>;

  beforeEach(() => {
    store = configureStore({
      reducer: rootReducer,
      preloadedState: {
        [SLICE_NAME]: ZERO_SLICE_STATE,
      },
    });
  });

  const state = () => store.getState();

  describe("place", () => {
    it("should insert a mosaic tab and select it", () => {
      store.dispatch(place(mosaicLayout("plot-1")));
      expect(select(state(), "plot-1")).toBeDefined();
      expect(selectActiveMosaicTabState(state()).layoutKey).toBe("plot-1");
    });

    it("should add a window-located layout without touching the mosaic", () => {
      store.dispatch(place(windowLayout("popup-1")));
      expect(select(state(), "popup-1")?.location).toBe("window");
      expect(selectActiveMosaicTabState(state()).layoutKey).toBeNull();
    });

    it("should recover when a layout claims location mosaic but is absent from the mosaic tree", () => {
      const layout = mosaicLayout("orphan-arc");
      store = configureStore({
        reducer: rootReducer,
        preloadedState: {
          [SLICE_NAME]: {
            ...ZERO_SLICE_STATE,
            layouts: { ...ZERO_SLICE_STATE.layouts, "orphan-arc": layout },
          },
        },
      });
      expect(() => store.dispatch(place(layout))).not.toThrow();
      const [, root] = selectMosaic(state());
      expect(Mosaic.findTabNode(root!, "orphan-arc")).toBeDefined();
      expect(selectActiveMosaicTabState(state()).layoutKey).toBe("orphan-arc");
    });

    it("should move the tab to the new window without throwing or leaving an orphan when prev.windowKey differs", () => {
      const subWindowKey = "sub-window-1";
      const subWindowLayout: State = {
        key: "cross-arc",
        windowKey: subWindowKey,
        type: "arc",
        name: "cross-arc",
        location: "mosaic",
      };
      store = configureStore({
        reducer: rootReducer,
        preloadedState: {
          [SLICE_NAME]: {
            ...ZERO_SLICE_STATE,
            layouts: {
              ...ZERO_SLICE_STATE.layouts,
              "cross-arc": subWindowLayout,
            },
            mosaics: {
              ...ZERO_SLICE_STATE.mosaics,
              [subWindowKey]: {
                activeTab: "cross-arc",
                focused: null,
                root: {
                  key: 1,
                  tabs: [{ tabKey: "cross-arc", name: "cross-arc", closable: true }],
                  selected: "cross-arc",
                },
              },
            },
          },
        },
      });
      expect(() =>
        store.dispatch(place({ ...subWindowLayout, windowKey: MAIN_WINDOW })),
      ).not.toThrow();
      const sliceState = selectSliceState(state());
      const mainRoot = sliceState.mosaics[MAIN_WINDOW].root;
      expect(Mosaic.findTabNode(mainRoot, "cross-arc")).toBeDefined();
      expect(selectActiveMosaicTabState(state()).layoutKey).toBe("cross-arc");
      const subMosaic = sliceState.mosaics[subWindowKey];
      if (subMosaic != null)
        expect(Mosaic.findTabNode(subMosaic.root, "cross-arc")).toBeUndefined();
    });
  });

  describe("remove", () => {
    it("should remove a mosaic layout and clear it from the mosaic", () => {
      store.dispatch(place(mosaicLayout("plot-1")));
      store.dispatch(remove({ keys: ["plot-1"] }));
      expect(select(state(), "plot-1")).toBeUndefined();
      expect(selectActiveMosaicTabState(state()).layoutKey).toBeNull();
    });

    it("should ignore the main layout", () => {
      store.dispatch(remove({ keys: ["main"] }));
      expect(select(state(), "main")).toBeDefined();
    });

    it("should remove multiple layouts in one dispatch", () => {
      store.dispatch(place(mosaicLayout("a")));
      store.dispatch(place(mosaicLayout("b")));
      store.dispatch(remove({ keys: ["a", "b"] }));
      expect(select(state(), "a")).toBeUndefined();
      expect(select(state(), "b")).toBeUndefined();
    });
  });

  describe("rename", () => {
    it("should rename a mosaic layout and its tab", () => {
      store.dispatch(place(mosaicLayout("plot-1")));
      store.dispatch(rename({ key: "plot-1", name: "Renamed" }));
      expect(select(state(), "plot-1")?.name).toBe("Renamed");
      const [, root] = selectMosaic(state());
      const tab = Mosaic.findTabNode(root!, "plot-1")?.tabs?.find(
        (t) => t.tabKey === "plot-1",
      );
      expect(tab?.name).toBe("Renamed");
    });

    it("should ignore an unknown key", () => {
      store.dispatch(rename({ key: "nope", name: "x" }));
      expect(select(state(), "nope")).toBeUndefined();
    });
  });

  describe("setAltKey", () => {
    it("should set an alt key for a layout", () => {
      store.dispatch(setAltKey({ key: "real", altKey: "alt" }));
      expect(selectAltKey(state(), "real")).toBe("alt");
    });

    it("should resolve a layout via its alt key", () => {
      store.dispatch(place(mosaicLayout("real")));
      store.dispatch(setAltKey({ key: "real", altKey: "alt" }));
      store.dispatch(rename({ key: "alt", name: "Resolved" }));
      expect(select(state(), "real")?.name).toBe("Resolved");
    });
  });

  describe("setArgs", () => {
    it("should set args on an existing layout", () => {
      store.dispatch(place(mosaicLayout("plot-1")));
      store.dispatch(setArgs({ key: "plot-1", args: { foo: 42 } }));
      expect(selectArgs(state(), "plot-1")).toEqual({ foo: 42 });
    });
  });

  describe("setFocus", () => {
    it("should focus a layout in its window's mosaic", () => {
      store.dispatch(place(mosaicLayout("plot-1")));
      store.dispatch(setFocus({ key: "plot-1", windowKey: MAIN_WINDOW }));
      expect(selectFocused(state()).focused).toBe("plot-1");
    });

    it("should clear focus when key is null", () => {
      store.dispatch(place(mosaicLayout("plot-1")));
      store.dispatch(setFocus({ key: "plot-1", windowKey: MAIN_WINDOW }));
      store.dispatch(setFocus({ key: null, windowKey: MAIN_WINDOW }));
      expect(selectFocused(state()).focused).toBeNull();
    });
  });

  describe("setUnsavedChanges", () => {
    it("should flag a mosaic layout and propagate to its tab", () => {
      store.dispatch(place(mosaicLayout("plot-1")));
      store.dispatch(setUnsavedChanges({ key: "plot-1", unsavedChanges: true }));
      expect(select(state(), "plot-1")?.unsavedChanges).toBe(true);
      const [, root] = selectMosaic(state());
      const tab = Mosaic.findTabNode(root!, "plot-1")?.tabs?.find(
        (t) => t.tabKey === "plot-1",
      );
      expect(tab?.unsavedChanges).toBe(true);
    });
  });

  describe("setHauled", () => {
    it("should overwrite the hauling state", () => {
      const haul: Haul.DraggingState = {
        source: { key: "src", type: "drag" },
        items: [{ key: "item-1", type: "drag" }],
      };
      store.dispatch(setHauled(haul));
      expect(selectHauling(state())).toEqual(haul);
    });
  });

  describe("setColorContext", () => {
    it("should overwrite the color context", () => {
      const ctx: Color.ContextState = {
        ...Color.ZERO_CONTEXT_STATE,
        frequent: { "#ff0000": { lastUsed: 1, count: 1, relevance: 1 } },
      };
      store.dispatch(setColorContext({ state: ctx }));
      expect(selectColorContext(state())).toEqual(ctx);
    });
  });

  describe("moveMosaicTab", () => {
    it("should be a no-op when key is omitted within the same window", () => {
      store.dispatch(place(mosaicLayout("plot-1")));
      const [, before] = selectMosaic(state());
      store.dispatch(moveMosaicTab({ tabKey: "plot-1", loc: "center" }));
      expect(selectMosaic(state())[1]).toEqual(before);
    });

    it("should move a tab to a different window's mosaic", () => {
      store.dispatch(place(mosaicLayout("mw-2", { type: "mosaicWindow" })));
      store.dispatch(place(mosaicLayout("plot-1", { windowKey: "mw-2" })));
      store.dispatch(
        moveMosaicTab({
          tabKey: "plot-1",
          windowKey: MAIN_WINDOW,
          loc: "center",
        }),
      );
      expect(select(state(), "plot-1")?.windowKey).toBe(MAIN_WINDOW);
      const [, root] = selectMosaic(state());
      expect(Mosaic.findTabNode(root!, "plot-1")).toBeDefined();
    });
  });

  describe("selectMosaicTab", () => {
    it("should set the active tab on the layout's window", () => {
      store.dispatch(place(mosaicLayout("plot-1")));
      store.dispatch(place(mosaicLayout("plot-2")));
      store.dispatch(selectMosaicTab({ tabKey: "plot-1" }));
      expect(selectActiveMosaicTabState(state()).layoutKey).toBe("plot-1");
    });
  });

  describe("splitMosaicNode", () => {
    it("should split the node containing the tab", () => {
      store.dispatch(place(mosaicLayout("plot-1")));
      store.dispatch(place(mosaicLayout("plot-2")));
      store.dispatch(
        splitMosaicNode({
          windowKey: MAIN_WINDOW,
          tabKey: "plot-1",
          direction: "x",
        }),
      );
      const [, root] = selectMosaic(state());
      expect(root?.first).toBeDefined();
      expect(root?.last).toBeDefined();
    });
  });

  describe("resizeMosaicTab", () => {
    it("should set the size on the targeted node", () => {
      store.dispatch(place(mosaicLayout("plot-1")));
      store.dispatch(place(mosaicLayout("plot-2")));
      store.dispatch(
        splitMosaicNode({
          windowKey: MAIN_WINDOW,
          tabKey: "plot-1",
          direction: "x",
        }),
      );
      const [, root] = selectMosaic(state());
      store.dispatch(
        resizeMosaicTab({
          windowKey: MAIN_WINDOW,
          key: root!.key,
          size: 0.25,
        }),
      );
      expect(selectMosaic(state())[1]?.size).toBe(0.25);
    });
  });

  describe("setProject", () => {
    it("should preserve window-located layouts when applying a project", () => {
      store.dispatch(place(windowLayout("popup-1")));
      const proj = {
        ...ZERO_SLICE_STATE,
        layouts: {
          ...ZERO_SLICE_STATE.layouts,
          "proj-plot": mosaicLayout("proj-plot"),
        },
      };
      store.dispatch(setProject({ slice: proj }));
      expect(select(state(), "popup-1")).toBeDefined();
      expect(select(state(), "proj-plot")).toBeDefined();
      expect(select(state(), "main")).toBeDefined();
    });

    it("should resurrect orphan mosaic layouts that the project's mosaics map omits", () => {
      const proj = {
        ...ZERO_SLICE_STATE,
        layouts: {
          ...ZERO_SLICE_STATE.layouts,
          "proj-orphan": mosaicLayout("proj-orphan"),
        },
      };
      store.dispatch(setProject({ slice: proj }));
      expect(select(state(), "proj-orphan")).toBeDefined();
      const [, root] = selectMosaic(state());
      expect(Mosaic.findTabNode(root!, "proj-orphan")).toBeDefined();
    });

    it("should fall back to the main mosaic when an orphan layout points at a missing window", () => {
      const proj = {
        ...ZERO_SLICE_STATE,
        layouts: {
          ...ZERO_SLICE_STATE.layouts,
          "stale-window-tab": mosaicLayout("stale-window-tab", {
            windowKey: "ghost-window",
          }),
        },
      };
      store.dispatch(setProject({ slice: proj }));
      expect(select(state(), "stale-window-tab")?.windowKey).toBe(MAIN_WINDOW);
      const [, root] = selectMosaic(state());
      expect(Mosaic.findTabNode(root!, "stale-window-tab")).toBeDefined();
    });
  });

  describe("clearProject", () => {
    it("should drop mosaic layouts but preserve window-located ones", () => {
      store.dispatch(place(mosaicLayout("plot-1")));
      store.dispatch(place(windowLayout("popup-1")));
      store.dispatch(clearProject());
      expect(select(state(), "plot-1")).toBeUndefined();
      expect(select(state(), "popup-1")).toBeDefined();
      expect(select(state(), "main")).toBeDefined();
    });
  });
});
