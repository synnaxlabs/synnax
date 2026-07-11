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
import { beforeEach, describe, expect, it } from "vitest";

import { Table } from "@/session/table";

const storeWith = (slice: Table.SliceState) =>
  configureStore({
    reducer: { [Table.SLICE_NAME]: Table.reducer },
    preloadedState: { [Table.SLICE_NAME]: slice },
  });

const KEY = "table-1";

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

const renderGetters = (store: ReturnType<typeof storeWith>, key: string = KEY) =>
  renderHook(
    () => ({
      state: Table.useGet(),
      exists: Table.useGetExists(),
      editable: Table.useGetEditable(),
      hideIndicators: Table.useGetHideIndicators(),
      selectedCellKeys: Table.useGetSelectedCellKeys(),
      lastSelected: Table.useGetLastSelected(),
    }),
    { wrapper: wrapperFor(store, key) },
  ).result.current;

describe("Table Slice", () => {
  let store: ReturnType<typeof storeWith>;

  beforeEach(() => {
    store = storeWith(Table.ZERO_SLICE_STATE);
  });

  describe("create", () => {
    it("should bootstrap session state from ZERO_STATE for the key", () => {
      const get = renderGetters(store);
      act(() => void store.dispatch(Table.create({ key: KEY })));
      expect(get.state()).toEqual(Table.ZERO_STATE);
    });

    it("should create multiple tables independently", () => {
      const get = renderGetters(store);
      act(() => {
        store.dispatch(Table.create({ key: "table-1" }));
        store.dispatch(Table.create({ key: "table-2" }));
      });
      expect(get.exists({ key: "table-1" })).toBe(true);
      expect(get.exists({ key: "table-2" })).toBe(true);
      expect(get.exists({ key: "absent" })).toBe(false);
    });

    it("should apply provided fields over the defaults", () => {
      const get = renderGetters(store);
      act(() => void store.dispatch(Table.create({ key: KEY, editable: false })));
      expect(get.editable()).toBe(false);
    });

    it("should not overwrite an existing entry", () => {
      const get = renderGetters(store);
      act(() => {
        store.dispatch(Table.create({ key: KEY }));
        store.dispatch(Table.setEditable({ key: KEY, editable: false }));
        store.dispatch(Table.create({ key: KEY }));
      });
      expect(get.editable()).toBe(false);
    });
  });

  describe("setSelectedCells", () => {
    it("should set the selected cells", () => {
      const get = renderGetters(store);
      act(() => {
        store.dispatch(Table.create({ key: KEY }));
        store.dispatch(Table.setSelectedCells({ key: KEY, cells: ["a", "b"] }));
      });
      expect(get.selectedCellKeys()).toEqual(["a", "b"]);
    });

    it("should set the anchor when provided", () => {
      const get = renderGetters(store);
      act(() => {
        store.dispatch(Table.create({ key: KEY }));
        store.dispatch(
          Table.setSelectedCells({ key: KEY, cells: ["a", "b"], anchor: "a" }),
        );
      });
      expect(get.lastSelected()).toBe("a");
    });

    it("should leave the anchor untouched when omitted", () => {
      const get = renderGetters(store);
      act(() => {
        store.dispatch(Table.create({ key: KEY }));
        store.dispatch(Table.setSelectedCells({ key: KEY, cells: ["a"], anchor: "a" }));
        store.dispatch(Table.setSelectedCells({ key: KEY, cells: ["b"] }));
      });
      expect(get.lastSelected()).toBe("a");
    });

    it("should lazily create the entry when the key does not exist", () => {
      const get = renderGetters(store);
      act(
        () => void store.dispatch(Table.setSelectedCells({ key: KEY, cells: ["a"] })),
      );
      expect(get.selectedCellKeys()).toEqual(["a"]);
    });
  });

  describe("setEditable", () => {
    it("should toggle editing when no value is provided", () => {
      const get = renderGetters(store);
      act(() => {
        store.dispatch(Table.create({ key: KEY, editable: true }));
        store.dispatch(Table.setEditable({ key: KEY }));
      });
      expect(get.editable()).toBe(false);
    });

    it("should clear the selection when editing is disabled", () => {
      const get = renderGetters(store);
      act(() => {
        store.dispatch(Table.create({ key: KEY }));
        store.dispatch(Table.setSelectedCells({ key: KEY, cells: ["a"], anchor: "a" }));
        store.dispatch(Table.setEditable({ key: KEY, editable: false }));
      });
      expect(get.selectedCellKeys()).toEqual([]);
      expect(get.lastSelected()).toBeNull();
    });
  });

  describe("setHideIndicators", () => {
    it("should set the hide indicators flag", () => {
      const get = renderGetters(store);
      act(() => {
        store.dispatch(Table.create({ key: KEY }));
        store.dispatch(Table.setHideIndicators({ key: KEY, hideIndicators: true }));
      });
      expect(get.hideIndicators()).toBe(true);
    });

    it("should toggle the flag when no value is provided", () => {
      const get = renderGetters(store);
      act(() => void store.dispatch(Table.create({ key: KEY })));
      act(() => void store.dispatch(Table.setHideIndicators({ key: KEY })));
      expect(get.hideIndicators()).toBe(true);
      act(() => void store.dispatch(Table.setHideIndicators({ key: KEY })));
      expect(get.hideIndicators()).toBe(false);
    });
  });

  describe("remove", () => {
    it("should remove a table by key", () => {
      const get = renderGetters(store);
      act(() => void store.dispatch(Table.create({ key: KEY })));
      expect(get.exists()).toBe(true);
      act(() => void store.dispatch(Table.remove({ keys: [KEY] })));
      expect(get.exists()).toBe(false);
    });

    it("should remove multiple tables at once", () => {
      const get = renderGetters(store);
      act(() => {
        store.dispatch(Table.create({ key: "table-1" }));
        store.dispatch(Table.create({ key: "table-2" }));
        store.dispatch(Table.remove({ keys: ["table-1", "table-2"] }));
      });
      expect(get.exists({ key: "table-1" })).toBe(false);
      expect(get.exists({ key: "table-2" })).toBe(false);
    });

    it("should ignore keys that do not exist", () => {
      const get = renderGetters(store);
      act(() => {
        store.dispatch(Table.create({ key: KEY }));
        store.dispatch(Table.remove({ keys: ["absent"] }));
      });
      expect(get.state()).toEqual(Table.ZERO_STATE);
    });
  });

  describe("stateZ schema", () => {
    it("should accept the zero state", () => {
      expect(() => Table.stateZ.parse(Table.ZERO_STATE)).not.toThrow();
    });

    it("should apply defaults when fields are missing", () => {
      const parsed = Table.stateZ.parse({});
      expect(parsed.editable).toBe(true);
      expect(parsed.selectedCells).toEqual([]);
      expect(parsed.lastSelected).toBeNull();
      expect(parsed.hideIndicators).toBe(false);
    });
  });

  describe("sliceStateZ schema", () => {
    it("should default the slice version to 0", () => {
      expect(Table.sliceStateZ.parse({}).version).toBe(0);
      expect(Table.ZERO_SLICE_STATE.version).toBe(0);
    });
  });

  describe("purgeState", () => {
    it("should reset the cell selection", () => {
      const state = Table.stateZ.parse({
        selectedCells: ["a", "b"],
        lastSelected: "b",
      });
      const purged = Table.purgeState(state);
      expect(purged.selectedCells).toEqual([]);
      expect(purged.lastSelected).toBeNull();
    });

    it("should leave other fields untouched", () => {
      const state = Table.stateZ.parse({ hideIndicators: true, editable: false });
      const purged = Table.purgeState(state);
      expect(purged.hideIndicators).toBe(true);
      expect(purged.editable).toBe(false);
    });
  });

  describe("purgeSliceState", () => {
    it("should reset the selection on every table in the slice", () => {
      const state = {
        [Table.SLICE_NAME]: {
          version: 0 as const,
          tables: {
            "table-1": Table.stateZ.parse({ selectedCells: ["a"], lastSelected: "a" }),
            "table-2": Table.stateZ.parse({ selectedCells: ["b"], lastSelected: "b" }),
          },
        },
      };
      const purged = Table.purgeSliceState(state);
      expect(purged[Table.SLICE_NAME].tables["table-1"].selectedCells).toEqual([]);
      expect(purged[Table.SLICE_NAME].tables["table-2"].lastSelected).toBeNull();
    });
  });
});
