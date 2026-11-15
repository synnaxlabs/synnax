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

const STATIC: Range.StaticState = {
  variant: "static",
  key: "static-1",
  name: "Static 1",
  timeRange: { start: 0, end: 1000 },
};

const DYNAMIC: Range.DynamicState = {
  variant: "dynamic",
  key: "dynamic-1",
  name: "Dynamic 1",
  span: 1000,
};

const EMPTY_STATE: Range.SliceState = {
  version: 0,
  selected: undefined,
  ranges: [],
  staticKeysRewritten: false,
};

const createStore = (initial: Range.SliceState = EMPTY_STATE) =>
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
  describe("getters", () => {
    it("should resolve a range by key and fall back to the selection on demand", () => {
      const store = createStore();
      const { result } = renderHook(() => Range.useGetState(), {
        wrapper: createWrapper(store),
      });
      const get = result.current;
      expect(get(STATIC.key)).toBeUndefined();
      act(() => {
        store.dispatch(Range.add([STATIC, DYNAMIC]));
      });
      expect(get(STATIC.key)).toEqual(STATIC);
      act(() => {
        store.dispatch(Range.select(DYNAMIC.key));
      });
      expect(get()).toEqual(DYNAMIC);
      expect(get("missing")).toBeUndefined();
    });

    // Callers fall back to the recent range when nothing is selected, so it has to be
    // there whatever the session has stored.
    it("should offer the rolling ranges whatever the session holds", () => {
      const store = createStore();
      const { result } = renderHook(() => Range.useGetState(), {
        wrapper: createWrapper(store),
      });
      const get = result.current;
      expect(Range.BUILT_IN.every(({ key }) => get(key) != null)).toBe(true);
      act(() => {
        store.dispatch(Range.remove({ keys: [Range.RECENT_KEY] }));
      });
      expect(get(Range.RECENT_KEY)).toBeDefined();
    });
  });
});
