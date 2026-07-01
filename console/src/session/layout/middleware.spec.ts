// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type MiddlewareAPI, type PayloadAction } from "@reduxjs/toolkit";
import { Drift, MAIN_WINDOW } from "@synnaxlabs/drift";
import { describe, expect, it, vi } from "vitest";

import { Layout } from "@/session/layout";

type StoreState = Layout.StoreState & Drift.StoreState;
type Mosaic = typeof Layout.ZERO_MOSAIC_STATE;

// Registers a window keyed by its own key so Drift.selectWindow resolves it directly.
const window = (key: string): Drift.WindowState =>
  ({ key }) as unknown as Drift.WindowState;

const mainDrift = (
  windows: Record<string, Drift.WindowState> = {},
): Drift.SliceState => ({
  ...Drift.ZERO_SLICE_STATE,
  windows: { ...Drift.ZERO_SLICE_STATE.windows, ...windows },
});

// A drift state whose active window is a non-main child window.
const childDrift = (): Drift.SliceState => ({
  ...Drift.ZERO_SLICE_STATE,
  label: "child",
  labelKeys: { ...Drift.ZERO_SLICE_STATE.labelKeys, child: "child" },
});

const layoutState = (
  layouts: Record<string, Layout.State> = {},
  mosaics: Record<string, Mosaic> = {},
): Layout.SliceState => ({
  ...Layout.ZERO_SLICE_STATE,
  layouts: { ...Layout.ZERO_SLICE_STATE.layouts, ...layouts },
  mosaics: { ...Layout.ZERO_SLICE_STATE.mosaics, ...mosaics },
});

const windowLayout = (key: string, windowKey = key): Layout.State => ({
  windowKey,
  key,
  type: "schematic",
  name: `Layout ${key}`,
  location: "window",
  window: { navTop: true },
});

const mosaicLayout = (key: string, windowKey: string): Layout.State => ({
  windowKey,
  key,
  type: "schematic",
  name: `Layout ${key}`,
  location: "mosaic",
});

const emptyMosaic = (): Mosaic => ({
  activeTab: null,
  focused: null,
  root: { key: 1, tabs: [] },
});

const filledMosaic = (): Mosaic => ({
  activeTab: "t",
  focused: null,
  root: { key: 1, tabs: [{ tabKey: "t", name: "T" }] },
});

const makeStore = (layout: Layout.SliceState, drift: Drift.SliceState) => {
  const dispatch = vi.fn();
  const store: MiddlewareAPI<typeof dispatch, StoreState> = {
    getState: () => ({
      [Layout.SLICE_NAME]: layout,
      [Drift.SLICE_NAME]: drift,
    }),
    dispatch,
  };
  return { store, dispatch };
};

const dispatchedOfType = (dispatch: ReturnType<typeof vi.fn>, type: string) =>
  dispatch.mock.calls
    .map(([a]) => a as PayloadAction<unknown>)
    .filter((a) => a.type === type);

