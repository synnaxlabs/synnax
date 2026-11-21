// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { LinePlot as PLinePlot } from "@synnaxlabs/pluto";
import { act, renderHook } from "@testing-library/react";
import type { FC, PropsWithChildren, ReactElement } from "react";
import { Provider } from "react-redux";
import { describe, expect, it } from "vitest";

import { LinePlot } from "@/session/lineplot";
import { createSliceStore, inWindow } from "@/session/window/testutil";

const KEY = "plot-1";

const ZERO = LinePlot.ZERO_STATE;

const customState: LinePlot.State = {
  ...ZERO,
  control: { ...ZERO.control, hold: true, enableTooltip: false, clickMode: "measure" },
  toolbar: { ...ZERO.toolbar, activeTab: "annotations" },
  mode: "pan",
  measure: { ...ZERO.measure, mode: "two" },
  annotations: { ...ZERO.annotations, visible: false },
  selectedRules: ["r1", "r2"],
  hiddenLines: ["l1"],
};

const storeWith = (slice: LinePlot.SliceState) =>
  createSliceStore({
    name: LinePlot.SLICE_NAME,
    reducer: LinePlot.reducer,
    preloadedState: slice,
    middleware: LinePlot.MIDDLEWARE,
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
  storeWith({ version: 0, windows: inWindow({ [KEY]: customState }) });

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

describe("lineplot hidden items", () => {
  it("should read the items and the solo that setHidden sets", () => {
    const store = storeWith({ version: 0, windows: inWindow({ [KEY]: ZERO }) });
    const { result } = renderHook(
      () => ({
        hidden: LinePlot.useSelectHidden(),
        solo: LinePlot.useSelectSolo(),
      }),
      { wrapper: wrapperFor(store, KEY) },
    );
    const hidden = { lines: ["l1"], ranges: ["r1"], traces: ["l2/r2"] };
    const solo = {
      item: "range:r1",
      restore: { lines: [], ranges: ["r3"], traces: [] },
    };
    act(() => void store.dispatch(LinePlot.setHidden({ key: KEY, hidden, solo })));
    expect(result.current).toEqual({ hidden, solo });
  });
});

describe("lineplot selector stability under dispatch", () => {
  it("should keep a stable reference when an unrelated field changes", () => {
    const store = createCustomStore();
    const { result } = renderHook(() => LinePlot.useSelectControlState(), {
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
    const { result } = renderHook(() => LinePlot.useSelectControlState(), {
      wrapper: wrapperFor(store, KEY),
    });
    const first = result.current;
    act(() => {
      store.dispatch(LinePlot.setControlHold({ key: KEY, hold: false }));
    });
    expect(result.current).not.toBe(first);
    expect(result.current.hold).toBe(false);
  });

  it("should re-point the selector when its key dependency changes", () => {
    const store = storeWith({
      version: 0,
      windows: inWindow({
        [KEY]: customState,
        "plot-2": { ...ZERO, toolbar: { ...ZERO.toolbar, activeTab: "axes" } },
      }),
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
