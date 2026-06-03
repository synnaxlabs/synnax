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
  selectExists,
  selectIsRemoteCreated,
  selectOptional,
  selectSliceState,
  selectVersion,
  useSelect,
  useSelectExists,
} from "@/log/selectors";
import {
  reducer,
  SLICE_NAME,
  type State,
  type StoreState,
  ZERO_SLICE_STATE,
  ZERO_STATE,
} from "@/log/slice";

const KEY = "log-1";

const entry: State = {
  ...ZERO_STATE,
  key: KEY,
  remoteCreated: true,
  toolbar: { ...ZERO_STATE.toolbar, activeTab: "properties" },
};

const state: StoreState = {
  [SLICE_NAME]: { ...ZERO_SLICE_STATE, logs: { [KEY]: entry } },
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

describe("log selectors", () => {
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

  describe("selectActiveToolbarTab", () => {
    it("reads the active toolbar tab", () => {
      expect(selectActiveToolbarTab(state, KEY)).toBe("properties");
    });
  });

  describe("selectVersion", () => {
    it("reads the version, undefined when absent", () => {
      expect(selectVersion(state, KEY)).toBe(ZERO_STATE.version);
      expect(selectVersion(empty, "absent")).toBeUndefined();
    });
  });

  describe("selectIsRemoteCreated", () => {
    it("reads remoteCreated, undefined when absent", () => {
      expect(selectIsRemoteCreated(state, KEY)).toBe(true);
      expect(selectIsRemoteCreated(empty, "absent")).toBeUndefined();
    });
  });

  describe("hooks", () => {
    it("useSelect reads through the Redux provider", () => {
      const { result } = renderHook(() => useSelect(KEY), {
        wrapper: wrapperFor(state),
      });
      expect(result.current).toBe(entry);
    });

    it("useSelectExists reads through the Redux provider", () => {
      const { result } = renderHook(() => useSelectExists(KEY), {
        wrapper: wrapperFor(state),
      });
      expect(result.current).toBe(true);
    });
  });
});
