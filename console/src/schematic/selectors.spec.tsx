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
  selectActiveToolbarTab,
  selectAuthority,
  selectControlStatus,
  selectEditable,
  selectExists,
  selectFitViewOnResize,
  selectLegend,
  selectLegendVisible,
  selectOptional,
  selectSelected,
  selectSelectedSymbolGroup,
  selectSliceState,
  selectToolbar,
  selectViewport,
  useSelectEditable,
} from "@/schematic/selectors";
import {
  reducer,
  SLICE_NAME,
  type State,
  type StoreState,
  ZERO_SLICE_STATE,
  ZERO_STATE,
} from "@/schematic/slice";

const KEY = "schematic-1";

const entry: State = {
  ...ZERO_STATE,
  editable: true,
  fitViewOnResize: true,
  selected: ["element-1"],
  controlStatus: "acquired",
  authority: 5,
  legend: { ...ZERO_STATE.legend, visible: false },
  toolbar: { activeTab: "properties", selectedSymbolGroup: "valves" },
};

const state: StoreState = {
  [SLICE_NAME]: { ...ZERO_SLICE_STATE, schematics: { [KEY]: entry } },
};

const empty: StoreState = { [SLICE_NAME]: ZERO_SLICE_STATE };

const wrapperFor = (s: StoreState) => {
  const store = configureStore({
    reducer: { [SLICE_NAME]: reducer },
    preloadedState: s,
  });
  const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
    <Provider store={store}>{children}</Provider>
  );
  return Wrapper;
};

describe("schematic selectors", () => {
  describe("selectSliceState", () => {
    it("returns the slice state", () => {
      expect(selectSliceState(state)).toBe(state[SLICE_NAME]);
    });
  });

  describe("select / selectOptional", () => {
    it("returns the entry when present", () => {
      expect(select(state, KEY)).toBe(entry);
      expect(selectOptional(state, KEY)).toBe(entry);
    });

    it("returns undefined from selectOptional when absent", () => {
      expect(selectOptional(empty, "absent")).toBeUndefined();
    });
  });

  describe("selectExists", () => {
    it("should report whether the slice entry is present", () => {
      expect(selectExists(state, KEY)).toBe(true);
      expect(selectExists(state, "absent")).toBe(false);
    });
  });

  describe("present slice entry", () => {
    it("should read the toolbar from the entry", () => {
      expect(selectToolbar(state, KEY)).toBe(entry.toolbar);
    });

    it("should read the active toolbar tab from the entry", () => {
      expect(selectActiveToolbarTab(state, KEY)).toBe("properties");
    });

    it("should read the selected symbol group from the entry", () => {
      expect(selectSelectedSymbolGroup(state, KEY)).toBe("valves");
    });

    it("should read editable from the entry", () => {
      expect(selectEditable(state, KEY)).toBe(true);
    });

    it("should read the legend and its visibility from the entry", () => {
      expect(selectLegend(state, KEY)).toBe(entry.legend);
      expect(selectLegendVisible(state, KEY)).toBe(false);
    });

    it("should read fit-view-on-resize from the entry", () => {
      expect(selectFitViewOnResize(state, KEY)).toBe(true);
    });

    it("should read the viewport from the entry", () => {
      expect(selectViewport(state, KEY)).toBe(entry.viewport);
    });

    it("should read selected, control status, and authority from the entry", () => {
      expect(selectSelected(state, KEY)).toBe(entry.selected);
      expect(selectControlStatus(state, KEY)).toBe("acquired");
      expect(selectAuthority(state, KEY)).toBe(5);
    });
  });

  describe("defensive selectors (absent entry)", () => {
    it("should fall back to defaults instead of throwing", () => {
      expect(selectSelected(empty, "absent")).toEqual([]);
      expect(selectControlStatus(empty, "absent")).toBe("released");
      expect(selectAuthority(empty, "absent")).toBe(1);
      expect(selectLegend(empty, "absent")).toBe(ZERO_STATE.legend);
    });
  });

  describe("hooks", () => {
    it("useSelectEditable reads through the Redux provider", () => {
      const { result } = renderHook(() => useSelectEditable(KEY), {
        wrapper: wrapperFor(state),
      });
      expect(result.current).toBe(true);
    });
  });
});
