// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { configureStore } from "@reduxjs/toolkit";
import { Table as PlutoTable } from "@synnaxlabs/pluto";
import { act, renderHook } from "@testing-library/react";
import { type FC, type PropsWithChildren, type ReactElement } from "react";
import { Provider } from "react-redux";
import { describe, expect, it } from "vitest";

import { Table } from "@/layered/session/table";

const KEY = "table-1";

const customState = Table.stateZ.parse({
  editable: false,
  selectedCells: ["a", "b"],
  lastSelected: "b",
  hideIndicators: true,
});

const storeState: Table.StoreState = {
  [Table.SLICE_NAME]: { version: 0, tables: { [KEY]: customState } },
};

const params = { state: storeState, key: KEY };

describe("table selectors", () => {
  describe("selectSliceState", () => {
    it("should return the slice state", () => {
      expect(Table.selectSliceState(storeState)).toBe(storeState[Table.SLICE_NAME]);
    });
  });

  describe("selectState", () => {
    it("should return the state for the given key", () => {
      expect(Table.selectState(params)).toEqual(customState);
    });

    it("should fall back to ZERO_STATE for an unknown key", () => {
      expect(Table.selectState({ state: storeState, key: "absent" })).toEqual(
        Table.ZERO_STATE,
      );
    });
  });

  describe("selectOptional", () => {
    it("should return the entry when present", () => {
      expect(Table.selectOptional(params)).toBe(customState);
    });

    it("should return undefined when absent", () => {
      expect(
        Table.selectOptional({ state: storeState, key: "absent" }),
      ).toBeUndefined();
    });
  });

  describe("selectExists", () => {
    it("should report whether the slice entry is present", () => {
      expect(Table.selectExists(params)).toBe(true);
      expect(Table.selectExists({ state: storeState, key: "absent" })).toBe(false);
    });
  });

  describe("selectEditable", () => {
    it("should return the editable flag", () => {
      expect(Table.selectEditable(params)).toBe(false);
    });
  });

  describe("selectHideIndicators", () => {
    it("should return the hide indicators flag", () => {
      expect(Table.selectHideIndicators(params)).toBe(true);
    });
  });

  describe("selectSelectedCellKeys", () => {
    it("should return the selected cell keys", () => {
      expect(Table.selectSelectedCellKeys(params)).toEqual(["a", "b"]);
    });
  });

  describe("selectLastSelected", () => {
    it("should return the last-selected cell key", () => {
      expect(Table.selectLastSelected(params)).toBe("b");
    });

    it("should return null when none is selected", () => {
      expect(Table.selectLastSelected({ state: storeState, key: "absent" })).toBeNull();
    });
  });
});

const storeWith = (slice: Table.SliceState) =>
  configureStore({
    reducer: { [Table.SLICE_NAME]: Table.reducer },
    preloadedState: { [Table.SLICE_NAME]: slice },
  });

const wrapperFor = (
  store: ReturnType<typeof storeWith>,
  key: string,
): FC<PropsWithChildren> => {
  const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
    <Provider store={store}>
      <PlutoTable.Scope.Provider value={key}>{children}</PlutoTable.Scope.Provider>
    </Provider>
  );
  Wrapper.displayName = "Wrapper";
  return Wrapper;
};

describe("table selector hooks", () => {
  const store = (): ReturnType<typeof storeWith> =>
    storeWith({ version: 0, tables: { [KEY]: customState } });

  it("should resolve the key from the surrounding scope", () => {
    const { result } = renderHook(() => Table.useSelect(), {
      wrapper: wrapperFor(store(), KEY),
    });
    expect(result.current).toEqual(customState);
  });

  it("should let an explicit key override the scope", () => {
    const { result } = renderHook(() => Table.useSelect({ key: "absent" }), {
      wrapper: wrapperFor(store(), KEY),
    });
    expect(result.current).toEqual(Table.ZERO_STATE);
  });

  it("should report existence", () => {
    const { result } = renderHook(() => Table.useSelectExists(), {
      wrapper: wrapperFor(store(), KEY),
    });
    expect(result.current).toBe(true);
  });

  it("should return the editable flag", () => {
    const { result } = renderHook(() => Table.useSelectEditable(), {
      wrapper: wrapperFor(store(), KEY),
    });
    expect(result.current).toBe(false);
  });

  it("should return the hide indicators flag", () => {
    const { result } = renderHook(() => Table.useSelectHideIndicators(), {
      wrapper: wrapperFor(store(), KEY),
    });
    expect(result.current).toBe(true);
  });

  it("should return the selected cell keys", () => {
    const { result } = renderHook(() => Table.useSelectSelectedCellKeys(), {
      wrapper: wrapperFor(store(), KEY),
    });
    expect(result.current).toEqual(["a", "b"]);
  });

  it("should return the last-selected cell key", () => {
    const { result } = renderHook(() => Table.useSelectLastSelected(), {
      wrapper: wrapperFor(store(), KEY),
    });
    expect(result.current).toBe("b");
  });
});

describe("table selector stability under dispatch", () => {
  const store = (): ReturnType<typeof storeWith> =>
    storeWith({ version: 0, tables: { [KEY]: customState } });

  it("should keep a stable reference when an unrelated field changes", () => {
    const s = store();
    const { result } = renderHook(() => Table.useSelectSelectedCellKeys(), {
      wrapper: wrapperFor(s, KEY),
    });
    const first = result.current;
    act(() => {
      s.dispatch(Table.setHideIndicators({ key: KEY, hideIndicators: false }));
    });
    expect(result.current).toBe(first);
  });

  it("should return a new reference when the tracked field changes", () => {
    const s = store();
    const { result } = renderHook(() => Table.useSelectSelectedCellKeys(), {
      wrapper: wrapperFor(s, KEY),
    });
    const first = result.current;
    act(() => {
      s.dispatch(Table.setSelectedCells({ key: KEY, cells: ["c"] }));
    });
    expect(result.current).not.toBe(first);
    expect(result.current).toEqual(["c"]);
  });

  it("should ignore changes to other tables", () => {
    const s = store();
    const { result } = renderHook(() => Table.useSelectSelectedCellKeys(), {
      wrapper: wrapperFor(s, KEY),
    });
    const first = result.current;
    act(() => {
      s.dispatch(Table.internalCreate({ key: "table-2" }));
      s.dispatch(Table.setSelectedCells({ key: "table-2", cells: ["z"] }));
    });
    expect(result.current).toBe(first);
  });

  it("should re-point the selector when its key dependency changes", () => {
    const s = storeWith({
      version: 0,
      tables: {
        [KEY]: customState,
        "table-2": Table.stateZ.parse({ selectedCells: ["z"] }),
      },
    });
    const { result, rerender } = renderHook(
      ({ key }: { key: string }) => Table.useSelectSelectedCellKeys({ key }),
      { wrapper: wrapperFor(s, KEY), initialProps: { key: KEY } },
    );
    expect(result.current).toEqual(["a", "b"]);
    rerender({ key: "table-2" });
    expect(result.current).toEqual(["z"]);
  });
});
