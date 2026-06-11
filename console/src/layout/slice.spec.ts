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
import { Color, type Haul } from "@synnaxlabs/pluto";
import { beforeEach, describe, expect, it } from "vitest";

import {
  select,
  selectActiveThemeKey,
  selectArgs,
  selectColorContext,
  selectFocused,
  selectHauling,
  selectNavDrawer,
  selectSliceState,
} from "@/layout/selectors";
import {
  clearProject,
  hideAllNavDrawers,
  place,
  reducer,
  remove,
  rename,
  resizeNavDrawer,
  setActiveTheme,
  setArgs,
  setColorContext,
  setFocus,
  setHauled,
  setNavDrawer,
  setNavDrawerVisible,
  setProject,
  setUnsavedChanges,
  SLICE_NAME,
  startNavHover,
  type State,
  stopNavHover,
  toggleActiveTheme,
  toggleNavHover,
  ZERO_SLICE_STATE,
} from "@/layout/slice";

const rootReducer = combineReducers({
  [SLICE_NAME]: reducer,
  drift: Drift.reducer,
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
    it("should store a document-content layout", () => {
      store.dispatch(place(mosaicLayout("plot-1")));
      expect(select(state(), "plot-1")).toBeDefined();
    });

    it("should store a window-located layout", () => {
      store.dispatch(place(windowLayout("popup-1")));
      expect(select(state(), "popup-1")?.location).toBe("window");
    });
  });

  describe("remove", () => {
    it("should remove a layout", () => {
      store.dispatch(place(mosaicLayout("plot-1")));
      store.dispatch(remove({ keys: ["plot-1"] }));
      expect(select(state(), "plot-1")).toBeUndefined();
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
    it("should rename a layout", () => {
      store.dispatch(place(mosaicLayout("plot-1")));
      store.dispatch(rename({ key: "plot-1", name: "Renamed" }));
      expect(select(state(), "plot-1")?.name).toBe("Renamed");
    });

    it("should ignore an unknown key", () => {
      store.dispatch(rename({ key: "nope", name: "x" }));
      expect(select(state(), "nope")).toBeUndefined();
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
    it("should flag a layout", () => {
      store.dispatch(place(mosaicLayout("plot-1")));
      store.dispatch(setUnsavedChanges({ key: "plot-1", unsavedChanges: true }));
      expect(select(state(), "plot-1")?.unsavedChanges).toBe(true);
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

  describe("setActiveTheme", () => {
    it("should set the named theme", () => {
      store.dispatch(setActiveTheme("synnaxLight"));
      expect(selectActiveThemeKey(state())).toBe("synnaxLight");
    });

    it("should cycle to the next theme when payload is undefined", () => {
      store.dispatch(setActiveTheme(undefined));
      expect(selectActiveThemeKey(state())).toBe("synnaxLight");
      store.dispatch(setActiveTheme(undefined));
      expect(selectActiveThemeKey(state())).toBe("synnaxDark");
    });
  });

  describe("toggleActiveTheme", () => {
    it("should cycle to the next theme", () => {
      store.dispatch(toggleActiveTheme());
      expect(selectActiveThemeKey(state())).toBe("synnaxLight");
    });
  });

  describe("setNavDrawer", () => {
    it("should overwrite the drawer entry at a location", () => {
      store.dispatch(
        setNavDrawer({
          windowKey: MAIN_WINDOW,
          location: "left",
          activeItem: "channel",
          menuItems: ["channel"],
          size: 320,
        }),
      );
      expect(selectNavDrawer(state(), "left")).toEqual({
        activeItem: "channel",
        menuItems: ["channel"],
        size: 320,
      });
    });

    it("should create nav state for an unknown window", () => {
      store.dispatch(
        setNavDrawer({
          windowKey: "popup",
          location: "right",
          activeItem: null,
          menuItems: ["x"],
        }),
      );
      expect(selectSliceState(state()).nav.popup.drawers.right?.menuItems).toEqual([
        "x",
      ]);
    });
  });

  describe("resizeNavDrawer", () => {
    it("should set the size of an existing drawer", () => {
      store.dispatch(
        resizeNavDrawer({
          windowKey: MAIN_WINDOW,
          location: "left",
          size: 480,
        }),
      );
      expect(selectNavDrawer(state(), "left")?.size).toBe(480);
    });

    it("should ignore a window or location without a drawer", () => {
      store.dispatch(
        resizeNavDrawer({ windowKey: "absent", location: "left", size: 100 }),
      );
      expect(selectSliceState(state()).nav.absent).toBeUndefined();
    });
  });

  describe("setNavDrawerVisible", () => {
    it("should throw when windowKey is missing", () => {
      expect(() =>
        store.dispatch(setNavDrawerVisible({ key: "visualization" })),
      ).toThrow(/windowKey/);
    });

    it("should throw when neither key nor location is provided", () => {
      expect(() =>
        store.dispatch(setNavDrawerVisible({ windowKey: MAIN_WINDOW })),
      ).toThrow(/key or location/);
    });

    it("should activate a menu item by key", () => {
      store.dispatch(
        setNavDrawerVisible({ windowKey: MAIN_WINDOW, key: "visualization" }),
      );
      expect(selectNavDrawer(state(), "bottom")?.activeItem).toBe("visualization");
    });

    it("should clear the active item when the same key is dispatched again", () => {
      store.dispatch(
        setNavDrawerVisible({ windowKey: MAIN_WINDOW, key: "visualization" }),
      );
      store.dispatch(
        setNavDrawerVisible({ windowKey: MAIN_WINDOW, key: "visualization" }),
      );
      expect(selectNavDrawer(state(), "bottom")?.activeItem).toBeNull();
    });

    it("should respect an explicit value=false", () => {
      store.dispatch(
        setNavDrawerVisible({ windowKey: MAIN_WINDOW, key: "visualization" }),
      );
      store.dispatch(
        setNavDrawerVisible({
          windowKey: MAIN_WINDOW,
          location: "bottom",
          value: false,
        }),
      );
      expect(selectNavDrawer(state(), "bottom")?.activeItem).toBeNull();
    });

    it("should activate the first menu item when location is provided with value=true", () => {
      store.dispatch(
        setNavDrawerVisible({
          windowKey: MAIN_WINDOW,
          location: "left",
          value: true,
        }),
      );
      expect(selectNavDrawer(state(), "left")?.activeItem).toBe("channel");
    });
  });

  describe("startNavHover", () => {
    it("should set hover and active item on an empty drawer", () => {
      store.dispatch(
        startNavHover({
          windowKey: MAIN_WINDOW,
          location: "left",
          key: "channel",
        }),
      );
      expect(selectNavDrawer(state(), "left")).toMatchObject({
        hover: true,
        activeItem: "channel",
      });
    });

    it("should ignore an already-active drawer that is not in hover mode", () => {
      store.dispatch(
        setNavDrawerVisible({ windowKey: MAIN_WINDOW, key: "visualization" }),
      );
      store.dispatch(
        startNavHover({
          windowKey: MAIN_WINDOW,
          location: "bottom",
          key: "channel",
        }),
      );
      expect(selectNavDrawer(state(), "bottom")?.activeItem).toBe("visualization");
      expect(selectNavDrawer(state(), "bottom")?.hover).toBeFalsy();
    });
  });

  describe("toggleNavHover", () => {
    it("should enter hover mode on an empty drawer that contains the key", () => {
      store.dispatch(toggleNavHover({ windowKey: MAIN_WINDOW, key: "channel" }));
      expect(selectNavDrawer(state(), "left")).toMatchObject({
        hover: true,
        activeItem: "channel",
      });
    });
  });

  describe("stopNavHover", () => {
    it("should clear hover and active item when in hover mode", () => {
      store.dispatch(
        startNavHover({
          windowKey: MAIN_WINDOW,
          location: "left",
          key: "channel",
        }),
      );
      store.dispatch(stopNavHover({ windowKey: MAIN_WINDOW, location: "left" }));
      expect(selectNavDrawer(state(), "left")).toMatchObject({
        hover: false,
        activeItem: null,
      });
    });
  });

  describe("hideAllNavDrawers", () => {
    it("should clear active item and hover for every drawer in every window", () => {
      store.dispatch(
        setNavDrawerVisible({ windowKey: MAIN_WINDOW, key: "visualization" }),
      );
      store.dispatch(
        startNavHover({
          windowKey: MAIN_WINDOW,
          location: "left",
          key: "channel",
        }),
      );
      store.dispatch(hideAllNavDrawers());
      expect(selectNavDrawer(state(), "bottom")?.activeItem).toBeNull();
      expect(selectNavDrawer(state(), "left")?.activeItem).toBeNull();
      expect(selectNavDrawer(state(), "left")?.hover).toBe(false);
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

    it("should adopt the project's nav state when keepNav is false", () => {
      const proj = {
        ...ZERO_SLICE_STATE,
        nav: {
          ...ZERO_SLICE_STATE.nav,
          main: {
            drawers: {
              ...ZERO_SLICE_STATE.nav.main.drawers,
              left: {
                ...ZERO_SLICE_STATE.nav.main.drawers.left,
                activeItem: "task",
              },
            },
          },
        },
      };
      store.dispatch(setProject({ slice: proj, keepNav: false }));
      expect(selectNavDrawer(state(), "left")?.activeItem).toBe("task");
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
