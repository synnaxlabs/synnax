// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { configureStore } from "@reduxjs/toolkit";
import { Arc as PArc } from "@synnaxlabs/pluto";
import { act, renderHook } from "@testing-library/react";
import { type FC, type PropsWithChildren, type ReactElement } from "react";
import { Provider } from "react-redux";
import { beforeEach, describe, expect, it } from "vitest";

import { Arc } from "@/session/arc";

const storeWith = (slice: Arc.SliceState) =>
  configureStore({
    reducer: { [Arc.SLICE_NAME]: Arc.reducer },
    preloadedState: { [Arc.SLICE_NAME]: slice },
  });

const KEY = "arc-1";

const wrapperFor = (
  store: ReturnType<typeof storeWith>,
  key: string,
): FC<PropsWithChildren> => {
  const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
    <Provider store={store}>
      <PArc.Scope.Provider value={key}>{children}</PArc.Scope.Provider>
    </Provider>
  );
  Wrapper.displayName = "Wrapper";
  return Wrapper;
};

describe("Arc Slice", () => {
  let store: ReturnType<typeof storeWith>;

  beforeEach(() => {
    store = storeWith(Arc.ZERO_SLICE_STATE);
  });

  const renderGetter = <G,>(useGetter: () => G, key: string = KEY): G =>
    renderHook(() => useGetter(), { wrapper: wrapperFor(store, key) }).result.current;

  describe("create", () => {
    it("should bootstrap session state from ZERO_STATE for the key", () => {
      const get = renderGetter(Arc.useGet);
      act(() => store.dispatch(Arc.create({ key: KEY })));
      expect(get()).toEqual(Arc.ZERO_STATE);
    });

    it("should default the toolbar tab to stages", () => {
      const getToolbar = renderGetter(Arc.useGetToolbar);
      act(() => store.dispatch(Arc.create({ key: KEY })));
      expect(getToolbar().selectedTab).toBe("stages");
    });

    it("should create multiple arcs independently", () => {
      const getSelected = renderGetter(Arc.useGetSelected);
      act(() => {
        store.dispatch(Arc.create({ key: "arc-1", graph: { selected: ["a"] } }));
        store.dispatch(Arc.create({ key: "arc-2" }));
      });
      expect(getSelected({ key: "arc-1" })).toEqual(["a"]);
      expect(getSelected({ key: "arc-2" })).toEqual([]);
    });

    it("should apply provided graph fields over the defaults", () => {
      const getEditable = renderGetter(Arc.useGetEditable);
      act(() => store.dispatch(Arc.create({ key: KEY, graph: { editable: false } })));
      expect(getEditable()).toBe(false);
    });

    it("should not overwrite an existing entry", () => {
      const getEditable = renderGetter(Arc.useGetEditable);
      act(() => {
        store.dispatch(Arc.create({ key: KEY }));
        store.dispatch(Arc.setEditable({ key: KEY, editable: false }));
        store.dispatch(Arc.create({ key: KEY }));
      });
      expect(getEditable()).toBe(false);
    });
  });

  describe("setSelected", () => {
    it("should set the per-arc selection", () => {
      const getSelected = renderGetter(Arc.useGetSelected);
      act(() => {
        store.dispatch(Arc.create({ key: KEY }));
        store.dispatch(Arc.setSelected({ key: KEY, selected: ["a", "b"] }));
      });
      expect(getSelected()).toEqual(["a", "b"]);
    });

    it("should switch the toolbar to properties when selecting", () => {
      const getToolbar = renderGetter(Arc.useGetToolbar);
      act(() => {
        store.dispatch(Arc.create({ key: KEY }));
        store.dispatch(Arc.setSelected({ key: KEY, selected: ["a"] }));
      });
      expect(getToolbar().selectedTab).toBe("properties");
    });

    it("should switch the toolbar back to stages when clearing", () => {
      const getToolbar = renderGetter(Arc.useGetToolbar);
      act(() => {
        store.dispatch(Arc.create({ key: KEY }));
        store.dispatch(Arc.setSelected({ key: KEY, selected: ["a"] }));
        store.dispatch(Arc.setSelected({ key: KEY, selected: [] }));
      });
      expect(getToolbar().selectedTab).toBe("stages");
    });

    it("should track selection and toolbar per arc independently", () => {
      const getSelected = renderGetter(Arc.useGetSelected);
      const getToolbar = renderGetter(Arc.useGetToolbar);
      act(() => {
        store.dispatch(Arc.create({ key: "arc-1", graph: { selected: ["a"] } }));
        store.dispatch(Arc.create({ key: "arc-2" }));
        store.dispatch(Arc.setSelected({ key: "arc-2", selected: ["b"] }));
      });
      expect(getSelected({ key: "arc-1" })).toEqual(["a"]);
      expect(getSelected({ key: "arc-2" })).toEqual(["b"]);
      expect(getToolbar({ key: "arc-2" }).selectedTab).toBe("properties");
    });

    it("should lazily create the entry when the key does not exist", () => {
      const getSelected = renderGetter(Arc.useGetSelected);
      act(() => store.dispatch(Arc.setSelected({ key: KEY, selected: ["a"] })));
      expect(getSelected()).toEqual(["a"]);
    });
  });

  describe("selectToolbarTab", () => {
    it("should set the active toolbar tab", () => {
      const getToolbar = renderGetter(Arc.useGetToolbar);
      act(() => {
        store.dispatch(Arc.create({ key: KEY }));
        store.dispatch(Arc.selectToolbarTab({ key: KEY, tab: "properties" }));
      });
      expect(getToolbar().selectedTab).toBe("properties");
    });

    it("should lazily create the entry when the key does not exist", () => {
      const getToolbar = renderGetter(Arc.useGetToolbar);
      act(() => store.dispatch(Arc.selectToolbarTab({ key: KEY, tab: "properties" })));
      expect(getToolbar().selectedTab).toBe("properties");
    });
  });

  describe("setViewport", () => {
    it("should merge the viewport over the existing one", () => {
      const getViewport = renderGetter(Arc.useGetViewport);
      const getViewportMode = renderGetter(Arc.useGetViewportMode);
      act(() => {
        store.dispatch(Arc.create({ key: KEY }));
        store.dispatch(
          Arc.setViewport({
            key: KEY,
            viewport: { position: { x: 5, y: 6 }, zoom: 2 },
          }),
        );
      });
      const viewport = getViewport();
      expect(viewport.position).toEqual({ x: 5, y: 6 });
      expect(viewport.zoom).toBe(2);
      expect(getViewportMode()).toBe(Arc.ZERO_STATE.graph.viewport.mode);
    });

    it("should lazily create the entry when the key does not exist", () => {
      const getViewport = renderGetter(Arc.useGetViewport);
      act(() =>
        store.dispatch(
          Arc.setViewport({
            key: KEY,
            viewport: { position: { x: 1, y: 2 }, zoom: 3 },
          }),
        ),
      );
      const viewport = getViewport();
      expect(viewport.position).toEqual({ x: 1, y: 2 });
      expect(viewport.zoom).toBe(3);
    });
  });

  describe("setViewportMode", () => {
    it("should set the per-arc viewport mode without touching position or zoom", () => {
      const getViewport = renderGetter(Arc.useGetViewport);
      const getViewportMode = renderGetter(Arc.useGetViewportMode);
      act(() => {
        store.dispatch(Arc.create({ key: KEY }));
        store.dispatch(Arc.setViewportMode({ key: KEY, mode: "pan" }));
      });
      expect(getViewportMode()).toBe("pan");
      expect(getViewport().position).toEqual(Arc.ZERO_STATE.graph.viewport.position);
    });
  });

  describe("setEditable", () => {
    it("should set the editable flag", () => {
      const getEditable = renderGetter(Arc.useGetEditable);
      act(() => {
        store.dispatch(Arc.create({ key: KEY, graph: { editable: false } }));
        store.dispatch(Arc.setEditable({ key: KEY, editable: true }));
      });
      expect(getEditable()).toBe(true);
    });

    it("should clear the selection when toggled", () => {
      const getSelected = renderGetter(Arc.useGetSelected);
      act(() => {
        store.dispatch(Arc.create({ key: KEY }));
        store.dispatch(Arc.setSelected({ key: KEY, selected: ["a"] }));
        store.dispatch(Arc.setEditable({ key: KEY, editable: false }));
      });
      expect(getSelected()).toEqual([]);
    });
  });

  describe("setFitViewOnResize", () => {
    it("should set the fit-view-on-resize flag", () => {
      const getFitViewOnResize = renderGetter(Arc.useGetFitViewOnResize);
      act(() => {
        store.dispatch(Arc.create({ key: KEY }));
        store.dispatch(Arc.setFitViewOnResize({ key: KEY, fitViewOnResize: true }));
      });
      expect(getFitViewOnResize()).toBe(true);
    });
  });

  describe("remove", () => {
    it("should remove an arc by key", () => {
      const get = renderGetter(Arc.useGet);
      act(() => {
        store.dispatch(Arc.create({ key: KEY, graph: { editable: false } }));
        store.dispatch(Arc.remove({ keys: [KEY] }));
      });
      expect(get()).toEqual(Arc.ZERO_STATE);
    });

    it("should remove multiple arcs at once", () => {
      const get = renderGetter(Arc.useGet);
      act(() => {
        store.dispatch(Arc.create({ key: "arc-1", graph: { editable: false } }));
        store.dispatch(Arc.create({ key: "arc-2", graph: { editable: false } }));
        store.dispatch(Arc.remove({ keys: ["arc-1", "arc-2"] }));
      });
      expect(get({ key: "arc-1" })).toEqual(Arc.ZERO_STATE);
      expect(get({ key: "arc-2" })).toEqual(Arc.ZERO_STATE);
    });

    it("should ignore keys that do not exist", () => {
      const getEditable = renderGetter(Arc.useGetEditable);
      act(() => {
        store.dispatch(Arc.create({ key: KEY, graph: { editable: false } }));
        store.dispatch(Arc.remove({ keys: ["absent"] }));
      });
      expect(getEditable()).toBe(false);
    });
  });

  describe("selectors", () => {
    it("should fall back to ZERO_STATE for an unknown key", () => {
      const get = renderGetter(Arc.useGet);
      expect(get()).toEqual(Arc.ZERO_STATE);
    });
  });

  describe("stateZ schema", () => {
    it("should accept the zero state", () => {
      expect(() => Arc.stateZ.parse(Arc.ZERO_STATE)).not.toThrow();
    });

    it("should apply prefaulted defaults when nested objects are missing", () => {
      const parsed = Arc.stateZ.parse({});
      expect(parsed.graph.editable).toBe(true);
      expect(parsed.graph.fitViewOnResize).toBe(false);
      expect(parsed.graph.selected).toEqual([]);
      expect(parsed.graph.viewport).toEqual({
        position: { x: 0, y: 0 },
        zoom: 1,
        mode: "select",
      });
      expect(parsed.toolbar.selectedTab).toBe("stages");
    });
  });

  describe("sliceStateZ schema", () => {
    it("should default the slice version to 0", () => {
      expect(Arc.sliceStateZ.parse({}).version).toBe(0);
      expect(Arc.ZERO_SLICE_STATE.version).toBe(0);
    });

    it("should default the arcs record to empty", () => {
      expect(Arc.sliceStateZ.parse({}).arcs).toEqual({});
    });
  });
});
