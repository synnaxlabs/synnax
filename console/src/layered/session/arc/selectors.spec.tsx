// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { configureStore } from "@reduxjs/toolkit";
import { Arc as PlutoArc } from "@synnaxlabs/pluto";
import { act, renderHook } from "@testing-library/react";
import { type FC, type PropsWithChildren, type ReactElement } from "react";
import { Provider } from "react-redux";
import { describe, expect, it } from "vitest";

import { Arc } from "@/layered/session/arc";

const KEY = "arc-1";

const customState = Arc.stateZ.parse({
  graph: {
    editable: false,
    fitViewOnResize: true,
    viewport: { position: { x: 7, y: 8 }, zoom: 2, mode: "pan" },
    selected: ["a", "b"],
  },
  toolbar: { selectedTab: "properties" },
});

const storeState: Arc.StoreState = {
  [Arc.SLICE_NAME]: { version: 0, arcs: { [KEY]: customState } },
};

const params = { state: storeState, key: KEY };

describe("arc selectors", () => {
  describe("selectSliceState", () => {
    it("should return the slice state", () => {
      expect(Arc.selectSliceState(storeState)).toBe(storeState[Arc.SLICE_NAME]);
    });
  });

  describe("selectState", () => {
    it("should return the state for the given key", () => {
      expect(Arc.selectState(params)).toEqual(customState);
    });

    it("should fall back to ZERO_STATE for an unknown key", () => {
      expect(Arc.selectState({ state: storeState, key: "absent" })).toEqual(
        Arc.ZERO_STATE,
      );
    });
  });

  describe("selectGraph", () => {
    it("should return the graph state", () => {
      expect(Arc.selectGraph(params)).toEqual(customState.graph);
    });
  });

  describe("selectSelected", () => {
    it("should return the selected elements", () => {
      expect(Arc.selectSelected(params)).toEqual(["a", "b"]);
    });
  });

  describe("selectEditable", () => {
    it("should return the editable flag", () => {
      expect(Arc.selectEditable(params)).toBe(false);
    });
  });

  describe("selectViewport", () => {
    it("should return the viewport", () => {
      expect(Arc.selectViewport(params)).toEqual(customState.graph.viewport);
    });
  });

  describe("selectViewportMode", () => {
    it("should return the viewport mode", () => {
      expect(Arc.selectViewportMode(params)).toBe("pan");
    });
  });

  describe("selectToolbar", () => {
    it("should return the toolbar state", () => {
      expect(Arc.selectToolbar(params)).toEqual(customState.toolbar);
    });
  });
});

const storeWith = (slice: Arc.SliceState) =>
  configureStore({
    reducer: { [Arc.SLICE_NAME]: Arc.reducer },
    preloadedState: { [Arc.SLICE_NAME]: slice },
  });

const wrapperFor = (
  store: ReturnType<typeof storeWith>,
  key: string,
): FC<PropsWithChildren> => {
  const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
    <Provider store={store}>
      <PlutoArc.Scope.Provider value={key}>{children}</PlutoArc.Scope.Provider>
    </Provider>
  );
  Wrapper.displayName = "Wrapper";
  return Wrapper;
};

describe("arc selector hooks", () => {
  const store = (): ReturnType<typeof storeWith> =>
    storeWith({ version: 0, arcs: { [KEY]: customState } });

  it("should resolve the key from the surrounding scope", () => {
    const { result } = renderHook(() => Arc.useSelect(), {
      wrapper: wrapperFor(store(), KEY),
    });
    expect(result.current).toEqual(customState);
  });

  it("should let an explicit key override the scope", () => {
    const { result } = renderHook(() => Arc.useSelect({ key: "absent" }), {
      wrapper: wrapperFor(store(), KEY),
    });
    expect(result.current).toEqual(Arc.ZERO_STATE);
  });

  it("should return the selected elements", () => {
    const { result } = renderHook(() => Arc.useSelectSelected(), {
      wrapper: wrapperFor(store(), KEY),
    });
    expect(result.current).toEqual(["a", "b"]);
  });

  it("should return the editable flag", () => {
    const { result } = renderHook(() => Arc.useSelectEditable(), {
      wrapper: wrapperFor(store(), KEY),
    });
    expect(result.current).toBe(false);
  });

  it("should return the viewport", () => {
    const { result } = renderHook(() => Arc.useSelectViewport(), {
      wrapper: wrapperFor(store(), KEY),
    });
    expect(result.current).toEqual(customState.graph.viewport);
  });

  it("should return the viewport mode", () => {
    const { result } = renderHook(() => Arc.useSelectViewportMode(), {
      wrapper: wrapperFor(store(), KEY),
    });
    expect(result.current).toBe("pan");
  });

  it("should return the toolbar state", () => {
    const { result } = renderHook(() => Arc.useSelectToolbar(), {
      wrapper: wrapperFor(store(), KEY),
    });
    expect(result.current).toEqual(customState.toolbar);
  });
});

describe("arc selector stability under dispatch", () => {
  const store = (): ReturnType<typeof storeWith> =>
    storeWith({ version: 0, arcs: { [KEY]: customState } });

  it("should keep a stable reference when an unrelated field changes", () => {
    const s = store();
    const { result } = renderHook(() => Arc.useSelectToolbar(), {
      wrapper: wrapperFor(s, KEY),
    });
    const first = result.current;
    act(() => {
      s.dispatch(Arc.setViewportMode({ key: KEY, mode: "select" }));
    });
    expect(result.current).toBe(first);
  });

  it("should return a new reference when the tracked field changes", () => {
    const s = store();
    const { result } = renderHook(() => Arc.useSelectToolbar(), {
      wrapper: wrapperFor(s, KEY),
    });
    const first = result.current;
    act(() => {
      s.dispatch(Arc.selectToolbarTab({ key: KEY, tab: "stages" }));
    });
    expect(result.current).not.toBe(first);
    expect(result.current.selectedTab).toBe("stages");
  });

  it("should ignore changes to other arcs", () => {
    const s = store();
    const { result } = renderHook(() => Arc.useSelectToolbar(), {
      wrapper: wrapperFor(s, KEY),
    });
    const first = result.current;
    act(() => {
      s.dispatch(Arc.internalCreate({ key: "arc-2" }));
      s.dispatch(Arc.selectToolbarTab({ key: "arc-2", tab: "stages" }));
    });
    expect(result.current).toBe(first);
  });

  it("should re-point the selector when its key dependency changes", () => {
    const s = storeWith({
      version: 0,
      arcs: {
        [KEY]: customState,
        "arc-2": Arc.stateZ.parse({ toolbar: { selectedTab: "stages" } }),
      },
    });
    const { result, rerender } = renderHook(
      ({ key }: { key: string }) => Arc.useSelectToolbar({ key }),
      { wrapper: wrapperFor(s, KEY), initialProps: { key: KEY } },
    );
    expect(result.current.selectedTab).toBe("properties");
    rerender({ key: "arc-2" });
    expect(result.current.selectedTab).toBe("stages");
  });
});
