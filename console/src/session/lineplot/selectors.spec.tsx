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

const createCustomStore = () =>
  storeWith({ version: 0, plots: { [KEY]: customState } });

describe("lineplot selector hooks", () => {
  it("should resolve the key from the surrounding scope", () => {
    const { result } = renderHook(() => LinePlot.useSelect(), {
      wrapper: wrapperFor(createCustomStore(), KEY),
    });
    expect(result.current).toEqual(customState);
  });

  it("should let an explicit key override the scope", () => {
    const { result } = renderHook(() => LinePlot.useSelect({ key: "absent" }), {
      wrapper: wrapperFor(createCustomStore(), KEY),
    });
    expect(result.current).toEqual(LinePlot.ZERO_STATE);
  });

  it("should read the active toolbar tab", () => {
    const { result } = renderHook(() => LinePlot.useSelectActiveToolbarTab(), {
      wrapper: wrapperFor(createCustomStore(), KEY),
    });
    expect(result.current).toBe("annotations");
  });

  it("should read the control state", () => {
    const { result } = renderHook(() => LinePlot.useSelectControlState(), {
      wrapper: wrapperFor(createCustomStore(), KEY),
    });
    expect(result.current).toEqual(customState.control);
  });

  it("should read the selected rules", () => {
    const { result } = renderHook(() => LinePlot.useSelectSelectedRules(), {
      wrapper: wrapperFor(createCustomStore(), KEY),
    });
    expect(result.current).toEqual(["r1", "r2"]);
  });
});

describe("lineplot selector stability under dispatch", () => {
  it("should keep a stable reference when an unrelated field changes", () => {
    const store = createCustomStore();
    const { result } = renderHook(() => LinePlot.useSelectToolbar(), {
      wrapper: wrapperFor(store, KEY),
    });
    const first = result.current;
    act(() => {
      store.dispatch(LinePlot.setMeasureMode({ key: KEY, mode: "one" }));
    });
    expect(result.current).toBe(first);
  });

  it("should return a new reference when the tracked field changes", () => {
    const store = createCustomStore();
    const { result } = renderHook(() => LinePlot.useSelectToolbar(), {
      wrapper: wrapperFor(store, KEY),
    });
    const first = result.current;
    act(() => {
      store.dispatch(LinePlot.setActiveToolbarTab({ key: KEY, tab: "lines" }));
    });
    expect(result.current).not.toBe(first);
    expect(result.current.activeTab).toBe("lines");
  });

  it("should re-point the selector when its key dependency changes", () => {
    const store = storeWith({
      version: 0,
      plots: {
        [KEY]: customState,
        "plot-2": LinePlot.stateZ.parse({ toolbar: { activeTab: "axes" } }),
      },
    });
    const { result, rerender } = renderHook(
      ({ key }: { key: string }) => LinePlot.useSelectActiveToolbarTab({ key }),
      { wrapper: wrapperFor(store, KEY), initialProps: { key: KEY } },
    );
    expect(result.current).toBe("annotations");
    rerender({ key: "plot-2" });
    expect(result.current).toBe("axes");
  });
});

describe("lineplot getters", () => {
  it("should read a plot's state on demand across dispatches", () => {
    const store = storeWith(LinePlot.ZERO_SLICE_STATE);
    const { result } = renderHook(() => LinePlot.useGet(), {
      wrapper: wrapperFor(store, KEY),
    });
    const get = result.current;
    expect(get()).toEqual(LinePlot.ZERO_STATE);
    act(() => {
      store.dispatch(LinePlot.create({ key: KEY }));
      store.dispatch(LinePlot.setActiveToolbarTab({ key: KEY, tab: "lines" }));
    });
    expect(get().toolbar.activeTab).toBe("lines");
  });

  it("should reflect dispatches through an explicit key override", () => {
    const store = storeWith(LinePlot.ZERO_SLICE_STATE);
    const { result } = renderHook(() => LinePlot.useGetActiveToolbarTab(), {
      wrapper: wrapperFor(store, "absent"),
    });
    const get = result.current;
    expect(get({ key: KEY })).toBe("data");
    act(() => {
      store.dispatch(LinePlot.create({ key: KEY }));
      store.dispatch(LinePlot.setActiveToolbarTab({ key: KEY, tab: "axes" }));
    });
    expect(get({ key: KEY })).toBe("axes");
  });

  it("should resolve the key from scope and allow an explicit override", () => {
    const { result } = renderHook(() => LinePlot.useGetActiveToolbarTab(), {
      wrapper: wrapperFor(createCustomStore(), "absent"),
    });
    expect(result.current()).toBe("data");
    expect(result.current({ key: KEY })).toBe("annotations");
  });
});
