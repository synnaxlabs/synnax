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
  selectControlState,
  selectControlStateOptional,
  selectExists,
  selectIsRemoteCreated,
  selectMeasureMode,
  selectOptional,
  selectSelection,
  selectSliceState,
  selectToolbar,
  selectVersion,
  selectViewportMode,
  useSelectActiveToolbarTab,
} from "@/lineplot/selectors";
import {
  reducer,
  SLICE_NAME,
  type State,
  type StoreState,
  ZERO_SLICE_STATE,
  ZERO_STATE,
} from "@/lineplot/slice";

const KEY = "plot-1";

const entry: State = {
  ...ZERO_STATE,
  key: KEY,
  remoteCreated: true,
  toolbar: { activeTab: "annotations" },
};

const state: StoreState = {
  [SLICE_NAME]: { ...ZERO_SLICE_STATE, plots: { [KEY]: entry } },
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

describe("lineplot selectors", () => {
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
    it("reports whether the entry is present", () => {
      expect(selectExists(state, KEY)).toBe(true);
      expect(selectExists(empty, "absent")).toBe(false);
    });
  });

  describe("toolbar selectors", () => {
    it("selectActiveToolbarTab reads the active tab", () => {
      expect(selectActiveToolbarTab(state, KEY)).toBe("annotations");
    });

    it("selectToolbar returns the toolbar, undefined when absent", () => {
      expect(selectToolbar(state, KEY)).toBe(entry.toolbar);
      expect(selectToolbar(empty, "absent")).toBeUndefined();
    });
  });

  describe("control / viewport / measure / selection", () => {
    it("selectControlState and its optional variant", () => {
      expect(selectControlState(state, KEY)).toBe(entry.control);
      expect(selectControlStateOptional(state, KEY)).toBe(entry.control);
      expect(selectControlStateOptional(empty, "absent")).toBeUndefined();
    });

    it("selectViewportMode reads the mode", () => {
      expect(selectViewportMode(state, KEY)).toBe(entry.mode);
    });

    it("selectMeasureMode reads the measure mode", () => {
      expect(selectMeasureMode(state, KEY)).toBe(entry.measure.mode);
    });

    it("selectSelection reads the selection", () => {
      expect(selectSelection(state, KEY)).toBe(entry.selection);
    });
  });

  describe("metadata selectors", () => {
    it("selectVersion reads the version, undefined when absent", () => {
      expect(selectVersion(state, KEY)).toBe(ZERO_STATE.version);
      expect(selectVersion(empty, "absent")).toBeUndefined();
    });

    it("selectIsRemoteCreated reads remoteCreated, undefined when absent", () => {
      expect(selectIsRemoteCreated(state, KEY)).toBe(true);
      expect(selectIsRemoteCreated(empty, "absent")).toBeUndefined();
    });
  });

  describe("hooks", () => {
    it("useSelectActiveToolbarTab reads through the Redux provider", () => {
      const { result } = renderHook(() => useSelectActiveToolbarTab(KEY), {
        wrapper: wrapperFor(state),
      });
      expect(result.current).toBe("annotations");
    });
  });
});
