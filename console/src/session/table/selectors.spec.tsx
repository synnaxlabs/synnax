// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { configureStore } from "@reduxjs/toolkit";
import { Table as PTable } from "@synnaxlabs/pluto";
import { act, renderHook } from "@testing-library/react";
import { type FC, type PropsWithChildren, type ReactElement } from "react";
import { Provider } from "react-redux";
import { describe, expect, it } from "vitest";

import { Table } from "@/session/table";

const KEY = "table-1";

const customState = Table.stateZ.parse({
  editable: false,
  selectedCells: ["a", "b"],
  lastSelected: "b",
  hideIndicators: true,
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
      <PTable.Scope.Provider value={key}>{children}</PTable.Scope.Provider>
    </Provider>
  );
  Wrapper.displayName = "Wrapper";
  return Wrapper;
};

const seeded = () => storeWith({ version: 0, tables: { [KEY]: customState } });

describe("table selector hooks", () => {
  it("should resolve the key from the surrounding scope", () => {
    const { result } = renderHook(() => Table.useSelect(), {
      wrapper: wrapperFor(seeded(), KEY),
    });
    expect(result.current).toEqual(customState);
  });

  it("should let an explicit key override the scope", () => {
    const { result } = renderHook(() => Table.useSelect({ key: "absent" }), {
      wrapper: wrapperFor(seeded(), KEY),
    });
    expect(result.current).toEqual(Table.ZERO_STATE);
  });

  it("should return the optional entry when present and undefined when absent", () => {
    const store = seeded();
    const { result } = renderHook(() => Table.useSelectOptional(), {
      wrapper: wrapperFor(store, KEY),
    });
    expect(result.current).toEqual(customState);
    const { result: absent } = renderHook(() => Table.useSelectOptional(), {
      wrapper: wrapperFor(store, "absent"),
    });
    expect(absent.current).toBeUndefined();
  });

  it("should report existence", () => {
    const { result } = renderHook(() => Table.useSelectExists(), {
      wrapper: wrapperFor(seeded(), KEY),
    });
    expect(result.current).toBe(true);
  });

  it("should return the editable flag", () => {
    const { result } = renderHook(() => Table.useSelectEditable(), {
      wrapper: wrapperFor(seeded(), KEY),
    });
    expect(result.current).toBe(false);
  });

  it("should return the hide indicators flag", () => {
    const { result } = renderHook(() => Table.useSelectHideIndicators(), {
      wrapper: wrapperFor(seeded(), KEY),
    });
    expect(result.current).toBe(true);
  });

  it("should return the selected cell keys", () => {
    const { result } = renderHook(() => Table.useSelectSelectedCellKeys(), {
      wrapper: wrapperFor(seeded(), KEY),
    });
    expect(result.current).toEqual(["a", "b"]);
  });

  it("should return the last-selected cell key", () => {
    const { result } = renderHook(() => Table.useSelectLastSelected(), {
      wrapper: wrapperFor(seeded(), KEY),
    });
    expect(result.current).toBe("b");
  });
});

describe("table selector stability under dispatch", () => {
  it("should keep a stable reference when an unrelated field changes", () => {
    const s = seeded();
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
    const s = seeded();
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
    const s = seeded();
    const { result } = renderHook(() => Table.useSelectSelectedCellKeys(), {
      wrapper: wrapperFor(s, KEY),
    });
    const first = result.current;
    act(() => {
      s.dispatch(Table.create({ key: "table-2" }));
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

describe("table getters", () => {
  it("should read a table's state on demand across dispatches", () => {
    const store = storeWith(Table.ZERO_SLICE_STATE);
    const { result } = renderHook(() => Table.useGet(), {
      wrapper: wrapperFor(store, KEY),
    });
    const get = result.current;
    expect(get()).toEqual(Table.ZERO_STATE);
    act(() => {
      store.dispatch(Table.create({ key: KEY }));
      store.dispatch(Table.setSelectedCells({ key: KEY, cells: ["c"] }));
    });
    expect(get().selectedCells).toEqual(["c"]);
  });

  it("should resolve the key from scope and allow an explicit override", () => {
    const { result } = renderHook(() => Table.useGetSelectedCellKeys(), {
      wrapper: wrapperFor(seeded(), "absent"),
    });
    expect(result.current()).toEqual([]);
    expect(result.current({ key: KEY })).toEqual(["a", "b"]);
  });

  it("should report existence on demand", () => {
    const store = storeWith(Table.ZERO_SLICE_STATE);
    const { result } = renderHook(() => Table.useGetExists(), {
      wrapper: wrapperFor(store, KEY),
    });
    const get = result.current;
    expect(get()).toBe(false);
    act(() => {
      store.dispatch(Table.create({ key: KEY }));
    });
    expect(get()).toBe(true);
    expect(get({ key: "absent" })).toBe(false);
  });
});
