// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { combineReducers, configureStore } from "@reduxjs/toolkit";
import { UnexpectedError } from "@synnaxlabs/client";
import { Drift } from "@synnaxlabs/drift";
import { describe, expect, it } from "vitest";

import { Nav } from "@/session/nav";

const rootReducer = combineReducers({
  [Nav.SLICE_NAME]: Nav.reducer,
  [Drift.SLICE_NAME]: Drift.reducer,
});

// run dispatches the actions through a store wired with the inject-key middleware, so
// actions without an explicit windowKey resolve to the active window (MAIN_WINDOW). The
// resulting state is read back through selectWindowState, exercising the slice,
// middleware, and Drift-coupled selectors together.
const run = (...actions: Nav.Action[]) => {
  const store = configureStore({
    reducer: rootReducer,
    middleware: (getDefault) => getDefault().concat(Nav.MIDDLEWARE),
  });
  actions.forEach((action) => store.dispatch(action));
  const windowState = () => Nav.selectWindowState(store.getState());
  return {
    left: () => windowState().left,
    leftSelected: () => windowState().left.selected,
    bottom: () => windowState().bottom,
    bottomVisible: () => windowState().bottom.visible,
  };
};

describe("Nav Slice", () => {
  describe("schemas", () => {
    it("should default the left state to collapsed with no selection", () => {
      expect(Nav.leftStateZ.parse({})).toEqual({
        hover: false,
        selected: undefined,
        size: undefined,
      });
    });

    it("should default the bottom state to hidden", () => {
      expect(Nav.bottomStateZ.parse({})).toEqual({
        hover: false,
        visible: false,
        size: undefined,
      });
    });

    it("should prefault the window state with zero left and bottom states", () => {
      expect(Nav.windowStateZ.parse({})).toEqual(Nav.ZERO_WINDOW_STATE);
    });

    it("should default the slice state to an empty window map", () => {
      expect(Nav.ZERO_SLICE_STATE).toEqual({ windows: {} });
    });
  });

  describe("withKey window bootstrapping", () => {
    it("should create a window entry from the zero state on first action", () => {
      const s = run(Nav.resizeLeft({ size: 200 }));
      expect(s.left()).toEqual({ ...Nav.ZERO_WINDOW_STATE.left, size: 200 });
      expect(s.bottom()).toEqual(Nav.ZERO_WINDOW_STATE.bottom);
    });

    it("should keep window states independent across keys", () => {
      const s = run(
        Nav.selectLeft({ key: "a" }),
        Nav.selectLeft({ windowKey: "window-2", key: "b" }),
      );
      expect(s.leftSelected()).toBe("a");
    });

    it("should throw an UnexpectedError when the windowKey is missing", () => {
      expect(() =>
        Nav.reducer(Nav.ZERO_SLICE_STATE, Nav.selectLeft({ key: "a" })),
      ).toThrow(UnexpectedError);
    });
  });

  describe("left navigation", () => {
    describe("selectLeft", () => {
      it("should select and pin an unselected item", () => {
        const s = run(Nav.selectLeft({ key: "a" }));
        expect(s.leftSelected()).toBe("a");
        expect(s.left().hover).toBe(false);
      });

      it("should deselect when selecting the pinned item again", () => {
        const s = run(Nav.selectLeft({ key: "a" }), Nav.selectLeft({ key: "a" }));
        expect(s.leftSelected()).toBeUndefined();
      });

      it("should switch to a different pinned item", () => {
        const s = run(Nav.selectLeft({ key: "a" }), Nav.selectLeft({ key: "b" }));
        expect(s.leftSelected()).toBe("b");
        expect(s.left().hover).toBe(false);
      });

      it("should pin a currently-hovered item instead of closing it", () => {
        const s = run(Nav.startLeftHover({ key: "a" }), Nav.selectLeft({ key: "a" }));
        expect(s.leftSelected()).toBe("a");
        expect(s.left().hover).toBe(false);
      });
    });

    describe("pinLeft", () => {
      it("should pin the item regardless of prior state", () => {
        const s = run(Nav.startLeftHover({ key: "a" }), Nav.pinLeft({ key: "b" }));
        expect(s.leftSelected()).toBe("b");
        expect(s.left().hover).toBe(false);
      });
    });

    describe("toggleLeft", () => {
      it("should open an item as a hover from the collapsed state", () => {
        const s = run(Nav.toggleLeft({ key: "a" }));
        expect(s.leftSelected()).toBe("a");
        expect(s.left().hover).toBe(true);
      });

      it("should close a hovered item when toggled again", () => {
        const s = run(Nav.toggleLeft({ key: "a" }), Nav.toggleLeft({ key: "a" }));
        expect(s.leftSelected()).toBeUndefined();
        expect(s.left().hover).toBe(false);
      });

      it("should close a pinned item when toggled", () => {
        const s = run(Nav.pinLeft({ key: "a" }), Nav.toggleLeft({ key: "a" }));
        expect(s.leftSelected()).toBeUndefined();
      });

      it("should switch pinned items without dropping the pin", () => {
        const s = run(Nav.pinLeft({ key: "a" }), Nav.toggleLeft({ key: "b" }));
        expect(s.leftSelected()).toBe("b");
        expect(s.left().hover).toBe(false);
      });

      it("should switch hovered items while staying in hover mode", () => {
        const s = run(Nav.startLeftHover({ key: "a" }), Nav.toggleLeft({ key: "b" }));
        expect(s.leftSelected()).toBe("b");
        expect(s.left().hover).toBe(true);
      });
    });

    describe("startLeftHover", () => {
      it("should hover an item from the collapsed state", () => {
        const s = run(Nav.startLeftHover({ key: "a" }));
        expect(s.leftSelected()).toBe("a");
        expect(s.left().hover).toBe(true);
      });

      it("should not hover over a pinned item", () => {
        const s = run(Nav.pinLeft({ key: "a" }), Nav.startLeftHover({ key: "b" }));
        expect(s.leftSelected()).toBe("a");
        expect(s.left().hover).toBe(false);
      });

      it("should move the hover to a different item", () => {
        const s = run(
          Nav.startLeftHover({ key: "a" }),
          Nav.startLeftHover({ key: "b" }),
        );
        expect(s.leftSelected()).toBe("b");
        expect(s.left().hover).toBe(true);
      });
    });

    describe("stopLeftHover", () => {
      it("should clear a hovered item", () => {
        const s = run(Nav.startLeftHover({ key: "a" }), Nav.stopLeftHover({}));
        expect(s.leftSelected()).toBeUndefined();
        expect(s.left().hover).toBe(false);
      });

      it("should leave a pinned item untouched", () => {
        const s = run(Nav.pinLeft({ key: "a" }), Nav.stopLeftHover({}));
        expect(s.leftSelected()).toBe("a");
        expect(s.left().hover).toBe(false);
      });
    });

    describe("resizeLeft", () => {
      it("should set the left size", () => {
        const s = run(Nav.resizeLeft({ size: 320 }));
        expect(s.left().size).toBe(320);
      });
    });

    describe("collapseLeft", () => {
      it("should clear a hovered item", () => {
        const s = run(Nav.startLeftHover({ key: "a" }), Nav.collapseLeft({}));
        expect(s.leftSelected()).toBeUndefined();
        expect(s.left().hover).toBe(false);
      });

      it("should clear a pinned item", () => {
        const s = run(Nav.pinLeft({ key: "a" }), Nav.collapseLeft({}));
        expect(s.leftSelected()).toBeUndefined();
        expect(s.left().hover).toBe(false);
      });

      it("should leave an already-collapsed left nav untouched", () => {
        const s = run(Nav.collapseLeft({}));
        expect(s.leftSelected()).toBeUndefined();
        expect(s.left().hover).toBe(false);
      });
    });
  });

  describe("bottom navigation", () => {
    describe("selectBottom", () => {
      it("should show the bottom nav when hidden", () => {
        const s = run(Nav.selectBottom({}));
        expect(s.bottomVisible()).toBe(true);
        expect(s.bottom().hover).toBe(false);
      });

      it("should hide the bottom nav when already shown", () => {
        const s = run(Nav.selectBottom({}), Nav.selectBottom({}));
        expect(s.bottomVisible()).toBe(false);
      });

      it("should pin a hovered bottom nav instead of hiding it", () => {
        const s = run(Nav.startBottomHover({}), Nav.selectBottom({}));
        expect(s.bottomVisible()).toBe(true);
        expect(s.bottom().hover).toBe(false);
      });
    });

    describe("showBottom", () => {
      it("should pin the bottom nav regardless of prior state", () => {
        const s = run(Nav.startBottomHover({}), Nav.showBottom({}));
        expect(s.bottomVisible()).toBe(true);
        expect(s.bottom().hover).toBe(false);
      });
    });

    describe("toggleBottom", () => {
      it("should open the bottom nav as a hover from hidden", () => {
        const s = run(Nav.toggleBottom({}));
        expect(s.bottomVisible()).toBe(true);
        expect(s.bottom().hover).toBe(true);
      });

      it("should drop only the hover when toggled while hovered", () => {
        const s = run(Nav.toggleBottom({}), Nav.toggleBottom({}));
        expect(s.bottomVisible()).toBe(true);
        expect(s.bottom().hover).toBe(false);
      });

      it("should hide the bottom nav when toggled while pinned", () => {
        const s = run(Nav.showBottom({}), Nav.toggleBottom({}));
        expect(s.bottomVisible()).toBe(false);
      });
    });

    describe("startBottomHover", () => {
      it("should hover the bottom nav from hidden", () => {
        const s = run(Nav.startBottomHover({}));
        expect(s.bottomVisible()).toBe(true);
        expect(s.bottom().hover).toBe(true);
      });

      it("should not hover over a pinned bottom nav", () => {
        const s = run(Nav.showBottom({}), Nav.startBottomHover({}));
        expect(s.bottomVisible()).toBe(true);
        expect(s.bottom().hover).toBe(false);
      });
    });

    describe("stopBottomHover", () => {
      it("should clear a hovered bottom nav", () => {
        const s = run(Nav.startBottomHover({}), Nav.stopBottomHover({}));
        expect(s.bottomVisible()).toBe(false);
        expect(s.bottom().hover).toBe(false);
      });

      it("should leave a pinned bottom nav untouched", () => {
        const s = run(Nav.showBottom({}), Nav.stopBottomHover({}));
        expect(s.bottomVisible()).toBe(true);
        expect(s.bottom().hover).toBe(false);
      });
    });

    describe("resizeBottom", () => {
      it("should set the bottom size", () => {
        const s = run(Nav.resizeBottom({ size: 180 }));
        expect(s.bottom().size).toBe(180);
      });
    });

    describe("collapseBottom", () => {
      it("should hide a hovered bottom nav", () => {
        const s = run(Nav.startBottomHover({}), Nav.collapseBottom({}));
        expect(s.bottomVisible()).toBe(false);
        expect(s.bottom().hover).toBe(false);
      });

      it("should hide a pinned bottom nav", () => {
        const s = run(Nav.showBottom({}), Nav.collapseBottom({}));
        expect(s.bottomVisible()).toBe(false);
      });

      it("should leave an already-collapsed bottom nav hidden", () => {
        const s = run(Nav.collapseBottom({}));
        expect(s.bottomVisible()).toBe(false);
        expect(s.bottom().hover).toBe(false);
      });

      it("should stay collapsed when invoked repeatedly", () => {
        const s = run(
          Nav.showBottom({}),
          Nav.collapseBottom({}),
          Nav.collapseBottom({}),
        );
        expect(s.bottomVisible()).toBe(false);
      });
    });
  });

  describe("hideAll", () => {
    it("should collapse both the left and bottom navigation", () => {
      const s = run(Nav.pinLeft({ key: "a" }), Nav.showBottom({}), Nav.hideAll({}));
      expect(s.leftSelected()).toBeUndefined();
      expect(s.left().hover).toBe(false);
      expect(s.bottomVisible()).toBe(false);
      expect(s.bottom().hover).toBe(false);
    });
  });

  describe("MIDDLEWARE", () => {
    it("should inject the active window key when the payload omits one", () => {
      const s = run(Nav.selectLeft({ key: "a" }));
      expect(s.leftSelected()).toBe("a");
    });

    it("should leave an explicit foreign window key untouched", () => {
      const s = run(Nav.selectLeft({ windowKey: "explicit", key: "a" }));
      expect(s.leftSelected()).toBeUndefined();
    });
  });
});
