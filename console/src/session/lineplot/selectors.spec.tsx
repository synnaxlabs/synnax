// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { configureStore } from "@reduxjs/toolkit";
import { LinePlot as PLinePlot } from "@synnaxlabs/pluto";
import { act, renderHook } from "@testing-library/react";
import { type FC, type PropsWithChildren, type ReactElement } from "react";
import { Provider } from "react-redux";
import { describe, expect, it } from "vitest";

import { LinePlot } from "@/session/lineplot";

const KEY = "plot-1";

const customState = LinePlot.stateZ.parse({
  control: { hold: true, enableTooltip: false, clickMode: "measure" },
  toolbar: { activeTab: "annotations" },
  mode: "pan",
  measure: { mode: "two" },
  annotations: { visible: false },
  selectedRules: ["r1", "r2"],
  hiddenLines: ["l1"],
});

const storeState: LinePlot.StoreState = {
  [LinePlot.SLICE_NAME]: { version: 0, plots: { [KEY]: customState } },
};

const params = { state: storeState, key: KEY };

describe("lineplot selectors", () => {
  describe("selectSliceState", () => {
    it("should return the slice state", () => {
      expect(LinePlot.selectSliceState(storeState)).toBe(
        storeState[LinePlot.SLICE_NAME],
      );
    });
  });

  describe("selectState / selectOptional", () => {
    it("should return the state for the given key", () => {
      expect(LinePlot.selectState(params)).toEqual(customState);
    });

    it("should fall back to ZERO_STATE for an unknown key", () => {
      expect(LinePlot.selectState({ state: storeState, key: "absent" })).toEqual(
        LinePlot.ZERO_STATE,
      );
    });
  });

  describe("toolbar selectors", () => {
    it("should return the active toolbar tab", () => {
      expect(LinePlot.selectActiveToolbarTab(params)).toBe("annotations");
    });

    it("should return the toolbar state", () => {
      expect(LinePlot.selectToolbar(params)).toEqual(customState.toolbar);
    });
  });

  describe("control selectors", () => {
    it("should return the control state", () => {
      expect(LinePlot.selectControlState(params)).toEqual(customState.control);
    });
  });

  describe("viewport / measure / selection / rules", () => {
    it("should read the viewport mode", () => {
      expect(LinePlot.selectViewportMode(params)).toBe("pan");
    });

    it("should read the measure mode", () => {
      expect(LinePlot.selectMeasureMode(params)).toBe("two");
    });

    it("should read the selection", () => {
      expect(LinePlot.selectSelection(params)).toEqual(customState.selection);
    });

    it("should read the hidden lines", () => {
      expect(LinePlot.selectHiddenLines(params)).toEqual(["l1"]);
    });

    it("should read the annotations visibility", () => {
      expect(LinePlot.selectAnnotationsVisible(params)).toBe(false);
    });

    it("should read the selected rules", () => {
      expect(LinePlot.selectSelectedRules(params)).toEqual(["r1", "r2"]);
    });
  });
});

const storeWith = (slice: LinePlot.SliceState) =>
  configureStore({
    reducer: { [LinePlot.SLICE_NAME]: LinePlot.reducer },
    preloadedState: { [LinePlot.SLICE_NAME]: slice },
  });

const wrapperFor = (
  store: ReturnType<typeof storeWith>,
  key: string,
): FC<PropsWithChildren> => {
  const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
    <Provider store={store}>
      <PLinePlot.Scope.Provider value={key}>{children}</PLinePlot.Scope.Provider>
    </Provider>
  );
  Wrapper.displayName = "Wrapper";
  return Wrapper;
};

describe("lineplot selector hooks", () => {
  const store = (): ReturnType<typeof storeWith> =>
    storeWith({ version: 0, plots: { [KEY]: customState } });

  it("should resolve the key from the surrounding scope", () => {
    const { result } = renderHook(() => LinePlot.useSelect(), {
      wrapper: wrapperFor(store(), KEY),
    });
    expect(result.current).toEqual(customState);
  });

  it("should let an explicit key override the scope", () => {
    const { result } = renderHook(() => LinePlot.useSelect({ key: "absent" }), {
      wrapper: wrapperFor(store(), KEY),
    });
    expect(result.current).toEqual(LinePlot.ZERO_STATE);
  });

  it("should return the active toolbar tab", () => {
    const { result } = renderHook(() => LinePlot.useSelectActiveToolbarTab(), {
      wrapper: wrapperFor(store(), KEY),
    });
    expect(result.current).toBe("annotations");
  });

  it("should return the control state", () => {
    const { result } = renderHook(() => LinePlot.useSelectControlState(), {
      wrapper: wrapperFor(store(), KEY),
    });
    expect(result.current).toEqual(customState.control);
  });

  it("should return the selected rules", () => {
    const { result } = renderHook(() => LinePlot.useSelectSelectedRules(), {
      wrapper: wrapperFor(store(), KEY),
    });
    expect(result.current).toEqual(["r1", "r2"]);
  });
});

describe("lineplot selector stability under dispatch", () => {
  const store = (): ReturnType<typeof storeWith> =>
    storeWith({ version: 0, plots: { [KEY]: customState } });

  it("should keep a stable reference when an unrelated field changes", () => {
    const s = store();
    const { result } = renderHook(() => LinePlot.useSelectToolbar(), {
      wrapper: wrapperFor(s, KEY),
    });
    const first = result.current;
    act(() => {
      s.dispatch(LinePlot.setMeasureMode({ key: KEY, mode: "one" }));
    });
    expect(result.current).toBe(first);
  });

  it("should return a new reference when the tracked field changes", () => {
    const s = store();
    const { result } = renderHook(() => LinePlot.useSelectToolbar(), {
      wrapper: wrapperFor(s, KEY),
    });
    const first = result.current;
    act(() => {
      s.dispatch(LinePlot.setActiveToolbarTab({ key: KEY, tab: "lines" }));
    });
    expect(result.current).not.toBe(first);
    expect(result.current.activeTab).toBe("lines");
  });

  it("should re-point the selector when its key dependency changes", () => {
    const s = storeWith({
      version: 0,
      plots: {
        [KEY]: customState,
        "plot-2": LinePlot.stateZ.parse({ toolbar: { activeTab: "axes" } }),
      },
    });
    const { result, rerender } = renderHook(
      ({ key }: { key: string }) => LinePlot.useSelectActiveToolbarTab({ key }),
      { wrapper: wrapperFor(s, KEY), initialProps: { key: KEY } },
    );
    expect(result.current).toBe("annotations");
    rerender({ key: "plot-2" });
    expect(result.current).toBe("axes");
  });
});
