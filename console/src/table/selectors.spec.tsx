// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { configureStore } from "@reduxjs/toolkit";
import { renderHook } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { Provider } from "react-redux";
import { describe, expect, it } from "vitest";

import {
  select,
  selectEditable,
  selectExists,
  selectHideIndicators,
  selectLastSelected,
  selectOptional,
  selectPendingUpload,
  selectSelectedCellKeys,
  selectSliceState,
  selectVersion,
  useSelectEditable,
  useSelectSelectedCellKeys,
} from "@/table/selectors";
import {
  reducer,
  SLICE_NAME,
  type SliceState,
  type State,
  type StoreState,
  ZERO_SLICE_STATE,
  ZERO_STATE,
} from "@/table/slice";

const buildState = (overrides: Partial<State>): StoreState => {
  const key = overrides.key ?? "table-1";
  const slice: SliceState = {
    ...ZERO_SLICE_STATE,
    tables: {
      ...ZERO_SLICE_STATE.tables,
      [key]: { ...ZERO_STATE, ...overrides, key },
    },
  };
  return { [SLICE_NAME]: slice };
};

const wrapperFor = (state: StoreState) => {
  const store = configureStore({
    reducer: { [SLICE_NAME]: reducer },
    preloadedState: state,
  });
  const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
    <Provider store={store}>{children}</Provider>
  );
  return Wrapper;
};

describe("table selectors", () => {
  describe("selectSliceState", () => {
    it("returns the slice state", () => {
      const state = buildState({});
      expect(selectSliceState(state)).toBe(state[SLICE_NAME]);
    });
  });

  describe("select / selectOptional", () => {
    it("returns the entry when present", () => {
      const state = buildState({});
      const entry = state[SLICE_NAME].tables["table-1"];
      expect(select(state, "table-1")).toBe(entry);
      expect(selectOptional(state, "table-1")).toBe(entry);
    });

    it("returns undefined from selectOptional when absent", () => {
      const state: StoreState = { [SLICE_NAME]: ZERO_SLICE_STATE };
      expect(selectOptional(state, "missing")).toBeUndefined();
    });
  });

  describe("selectExists", () => {
    it("should report whether the slice entry is present", () => {
      expect(selectExists(buildState({}), "table-1")).toBe(true);
      const state: StoreState = { [SLICE_NAME]: ZERO_SLICE_STATE };
      expect(selectExists(state, "missing")).toBe(false);
    });
  });

  describe("selectEditable", () => {
    it("returns the editable flag from the slice", () => {
      expect(selectEditable(buildState({ editable: true }), "table-1")).toBe(true);
      expect(selectEditable(buildState({ editable: false }), "table-1")).toBe(false);
    });
  });

  describe("selectSelectedCellKeys", () => {
    it("returns the selectedCells array", () => {
      const state = buildState({ selectedCells: ["a", "b"] });
      expect(selectSelectedCellKeys(state, "table-1")).toEqual(["a", "b"]);
    });
  });

  describe("selectLastSelected", () => {
    it("returns the last-selected cell key", () => {
      expect(selectLastSelected(buildState({ lastSelected: "a" }), "table-1")).toBe(
        "a",
      );
    });

    it("returns null when none is selected", () => {
      expect(selectLastSelected(buildState({}), "table-1")).toBeNull();
    });
  });

  describe("selectHideIndicators", () => {
    it("returns the hideIndicators flag from the slice", () => {
      expect(
        selectHideIndicators(buildState({ hideIndicators: true }), "table-1"),
      ).toBe(true);
      expect(
        selectHideIndicators(buildState({ hideIndicators: false }), "table-1"),
      ).toBe(false);
    });
  });

  describe("selectVersion", () => {
    it("returns the version, undefined when absent", () => {
      expect(selectVersion(buildState({}), "table-1")).toBe(ZERO_STATE.version);
      const state: StoreState = { [SLICE_NAME]: ZERO_SLICE_STATE };
      expect(selectVersion(state, "missing")).toBeUndefined();
    });
  });

  describe("selectPendingUpload", () => {
    it("returns the pending upload, undefined when absent", () => {
      const state = buildState({});
      expect(selectPendingUpload(state, "table-1")).toBe(
        state[SLICE_NAME].tables["table-1"].pendingUpload,
      );
      const emptyState: StoreState = { [SLICE_NAME]: ZERO_SLICE_STATE };
      expect(selectPendingUpload(emptyState, "missing")).toBeUndefined();
    });
  });

  describe("useSelectEditable", () => {
    it("reads through the Redux provider", () => {
      const state = buildState({ editable: true });
      const { result } = renderHook(() => useSelectEditable("table-1"), {
        wrapper: wrapperFor(state),
      });
      expect(result.current).toBe(true);
    });
  });

  describe("useSelectSelectedCellKeys", () => {
    it("exposes the selectedCells array", () => {
      const state = buildState({ selectedCells: ["a", "b"] });
      const { result } = renderHook(() => useSelectSelectedCellKeys("table-1"), {
        wrapper: wrapperFor(state),
      });
      expect(result.current).toEqual(["a", "b"]);
    });
  });
});
