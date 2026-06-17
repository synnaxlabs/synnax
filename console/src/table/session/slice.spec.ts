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

import {
  actions,
  reducer,
  select,
  selectEditable,
  selectHideIndicators,
  selectLastSelected,
  selectSelectedCellKeys,
  SLICE_NAME,
  type SliceState,
  sliceStateZ,
  stateZ,
  type StoreState,
} from "@/table/session/slice";

const storeWith = (slice: SliceState) =>
  configureStore({
    reducer: { [SLICE_NAME]: reducer },
    preloadedState: { [SLICE_NAME]: slice },
  });

describe("Table Slice", () => {
  let store: ReturnType<typeof storeWith>;

  beforeEach(() => {
    store = storeWith(sliceStateZ.parse({}));
  });

  const state = (): StoreState => store.getState();

  describe("create", () => {
    it("should bootstrap state with editable defaulting to true", () => {
      store.dispatch(actions.create({ key: "t1" }));
      expect(select(state(), "t1")).toEqual(stateZ.parse({}));
      expect(selectEditable(state(), "t1")).toBe(true);
    });

    it("should honor an explicit editable override", () => {
      store.dispatch(actions.create({ key: "t1", editable: false }));
      expect(selectEditable(state(), "t1")).toBe(false);
    });

    it("should create multiple tables independently", () => {
      store.dispatch(actions.create({ key: "t1" }));
      store.dispatch(actions.create({ key: "t2" }));
      expect(Object.keys(state()[SLICE_NAME].tables)).toHaveLength(2);
    });

    it("should not overwrite an existing entry", () => {
      store.dispatch(actions.create({ key: "t1" }));
      store.dispatch(actions.setHideIndicators({ key: "t1", hideIndicators: true }));
      store.dispatch(actions.create({ key: "t1" }));
      expect(selectHideIndicators(state(), "t1")).toBe(true);
    });
  });

  describe("remove", () => {
    it("should remove tables by key", () => {
      store.dispatch(actions.create({ key: "t1" }));
      store.dispatch(actions.create({ key: "t2" }));
      store.dispatch(actions.remove({ keys: ["t1", "t2"] }));
      expect(Object.keys(state()[SLICE_NAME].tables)).toHaveLength(0);
    });
  });

  describe("selectCells", () => {
    it("should replace the selection and set the anchor", () => {
      store.dispatch(actions.create({ key: "t1" }));
      store.dispatch(
        actions.selectCells({ key: "t1", mode: "replace", cells: ["a", "b"] }),
      );
      expect(selectSelectedCellKeys(state(), "t1")).toEqual(["a", "b"]);
      expect(selectLastSelected(state(), "t1")).toBe("b");
    });

    it("should add to the existing selection", () => {
      store.dispatch(actions.create({ key: "t1" }));
      store.dispatch(actions.selectCells({ key: "t1", mode: "replace", cells: ["a"] }));
      store.dispatch(actions.selectCells({ key: "t1", mode: "add", cells: ["b"] }));
      expect(selectSelectedCellKeys(state(), "t1")).toEqual(["a", "b"]);
    });

    it("should clear the selection on an empty replace", () => {
      store.dispatch(actions.create({ key: "t1" }));
      store.dispatch(actions.selectCells({ key: "t1", mode: "replace", cells: ["a"] }));
      store.dispatch(actions.selectCells({ key: "t1", mode: "replace", cells: [] }));
      expect(selectSelectedCellKeys(state(), "t1")).toEqual([]);
      expect(selectLastSelected(state(), "t1")).toBeNull();
    });

    it("should not select cells when the table is not editable", () => {
      store.dispatch(actions.create({ key: "t1", editable: false }));
      store.dispatch(actions.selectCells({ key: "t1", mode: "replace", cells: ["a"] }));
      expect(selectSelectedCellKeys(state(), "t1")).toEqual([]);
    });
  });

  describe("setSelectedCells", () => {
    it("should set the selection and anchor", () => {
      store.dispatch(actions.create({ key: "t1" }));
      store.dispatch(
        actions.setSelectedCells({ key: "t1", cells: ["a", "b"], anchor: "a" }),
      );
      expect(selectSelectedCellKeys(state(), "t1")).toEqual(["a", "b"]);
      expect(selectLastSelected(state(), "t1")).toBe("a");
    });
  });

  describe("setEditable", () => {
    it("should toggle editable when no value is given", () => {
      store.dispatch(actions.create({ key: "t1" }));
      store.dispatch(actions.setEditable({ key: "t1" }));
      expect(selectEditable(state(), "t1")).toBe(false);
    });

    it("should clear the selection when disabling editing", () => {
      store.dispatch(actions.create({ key: "t1" }));
      store.dispatch(actions.selectCells({ key: "t1", mode: "replace", cells: ["a"] }));
      store.dispatch(actions.setEditable({ key: "t1", editable: false }));
      expect(selectSelectedCellKeys(state(), "t1")).toEqual([]);
      expect(selectLastSelected(state(), "t1")).toBeNull();
    });
  });

  describe("setHideIndicators", () => {
    it("should toggle hideIndicators when no value is given", () => {
      store.dispatch(actions.create({ key: "t1" }));
      store.dispatch(actions.setHideIndicators({ key: "t1" }));
      expect(selectHideIndicators(state(), "t1")).toBe(true);
    });
  });

  describe("stateZ schema", () => {
    it("should accept the zero state", () => {
      expect(() => stateZ.parse({})).not.toThrow();
    });

    it("should default editable to true and hideIndicators to false", () => {
      const s = stateZ.parse({});
      expect(s.editable).toBe(true);
      expect(s.hideIndicators).toBe(false);
    });
  });
});
