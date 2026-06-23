// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { configureStore } from "@reduxjs/toolkit";
import { beforeEach, describe, expect, it } from "vitest";

import { Table } from "@/layered/session/table";

const storeWith = (slice: Table.SliceState) =>
  configureStore({
    reducer: { [Table.SLICE_NAME]: Table.reducer },
    preloadedState: { [Table.SLICE_NAME]: slice },
  });

const KEY = "table-1";

describe("Table Slice", () => {
  let store: ReturnType<typeof storeWith>;

  beforeEach(() => {
    store = storeWith(Table.ZERO_SLICE_STATE);
  });

  const select = <R>(
    selector: (params: Table.KeyedSelectorParams) => R,
    key: string = KEY,
  ): R => selector({ state: store.getState(), key });

  describe("create", () => {
    it("should bootstrap session state from ZERO_STATE for the key", () => {
      store.dispatch(Table.internalCreate({ key: KEY }));
      expect(select(Table.selectState)).toEqual(Table.ZERO_STATE);
    });

    it("should create multiple tables independently", () => {
      store.dispatch(Table.internalCreate({ key: "table-1" }));
      store.dispatch(Table.internalCreate({ key: "table-2" }));
      expect(Object.keys(Table.selectSliceState(store.getState()).tables)).toHaveLength(
        2,
      );
    });

    it("should apply provided fields over the defaults", () => {
      store.dispatch(Table.internalCreate({ key: KEY, editable: false }));
      expect(select(Table.selectEditable)).toBe(false);
    });

    it("should not overwrite an existing entry", () => {
      store.dispatch(Table.internalCreate({ key: KEY }));
      store.dispatch(Table.setEditable({ key: KEY, editable: false }));
      store.dispatch(Table.internalCreate({ key: KEY }));
      expect(select(Table.selectEditable)).toBe(false);
    });
  });

  describe("selectCells", () => {
    it("should replace the selection", () => {
      store.dispatch(Table.internalCreate({ key: KEY }));
      store.dispatch(
        Table.selectCells({ key: KEY, mode: "replace", cells: ["a", "b"] }),
      );
      expect(select(Table.selectSelectedCellKeys)).toEqual(["a", "b"]);
      expect(select(Table.selectLastSelected)).toBe("b");
    });

    it("should add to the existing selection without duplicates", () => {
      store.dispatch(Table.internalCreate({ key: KEY }));
      store.dispatch(Table.selectCells({ key: KEY, mode: "replace", cells: ["a"] }));
      store.dispatch(Table.selectCells({ key: KEY, mode: "add", cells: ["a", "b"] }));
      expect(select(Table.selectSelectedCellKeys)).toEqual(["a", "b"]);
    });

    it("should set cells directly for region mode", () => {
      store.dispatch(Table.internalCreate({ key: KEY }));
      store.dispatch(
        Table.selectCells({ key: KEY, mode: "region", cells: ["a", "b", "c"] }),
      );
      expect(select(Table.selectSelectedCellKeys)).toEqual(["a", "b", "c"]);
    });

    it("should honor an explicit anchor", () => {
      store.dispatch(Table.internalCreate({ key: KEY }));
      store.dispatch(
        Table.selectCells({
          key: KEY,
          mode: "replace",
          cells: ["a", "b"],
          anchor: "a",
        }),
      );
      expect(select(Table.selectLastSelected)).toBe("a");
    });

    it("should clear the selection on an empty replace", () => {
      store.dispatch(Table.internalCreate({ key: KEY }));
      store.dispatch(Table.selectCells({ key: KEY, mode: "replace", cells: ["a"] }));
      store.dispatch(Table.selectCells({ key: KEY, mode: "replace", cells: [] }));
      expect(select(Table.selectSelectedCellKeys)).toEqual([]);
      expect(select(Table.selectLastSelected)).toBeNull();
    });

    it("should ignore selection when the table is not editable", () => {
      store.dispatch(Table.internalCreate({ key: KEY, editable: false }));
      store.dispatch(Table.selectCells({ key: KEY, mode: "replace", cells: ["a"] }));
      expect(select(Table.selectSelectedCellKeys)).toEqual([]);
    });
  });

  describe("setSelectedCells", () => {
    it("should set the selected cells", () => {
      store.dispatch(Table.internalCreate({ key: KEY }));
      store.dispatch(Table.setSelectedCells({ key: KEY, cells: ["a", "b"] }));
      expect(select(Table.selectSelectedCellKeys)).toEqual(["a", "b"]);
    });

    it("should set the anchor when provided", () => {
      store.dispatch(Table.internalCreate({ key: KEY }));
      store.dispatch(
        Table.setSelectedCells({ key: KEY, cells: ["a", "b"], anchor: "a" }),
      );
      expect(select(Table.selectLastSelected)).toBe("a");
    });

    it("should leave the anchor untouched when omitted", () => {
      store.dispatch(Table.internalCreate({ key: KEY }));
      store.dispatch(Table.setSelectedCells({ key: KEY, cells: ["a"], anchor: "a" }));
      store.dispatch(Table.setSelectedCells({ key: KEY, cells: ["b"] }));
      expect(select(Table.selectLastSelected)).toBe("a");
    });

    it("should lazily create the entry when the key does not exist", () => {
      store.dispatch(Table.setSelectedCells({ key: KEY, cells: ["a"] }));
      expect(select(Table.selectSelectedCellKeys)).toEqual(["a"]);
    });
  });

  describe("setEditable", () => {
    it("should toggle editing when no value is provided", () => {
      store.dispatch(Table.internalCreate({ key: KEY, editable: true }));
      store.dispatch(Table.setEditable({ key: KEY }));
      expect(select(Table.selectEditable)).toBe(false);
    });

    it("should clear the selection when editing is disabled", () => {
      store.dispatch(Table.internalCreate({ key: KEY }));
      store.dispatch(Table.setSelectedCells({ key: KEY, cells: ["a"], anchor: "a" }));
      store.dispatch(Table.setEditable({ key: KEY, editable: false }));
      expect(select(Table.selectSelectedCellKeys)).toEqual([]);
      expect(select(Table.selectLastSelected)).toBeNull();
    });
  });

  describe("setHideIndicators", () => {
    it("should set the hide indicators flag", () => {
      store.dispatch(Table.internalCreate({ key: KEY }));
      store.dispatch(Table.setHideIndicators({ key: KEY, hideIndicators: true }));
      expect(select(Table.selectHideIndicators)).toBe(true);
    });

    it("should toggle the flag when no value is provided", () => {
      store.dispatch(Table.internalCreate({ key: KEY }));
      store.dispatch(Table.setHideIndicators({ key: KEY }));
      expect(select(Table.selectHideIndicators)).toBe(true);
      store.dispatch(Table.setHideIndicators({ key: KEY }));
      expect(select(Table.selectHideIndicators)).toBe(false);
    });
  });

  describe("remove", () => {
    it("should remove a table by key", () => {
      store.dispatch(Table.internalCreate({ key: KEY }));
      store.dispatch(Table.remove({ keys: [KEY] }));
      expect(Table.selectSliceState(store.getState()).tables[KEY]).toBeUndefined();
    });

    it("should remove multiple tables at once", () => {
      store.dispatch(Table.internalCreate({ key: "table-1" }));
      store.dispatch(Table.internalCreate({ key: "table-2" }));
      store.dispatch(Table.remove({ keys: ["table-1", "table-2"] }));
      expect(Object.keys(Table.selectSliceState(store.getState()).tables)).toHaveLength(
        0,
      );
    });

    it("should ignore keys that do not exist", () => {
      store.dispatch(Table.internalCreate({ key: KEY }));
      store.dispatch(Table.remove({ keys: ["absent"] }));
      expect(select(Table.selectState)).toEqual(Table.ZERO_STATE);
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