describe("layout middleware", () => {
  describe("effectMiddleware", () => {
    it("should invoke effects only for declared action types", () => {
      const effect = vi.fn();
      const mw = Layout.effectMiddleware(["match/me"], [effect]);
      const { store } = makeStore(layoutState(), mainDrift());
      const next = vi.fn((a: unknown) => a);
      const dispatch = mw(store as never)(next);
      dispatch({ type: "ignore/me" });
      expect(effect).not.toHaveBeenCalled();
      dispatch({ type: "match/me" });
      expect(effect).toHaveBeenCalledOnce();
      expect(next).toHaveBeenCalledTimes(2);
    });

    it("should run the effect after next by default", () => {
      const order: string[] = [];
      const effect = vi.fn(() => order.push("effect"));
      const mw = Layout.effectMiddleware(["a"], [effect]);
      const { store } = makeStore(layoutState(), mainDrift());
      const next = vi.fn((a: unknown) => {
        order.push("next");
        return a;
      });
      mw(store as never)(next)({ type: "a" });
      expect(order).toEqual(["next", "effect"]);
    });

    it("should run the effect before next when before is true", () => {
      const order: string[] = [];
      const effect = vi.fn(() => order.push("effect"));
      const mw = Layout.effectMiddleware(["a"], [effect], true);
      const { store } = makeStore(layoutState(), mainDrift());
      const next = vi.fn((a: unknown) => {
        order.push("next");
        return a;
      });
      mw(store as never)(next)({ type: "a" });
      expect(order).toEqual(["effect", "next"]);
    });
  });

  describe("createWindowOnPlaceEffect", () => {
    it("should create a window when placing a window layout from the main window", () => {
      const { store, dispatch } = makeStore(layoutState(), mainDrift());
      Layout.createWindowOnPlaceEffect({
        store,
        action: Layout.place(windowLayout("win-1")),
      });
      const created = dispatchedOfType(dispatch, Drift.createWindow.type);
      expect(created).toHaveLength(1);
      expect(created[0].payload).toMatchObject({ key: "win-1", title: "Layout win-1" });
    });

    it("should not create a window for a mosaic-located layout", () => {
      const { store, dispatch } = makeStore(layoutState(), mainDrift());
      Layout.createWindowOnPlaceEffect({
        store,
        action: Layout.place(mosaicLayout("m-1", MAIN_WINDOW)),
      });
      expect(dispatchedOfType(dispatch, Drift.createWindow.type)).toHaveLength(0);
    });

    it("should not create a window when dispatched from a non-main window", () => {
      const { store, dispatch } = makeStore(layoutState(), childDrift());
      Layout.createWindowOnPlaceEffect({
        store,
        action: Layout.place(windowLayout("win-1")),
      });
      expect(dispatchedOfType(dispatch, Drift.createWindow.type)).toHaveLength(0);
    });
  });

  describe("closeWindowOnRemoveEffect", () => {
    it("should close windows for removed window-located layouts", () => {
      const layout = layoutState({ "win-1": windowLayout("win-1") });
      const { store, dispatch } = makeStore(layout, mainDrift());
      Layout.closeWindowOnRemoveEffect({
        store,
        action: Layout.remove({ keys: ["win-1"] }),
      });
      const closed = dispatchedOfType(dispatch, Drift.closeWindow.type);
      expect(closed).toHaveLength(1);
      expect(closed[0].payload).toMatchObject({ key: "win-1" });
    });

    it("should close windows for removed keys that no longer have a layout", () => {
      const { store, dispatch } = makeStore(layoutState(), mainDrift());
      Layout.closeWindowOnRemoveEffect({
        store,
        action: Layout.remove({ keys: ["ghost"] }),
      });
      expect(dispatchedOfType(dispatch, Drift.closeWindow.type)).toHaveLength(1);
    });

    it("should not close a window for a mosaic-located layout", () => {
      const layout = layoutState({ "m-1": mosaicLayout("m-1", MAIN_WINDOW) });
      const { store, dispatch } = makeStore(layout, mainDrift());
      Layout.closeWindowOnRemoveEffect({
        store,
        action: Layout.remove({ keys: ["m-1"] }),
      });
      expect(dispatchedOfType(dispatch, Drift.closeWindow.type)).toHaveLength(0);
    });

    it("should do nothing from a non-main window", () => {
      const layout = layoutState({ "win-1": windowLayout("win-1") });
      const { store, dispatch } = makeStore(layout, childDrift());
      Layout.closeWindowOnRemoveEffect({
        store,
        action: Layout.remove({ keys: ["win-1"] }),
      });
      expect(dispatch).not.toHaveBeenCalled();
    });
  });

  describe("createWindowsOnSetProjectEffect", () => {
    it("should create a window for each window-located layout except main", () => {
      const layout = layoutState({
        "win-1": windowLayout("win-1"),
        "win-2": windowLayout("win-2"),
      });
      const { store, dispatch } = makeStore(layout, mainDrift());
      Layout.createWindowsOnSetProjectEffect({
        store,
        action: { type: "project/select", payload: "p-1" },
      });
      const created = dispatchedOfType(dispatch, Drift.createWindow.type);
      expect(created.map((a) => (a.payload as { key: string }).key).sort()).toEqual([
        "win-1",
        "win-2",
      ]);
    });

    it("should not recreate the main window layout", () => {
      const { store, dispatch } = makeStore(layoutState(), mainDrift());
      Layout.createWindowsOnSetProjectEffect({
        store,
        action: { type: "project/select", payload: "p-1" },
      });
      expect(dispatchedOfType(dispatch, Drift.createWindow.type)).toHaveLength(0);
    });

    it("should do nothing from a non-main window", () => {
      const layout = layoutState({ "win-1": windowLayout("win-1") });
      const { store, dispatch } = makeStore(layout, childDrift());
      Layout.createWindowsOnSetProjectEffect({
        store,
        action: { type: "project/select", payload: "p-1" },
      });
      expect(dispatch).not.toHaveBeenCalled();
    });
  });

  describe("closeWindowOnEmptyMosaicEffect", () => {
    it("should close a non-main window whose mosaic is empty", () => {
      const layout = layoutState({}, { "mosaic-2": emptyMosaic() });
      const { store, dispatch } = makeStore(
        layout,
        mainDrift({ "mosaic-2": window("mosaic-2") }),
      );
      Layout.closeWindowOnEmptyMosaicEffect({
        store,
        action: { type: "layout/remove", payload: { keys: [] } },
      });
      const closed = dispatchedOfType(dispatch, Drift.closeWindow.type);
      expect(closed).toHaveLength(1);
      expect(closed[0].payload).toMatchObject({ key: "mosaic-2" });
    });

    it("should not close a window whose mosaic still has tabs", () => {
      const layout = layoutState({}, { "mosaic-2": filledMosaic() });
      const { store, dispatch } = makeStore(
        layout,
        mainDrift({ "mosaic-2": window("mosaic-2") }),
      );
      Layout.closeWindowOnEmptyMosaicEffect({
        store,
        action: { type: "layout/remove", payload: { keys: [] } },
      });
      expect(dispatchedOfType(dispatch, Drift.closeWindow.type)).toHaveLength(0);
    });

    it("should close orphaned mosaic windows that no longer have a mosaic", () => {
      const { store, dispatch } = makeStore(
        layoutState(),
        mainDrift({ "mosaicWindow-9": window("mosaicWindow-9") }),
      );
      Layout.closeWindowOnEmptyMosaicEffect({
        store,
        action: { type: "layout/remove", payload: { keys: [] } },
      });
      const closed = dispatchedOfType(dispatch, Drift.closeWindow.type);
      expect(closed).toHaveLength(1);
      expect(closed[0].payload).toMatchObject({ key: "mosaicWindow-9" });
    });

    it("should never close the main window even when its mosaic is empty", () => {
      const { store, dispatch } = makeStore(layoutState(), mainDrift());
      Layout.closeWindowOnEmptyMosaicEffect({
        store,
        action: { type: "layout/remove", payload: { keys: [] } },
      });
      expect(dispatchedOfType(dispatch, Drift.closeWindow.type)).toHaveLength(0);
    });

    it("should do nothing from a non-main window", () => {
      const layout = layoutState({}, { "mosaic-2": emptyMosaic() });
      const { store, dispatch } = makeStore(
        layout,
        (() => {
          const d = childDrift();
          d.windows = { ...d.windows, "mosaic-2": window("mosaic-2") };
          return d;
        })(),
      );
      Layout.closeWindowOnEmptyMosaicEffect({
        store,
        action: { type: "layout/remove", payload: { keys: [] } },
      });
      expect(dispatch).not.toHaveBeenCalled();
    });
  });

  // deleteLayoutsOnMosaicCloseEffect is not exported; exercise it through the composed
  // MIDDLEWARE. Only the close-window effect reacts to Drift.closeWindow, so running the
  // action through the whole chain isolates its behavior.
  describe("MIDDLEWARE (deleteLayoutsOnMosaicCloseEffect)", () => {
    it("should remove layouts belonging to a closed mosaic window", () => {
      const layout = layoutState({
        "content-1": mosaicLayout("content-1", "mosaicWindow-1"),
        "content-2": mosaicLayout("content-2", "mosaicWindow-1"),
        other: mosaicLayout("other", "mosaicWindow-2"),
      });
      const { store, dispatch } = makeStore(layout, mainDrift());
      const next = vi.fn((a: unknown) => a);
      // Run through every middleware in the chain; only the close-window effect reacts.
      Layout.MIDDLEWARE.forEach((mw) =>
        mw(store as never)(next)({
          type: Drift.closeWindow.type,
          payload: { key: "mosaicWindow-1" },
        }),
      );
      const removes = dispatchedOfType(dispatch, Layout.remove.type);
      expect(removes.length).toBeGreaterThanOrEqual(1);
      const removed = removes.flatMap((a) => (a.payload as { keys: string[] }).keys);
      expect(removed.sort()).toEqual(["content-1", "content-2"]);
    });

    it("should ignore closes of non-mosaic windows", () => {
      const layout = layoutState({
        "content-1": mosaicLayout("content-1", "some-window"),
      });
      const { store, dispatch } = makeStore(layout, mainDrift());
      const next = vi.fn((a: unknown) => a);
      Layout.MIDDLEWARE.forEach((mw) =>
        mw(store as never)(next)({
          type: Drift.closeWindow.type,
          payload: { key: "some-window" },
        }),
      );
      expect(dispatchedOfType(dispatch, Layout.remove.type)).toHaveLength(0);
    });
  });
});
