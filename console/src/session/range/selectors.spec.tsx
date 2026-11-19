// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { act, renderHook } from "@testing-library/react";
import { type FC, type PropsWithChildren, type ReactElement } from "react";
import { Provider } from "react-redux";
import { describe, expect, it } from "vitest";

import { Range } from "@/session/range";
import { createMinimalStore } from "@/testutil";

const PERSISTED: Range.PersistedState = { variant: "persisted", key: "persisted-1" };

const createStore = (initial: Range.SliceState = Range.ZERO_SLICE_STATE) =>
  createMinimalStore({
    reducer: { [Range.SLICE_NAME]: Range.reducer },
    preloadedState: { [Range.SLICE_NAME]: initial },
  });

const createWrapper = (
  store: ReturnType<typeof createStore>,
): FC<PropsWithChildren> => {
  const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
    <Provider store={store}>{children}</Provider>
  );
  Wrapper.displayName = "Wrapper";
  return Wrapper;
};

describe("range selectors", () => {
  describe("useSelectState", () => {
    it("should resolve a range by key", () => {
      const store = createStore();
      const { result } = renderHook(() => Range.useSelectState(PERSISTED.key), {
        wrapper: createWrapper(store),
      });
      expect(result.current).toBeUndefined();
      act(() => {
        store.dispatch(Range.add(PERSISTED));
      });
      expect(result.current).toEqual(PERSISTED);
    });

    it("should offer the rolling ranges whatever the session holds", () => {
      const store = createStore();
      const { result } = renderHook(() => Range.useSelectState(Range.BUILT_IN[0].key), {
        wrapper: createWrapper(store),
      });
      expect(result.current).toEqual(Range.BUILT_IN[0]);
      act(() => {
        store.dispatch(Range.remove({ keys: [Range.BUILT_IN[0].key] }));
      });
      expect(result.current).toEqual(Range.BUILT_IN[0]);
    });
  });

  describe("useSelectFavoriteKeys", () => {
    it("should list the added Core ranges and leave out the rolling ones", () => {
      const store = createStore();
      const { result } = renderHook(() => Range.useSelectFavoriteKeys(), {
        wrapper: createWrapper(store),
      });
      expect(result.current).toEqual([]);
      act(() => {
        store.dispatch(Range.add(PERSISTED));
      });
      expect(result.current).toEqual([PERSISTED.key]);
    });
  });

  describe("useSelectGroupCollapsed", () => {
    it("should follow the toolbar group it names", () => {
      const store = createStore();
      const { result } = renderHook(() => Range.useSelectGroupCollapsed("recent"), {
        wrapper: createWrapper(store),
      });
      expect(result.current).toBe(false);
      act(() => {
        store.dispatch(Range.setGroupCollapsed({ group: "recent", collapsed: true }));
      });
      expect(result.current).toBe(true);
      act(() => {
        store.dispatch(Range.setGroupCollapsed({ group: "running", collapsed: true }));
      });
      expect(result.current).toBe(true);
    });
  });
});
