// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { configureStore } from "@reduxjs/toolkit";
import { act, renderHook } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { Provider } from "react-redux";
import { describe, expect, it } from "vitest";

import { Range } from "@/session/range";

const STATIC: Range.StaticState = {
  key: "static-1",
  name: "Static 1",
  persisted: true,
  variant: "static",
  timeRange: { start: 0, end: 1000 },
};

const DYNAMIC: Range.DynamicState = {
  key: "dynamic-1",
  name: "Dynamic 1",
  persisted: false,
  variant: "dynamic",
  span: 1000,
};

const EMPTY_STATE: Range.SliceState = { version: 0, selected: undefined, ranges: [] };

const stateWith = (ranges: Range.State[], selected?: string): Range.SliceState => ({
  version: 0,
  selected,
  ranges,
});

const storeWith = (ranges: Range.State[], selected?: string): Range.StoreState => ({
  [Range.SLICE_NAME]: stateWith(ranges, selected),
});

const createStore = (initial: Range.SliceState = EMPTY_STATE) =>
  configureStore({
    reducer: { [Range.SLICE_NAME]: Range.reducer },
    preloadedState: { [Range.SLICE_NAME]: initial },
  });

const createWrapper =
  (store: ReturnType<typeof createStore>) =>
  ({ children }: PropsWithChildren): ReactElement => (
    <Provider store={store}>{children}</Provider>
  );

describe("range selectors", () => {
  describe("raw selectors", () => {
    describe("selectSelectedKey", () => {
      it("should return the selected key", () => {
        expect(Range.selectSelectedKey(storeWith([STATIC], STATIC.key))).toEqual(
          STATIC.key,
        );
      });

      it("should return undefined when no range is selected", () => {
        expect(Range.selectSelectedKey(storeWith([STATIC]))).toBeUndefined();
      });
    });

    describe("selectState", () => {
      it("should resolve a range by explicit key", () => {
        expect(Range.selectState(storeWith([STATIC, DYNAMIC]), STATIC.key)).toEqual(
          STATIC,
        );
      });

      it("should fall back to the selected range when no key is given", () => {
        expect(Range.selectState(storeWith([STATIC, DYNAMIC], STATIC.key))).toEqual(
          STATIC,
        );
      });

      it("should return undefined when no range matches", () => {
        expect(Range.selectState(storeWith([STATIC]), "missing")).toBeUndefined();
      });
    });
  });

  describe("getters", () => {
    it("should read the whole slice state on demand across dispatches", () => {
      const store = createStore();
      const { result } = renderHook(() => Range.useGetSliceState(), {
        wrapper: createWrapper(store),
      });
      const get = result.current;
      expect(get().ranges).toEqual([]);
      act(() => {
        store.dispatch(Range.add(STATIC));
      });
      expect(get().ranges).toEqual([STATIC]);
    });

    it("should read the current selected key on demand across dispatches", () => {
      const store = createStore(stateWith([STATIC, DYNAMIC]));
      const { result } = renderHook(() => Range.useGetSelectedKey(), {
        wrapper: createWrapper(store),
      });
      const get = result.current;
      expect(get()).toBeUndefined();
      act(() => {
        store.dispatch(Range.select(STATIC.key));
      });
      expect(get()).toEqual(STATIC.key);
      act(() => {
        store.dispatch(Range.select(DYNAMIC.key));
      });
      expect(get()).toEqual(DYNAMIC.key);
    });

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
    });

    it("should resolve only static ranges on demand", () => {
      const store = createStore();
      const { result } = renderHook(() => Range.useGetStatic(), {
        wrapper: createWrapper(store),
      });
      const get = result.current;
      expect(get(STATIC.key)).toBeUndefined();
      act(() => {
        store.dispatch(Range.add([STATIC, DYNAMIC]));
      });
      expect(get(STATIC.key)).toEqual(STATIC);
      expect(get(DYNAMIC.key)).toBeUndefined();
    });

    it("should return all ranges and filter to requested keys on demand", () => {
      const store = createStore();
      const { result } = renderHook(() => Range.useGetMultiple(), {
        wrapper: createWrapper(store),
      });
      const get = result.current;
      expect(get()).toEqual([]);
      act(() => {
        store.dispatch(Range.add([STATIC, DYNAMIC]));
      });
      expect(get()).toEqual([STATIC, DYNAMIC]);
      expect(get([DYNAMIC.key])).toEqual([DYNAMIC]);
    });

    it("should return the keys of every range on demand", () => {
      const store = createStore();
      const { result } = renderHook(() => Range.useGetKeys(), {
        wrapper: createWrapper(store),
      });
      const get = result.current;
      expect(get()).toEqual([]);
      act(() => {
        store.dispatch(Range.add([STATIC, DYNAMIC]));
      });
      expect(get()).toEqual([STATIC.key, DYNAMIC.key]);
    });

    it("should return only the keys of static ranges on demand", () => {
      const store = createStore();
      const { result } = renderHook(() => Range.useGetStaticKeys(), {
        wrapper: createWrapper(store),
      });
      const get = result.current;
      expect(get()).toEqual([]);
      act(() => {
        store.dispatch(Range.add([STATIC, DYNAMIC]));
      });
      expect(get()).toEqual([STATIC.key]);
    });
  });
});
