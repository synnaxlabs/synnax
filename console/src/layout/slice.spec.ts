// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { configureStore } from "@reduxjs/toolkit";
import { MAIN_WINDOW } from "@synnaxlabs/drift";
import { Color, type Haul, Mosaic } from "@synnaxlabs/pluto";
import { beforeEach, describe, expect, it } from "vitest";

import {
  actions,
  reducer,
  setArgs,
  SLICE_NAME,
  type State,
  type StoreState,
  ZERO_SLICE_STATE,
} from "@/layout/slice";

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
  let store: ReturnType<typeof configureStore<StoreState>>;

  beforeEach(() => {
    store = configureStore({
      reducer: { [SLICE_NAME]: reducer },
      preloadedState: { [SLICE_NAME]: ZERO_SLICE_STATE },
    });
  });

  const slice = () => store.getState()[SLICE_NAME];

  describe("place", () => {
    it("should insert a mosaic tab and select it", () => {
      store.dispatch(actions.place(mosaicLayout("plot-1")));
      expect(slice().layouts["plot-1"]).toBeDefined();
      expect(slice().mosaics[MAIN_WINDOW].activeTab).toBe("plot-1");
    });

    it("should add a window-located layout without touching the mosaic", () => {
      store.dispatch(actions.place(windowLayout("popup-1")));
      expect(slice().layouts["popup-1"].location).toBe("window");
      expect(slice().mosaics[MAIN_WINDOW].activeTab).toBeNull();
    });
  });

  describe("remove", () => {
    it("should remove a mosaic layout and clear it from the mosaic", () => {
      store.dispatch(actions.place(mosaicLayout("plot-1")));
      store.dispatch(actions.remove({ keys: ["plot-1"] }));
      expect(slice().layouts["plot-1"]).toBeUndefined();
      expect(slice().mosaics[MAIN_WINDOW].activeTab).toBeNull();
    });

    it("should ignore the main layout", () => {
      store.dispatch(actions.remove({ keys: ["main"] }));
      expect(slice().layouts.main).toBeDefined();
    });

    it("should remove multiple layouts in one dispatch", () => {
      store.dispatch(actions.place(mosaicLayout("a")));
      store.dispatch(actions.place(mosaicLayout("b")));
      store.dispatch(actions.remove({ keys: ["a", "b"] }));
      expect(slice().layouts.a).toBeUndefined();
      expect(slice().layouts.b).toBeUndefined();
    });
  });

  describe("rename", () => {
    it("should rename a mosaic layout and its tab", () => {
      store.dispatch(actions.place(mosaicLayout("plot-1")));
      store.dispatch(actions.rename({ key: "plot-1", name: "Renamed" }));
      expect(slice().layouts["plot-1"].name).toBe("Renamed");
      const node = Mosaic.findTabNode(slice().mosaics[MAIN_WINDOW].root, "plot-1");
      expect(node?.tabs?.find((t) => t.tabKey === "plot-1")?.name).toBe("Renamed");
    });

    it("should ignore an unknown key", () => {
      store.dispatch(actions.rename({ key: "nope", name: "x" }));
      expect(slice().layouts.nope).toBeUndefined();
    });
  });

  describe("setAltKey", () => {
    it("should map a key to its alt key in both directions", () => {
      store.dispatch(actions.setAltKey({ key: "real", altKey: "alt" }));
      expect(slice().keyToAltKey.real).toBe("alt");
      expect(slice().altKeyToKey.alt).toBe("real");
    });

    it("should resolve a layout via its alt key", () => {
      store.dispatch(actions.place(mosaicLayout("real")));
      store.dispatch(actions.setAltKey({ key: "real", altKey: "alt" }));
      store.dispatch(actions.rename({ key: "alt", name: "Resolved" }));
      expect(slice().layouts.real.name).toBe("Resolved");
    });
  });

  describe("setArgs", () => {
    it("should set args on an existing layout", () => {
      store.dispatch(actions.place(mosaicLayout("plot-1")));
      store.dispatch(setArgs({ key: "plot-1", args: { foo: 42 } }));
      expect(slice().layouts["plot-1"].args).toEqual({ foo: 42 });
    });
  });

  describe("setFocus", () => {
    it("should focus a layout in its window's mosaic", () => {
      store.dispatch(actions.place(mosaicLayout("plot-1")));
      store.dispatch(actions.setFocus({ key: "plot-1", windowKey: MAIN_WINDOW }));
      expect(slice().mosaics[MAIN_WINDOW].focused).toBe("plot-1");
    });

    it("should clear focus when key is null", () => {
      store.dispatch(actions.place(mosaicLayout("plot-1")));
      store.dispatch(actions.setFocus({ key: "plot-1", windowKey: MAIN_WINDOW }));
      store.dispatch(actions.setFocus({ key: null, windowKey: MAIN_WINDOW }));
      expect(slice().mosaics[MAIN_WINDOW].focused).toBeNull();
    });
  });

  describe("setUnsavedChanges", () => {
    it("should flag a mosaic layout and propagate to its tab", () => {
      store.dispatch(actions.place(mosaicLayout("plot-1")));
      store.dispatch(
        actions.setUnsavedChanges({ key: "plot-1", unsavedChanges: true }),
      );
      expect(slice().layouts["plot-1"].unsavedChanges).toBe(true);
      const node = Mosaic.findTabNode(slice().mosaics[MAIN_WINDOW].root, "plot-1");
      expect(node?.tabs?.find((t) => t.tabKey === "plot-1")?.unsavedChanges).toBe(true);
    });
  });

  describe("setHauled", () => {
    it("should overwrite the hauling state", () => {
      const haul: Haul.DraggingState = {
        source: { key: "src", type: "drag" },
        items: [{ key: "item-1", type: "drag" }],
      };
      store.dispatch(actions.setHauled(haul));
      expect(slice().hauling).toEqual(haul);
    });
  });

  describe("setColorContext", () => {
    it("should overwrite the color context", () => {
      const ctx: Color.ContextState = {
        ...Color.ZERO_CONTEXT_STATE,
        frequent: { "#ff0000": { lastUsed: 1, count: 1, relevance: 1 } },
      };
      store.dispatch(actions.setColorContext({ state: ctx }));
      expect(slice().colorContext).toEqual(ctx);
    });
  });

  describe("setActiveTheme", () => {
    it("should set the named theme", () => {
      store.dispatch(actions.setActiveTheme("synnaxLight"));
      expect(slice().activeTheme).toBe("synnaxLight");
    });

    it("should cycle to the next theme when payload is undefined", () => {
      store.dispatch(actions.setActiveTheme(undefined));
      expect(slice().activeTheme).toBe("synnaxLight");
      store.dispatch(actions.setActiveTheme(undefined));
      expect(slice().activeTheme).toBe("synnaxDark");
    });
  });

  describe("toggleActiveTheme", () => {
    it("should cycle to the next theme", () => {
      store.dispatch(actions.toggleActiveTheme());
      expect(slice().activeTheme).toBe("synnaxLight");
    });
  });

  describe("moveMosaicTab", () => {
    it("should be a no-op when key is omitted within the same window", () => {
      store.dispatch(actions.place(mosaicLayout("plot-1")));
      const before = slice().mosaics[MAIN_WINDOW].root;
      store.dispatch(actions.moveMosaicTab({ tabKey: "plot-1", loc: "center" }));
      expect(slice().mosaics[MAIN_WINDOW].root).toEqual(before);
    });

    it("should move a tab to a different window's mosaic", () => {
      store.dispatch(actions.place(mosaicLayout("mw-2", { type: "mosaicWindow" })));
      store.dispatch(actions.place(mosaicLayout("plot-1", { windowKey: "mw-2" })));
      store.dispatch(
        actions.moveMosaicTab({
          tabKey: "plot-1",
          windowKey: MAIN_WINDOW,
          loc: "center",
        }),
      );
      expect(slice().layouts["plot-1"].windowKey).toBe(MAIN_WINDOW);
      expect(
        Mosaic.findTabNode(slice().mosaics[MAIN_WINDOW].root, "plot-1"),
      ).toBeDefined();
    });
  });

  describe("selectMosaicTab", () => {
    it("should set the active tab on the layout's window", () => {
      store.dispatch(actions.place(mosaicLayout("plot-1")));
      store.dispatch(actions.place(mosaicLayout("plot-2")));
      store.dispatch(actions.selectMosaicTab({ tabKey: "plot-1" }));
      expect(slice().mosaics[MAIN_WINDOW].activeTab).toBe("plot-1");
    });
  });

  describe("splitMosaicNode", () => {
    it("should split the node containing the tab", () => {
      store.dispatch(actions.place(mosaicLayout("plot-1")));
      store.dispatch(actions.place(mosaicLayout("plot-2")));
      store.dispatch(
        actions.splitMosaicNode({
          windowKey: MAIN_WINDOW,
          tabKey: "plot-1",
          direction: "x",
        }),
      );
      const root = slice().mosaics[MAIN_WINDOW].root;
      expect(root.first).toBeDefined();
      expect(root.last).toBeDefined();
    });
  });

  describe("resizeMosaicTab", () => {
    it("should set the size on the targeted node", () => {
      store.dispatch(actions.place(mosaicLayout("plot-1")));
      store.dispatch(actions.place(mosaicLayout("plot-2")));
      store.dispatch(
        actions.splitMosaicNode({
          windowKey: MAIN_WINDOW,
          tabKey: "plot-1",
          direction: "x",
        }),
      );
      const rootKey = slice().mosaics[MAIN_WINDOW].root.key;
      store.dispatch(
        actions.resizeMosaicTab({ windowKey: MAIN_WINDOW, key: rootKey, size: 0.25 }),
      );
      expect(slice().mosaics[MAIN_WINDOW].root.size).toBe(0.25);
    });
  });

  describe("setNavDrawer", () => {
    it("should overwrite the drawer entry at a location", () => {
      store.dispatch(
        actions.setNavDrawer({
          windowKey: MAIN_WINDOW,
          location: "left",
          activeItem: "channel",
          menuItems: ["channel"],
          size: 320,
        }),
      );
      expect(slice().nav.main.drawers.left).toEqual({
        activeItem: "channel",
        menuItems: ["channel"],
        size: 320,
      });
    });

    it("should create nav state for an unknown window", () => {
      store.dispatch(
        actions.setNavDrawer({
          windowKey: "popup",
          location: "right",
          activeItem: null,
          menuItems: ["x"],
        }),
      );
      expect(slice().nav.popup.drawers.right?.menuItems).toEqual(["x"]);
    });
  });

  describe("resizeNavDrawer", () => {
    it("should set the size of an existing drawer", () => {
      store.dispatch(
        actions.resizeNavDrawer({
          windowKey: MAIN_WINDOW,
          location: "left",
          size: 480,
        }),
      );
      expect(slice().nav.main.drawers.left.size).toBe(480);
    });

    it("should ignore a window or location without a drawer", () => {
      store.dispatch(
        actions.resizeNavDrawer({ windowKey: "absent", location: "left", size: 100 }),
      );
      expect(slice().nav.absent).toBeUndefined();
    });
  });

  describe("setNavDrawerVisible", () => {
    it("should throw when windowKey is missing", () => {
      expect(() =>
        store.dispatch(actions.setNavDrawerVisible({ key: "visualization" })),
      ).toThrow(/windowKey/);
    });

    it("should throw when neither key nor location is provided", () => {
      expect(() =>
        store.dispatch(actions.setNavDrawerVisible({ windowKey: MAIN_WINDOW })),
      ).toThrow(/key or location/);
    });

    it("should activate a menu item by key", () => {
      store.dispatch(
        actions.setNavDrawerVisible({ windowKey: MAIN_WINDOW, key: "visualization" }),
      );
      expect(slice().nav.main.drawers.bottom?.activeItem).toBe("visualization");
    });

    it("should clear the active item when the same key is dispatched again", () => {
      store.dispatch(
        actions.setNavDrawerVisible({ windowKey: MAIN_WINDOW, key: "visualization" }),
      );
      store.dispatch(
        actions.setNavDrawerVisible({ windowKey: MAIN_WINDOW, key: "visualization" }),
      );
      expect(slice().nav.main.drawers.bottom?.activeItem).toBeNull();
    });

    it("should respect an explicit value=false", () => {
      store.dispatch(
        actions.setNavDrawerVisible({ windowKey: MAIN_WINDOW, key: "visualization" }),
      );
      store.dispatch(
        actions.setNavDrawerVisible({
          windowKey: MAIN_WINDOW,
          location: "bottom",
          value: false,
        }),
      );
      expect(slice().nav.main.drawers.bottom?.activeItem).toBeNull();
    });

    it("should activate the first menu item when location is provided with value=true", () => {
      store.dispatch(
        actions.setNavDrawerVisible({
          windowKey: MAIN_WINDOW,
          location: "left",
          value: true,
        }),
      );
      expect(slice().nav.main.drawers.left.activeItem).toBe("channel");
    });
  });

  describe("startNavHover", () => {
    it("should set hover and active item on an empty drawer", () => {
      store.dispatch(
        actions.startNavHover({
          windowKey: MAIN_WINDOW,
          location: "left",
          key: "channel",
        }),
      );
      expect(slice().nav.main.drawers.left).toMatchObject({
        hover: true,
        activeItem: "channel",
      });
    });

    it("should ignore an already-active drawer that is not in hover mode", () => {
      store.dispatch(
        actions.setNavDrawerVisible({ windowKey: MAIN_WINDOW, key: "visualization" }),
      );
      store.dispatch(
        actions.startNavHover({
          windowKey: MAIN_WINDOW,
          location: "bottom",
          key: "channel",
        }),
      );
      expect(slice().nav.main.drawers.bottom?.activeItem).toBe("visualization");
      expect(slice().nav.main.drawers.bottom?.hover).toBeFalsy();
    });
  });

  describe("toggleNavHover", () => {
    it("should enter hover mode on an empty drawer that contains the key", () => {
      store.dispatch(
        actions.toggleNavHover({ windowKey: MAIN_WINDOW, key: "channel" }),
      );
      expect(slice().nav.main.drawers.left).toMatchObject({
        hover: true,
        activeItem: "channel",
      });
    });
  });

  describe("stopNavHover", () => {
    it("should clear hover and active item when in hover mode", () => {
      store.dispatch(
        actions.startNavHover({
          windowKey: MAIN_WINDOW,
          location: "left",
          key: "channel",
        }),
      );
      store.dispatch(
        actions.stopNavHover({ windowKey: MAIN_WINDOW, location: "left" }),
      );
      expect(slice().nav.main.drawers.left).toMatchObject({
        hover: false,
        activeItem: null,
      });
    });
  });

  describe("hideAllNavDrawers", () => {
    it("should clear active item and hover for every drawer in every window", () => {
      store.dispatch(
        actions.setNavDrawerVisible({ windowKey: MAIN_WINDOW, key: "visualization" }),
      );
      store.dispatch(
        actions.startNavHover({
          windowKey: MAIN_WINDOW,
          location: "left",
          key: "channel",
        }),
      );
      store.dispatch(actions.hideAllNavDrawers());
      expect(slice().nav.main.drawers.bottom?.activeItem).toBeNull();
      expect(slice().nav.main.drawers.left.activeItem).toBeNull();
      expect(slice().nav.main.drawers.left.hover).toBe(false);
    });
  });

  describe("setWorkspace", () => {
    it("should preserve window-located layouts when applying a workspace", () => {
      store.dispatch(actions.place(windowLayout("popup-1")));
      const ws = {
        ...ZERO_SLICE_STATE,
        layouts: { ...ZERO_SLICE_STATE.layouts, "ws-plot": mosaicLayout("ws-plot") },
      };
      store.dispatch(actions.setWorkspace({ slice: ws }));
      expect(slice().layouts["popup-1"]).toBeDefined();
      expect(slice().layouts["ws-plot"]).toBeDefined();
      expect(slice().layouts.main).toBeDefined();
    });

    it("should adopt the workspace's nav state when keepNav is false", () => {
      const ws = {
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
      store.dispatch(actions.setWorkspace({ slice: ws, keepNav: false }));
      expect(slice().nav.main.drawers.left.activeItem).toBe("task");
    });
  });

  describe("clearWorkspace", () => {
    it("should drop mosaic layouts but preserve window-located ones", () => {
      store.dispatch(actions.place(mosaicLayout("plot-1")));
      store.dispatch(actions.place(windowLayout("popup-1")));
      store.dispatch(actions.clearWorkspace());
      expect(slice().layouts["plot-1"]).toBeUndefined();
      expect(slice().layouts["popup-1"]).toBeDefined();
      expect(slice().layouts.main).toBeDefined();
    });
  });
});
