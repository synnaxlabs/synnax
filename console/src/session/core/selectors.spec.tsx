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
import { type FC, type PropsWithChildren, type ReactElement } from "react";
import { Provider } from "react-redux";
import { describe, expect, it } from "vitest";

import { Core } from "@/session/core";
import { createCore } from "@/session/core/testutil";

const CORE_A = createCore("Alpha", { host: "a.example.com" });
const CORE_B = createCore("Beta", { host: "b.example.com", secure: true });

const KEY_A = CORE_A.key;
const KEY_B = CORE_B.key;

const EMPTY_STATE: Core.SliceState = {
  version: 0,
  selected: undefined,
  cores: {},
};

const stateWith = (cores: Core.Core[], selected?: string): Core.SliceState => ({
  version: 0,
  selected,
  cores: Object.fromEntries(cores.map((c) => [c.key, c])),
});

const storeWith = (cores: Core.Core[], selected?: string): Core.StoreState => ({
  [Core.SLICE_NAME]: stateWith(cores, selected),
});

const createStore = (initial: Core.SliceState = EMPTY_STATE) =>
  configureStore({
    reducer: { [Core.SLICE_NAME]: Core.reducer },
    preloadedState: { [Core.SLICE_NAME]: initial },
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

describe("Core selectors", () => {
  describe("raw selectors", () => {
    describe("selectSelectedKey", () => {
      it("should return the selected key", () => {
        expect(Core.selectSelectedKey(storeWith([CORE_A], KEY_A))).toBe(KEY_A);
      });

      it("should return undefined when nothing is selected", () => {
        expect(Core.selectSelectedKey(storeWith([CORE_A]))).toBeUndefined();
      });
    });

    describe("selectState", () => {
      it("should resolve a Core by explicit key", () => {
        expect(Core.selectState(storeWith([CORE_A, CORE_B]), KEY_B)).toEqual(CORE_B);
      });

      it("should fall back to the selected Core when no key is given", () => {
        expect(Core.selectState(storeWith([CORE_A, CORE_B], KEY_A))).toEqual(CORE_A);
      });

      it("should return undefined when neither a key nor a selection resolves", () => {
        expect(Core.selectState(storeWith([CORE_A]))).toBeUndefined();
      });
    });
  });

  describe("getters", () => {
    it("should read the whole slice state on demand across dispatches", () => {
      const store = createStore();
      const { result } = renderHook(() => Core.useGetSliceState(), {
        wrapper: createWrapper(store),
      });
      const get = result.current;
      expect(get().cores).toEqual({});
      act(() => {
        store.dispatch(Core.set(CORE_A));
      });
      expect(get().cores).toEqual({ [KEY_A]: CORE_A });
    });

    it("should read the current selected key on demand across dispatches", () => {
      const store = createStore(stateWith([CORE_A, CORE_B]));
      const { result } = renderHook(() => Core.useGetSelectedKey(), {
        wrapper: createWrapper(store),
      });
      const get = result.current;
      expect(get()).toBeUndefined();
      act(() => {
        store.dispatch(Core.select(KEY_A));
      });
      expect(get()).toBe(KEY_A);
      act(() => {
        store.dispatch(Core.select(KEY_B));
      });
      expect(get()).toBe(KEY_B);
    });

    it("should resolve a Core by key and fall back to the selection on demand", () => {
      const store = createStore();
      const { result } = renderHook(() => Core.useGetState(), {
        wrapper: createWrapper(store),
      });
      const get = result.current;
      expect(get(KEY_B)).toBeUndefined();
      act(() => {
        store.dispatch(Core.set(CORE_A));
        store.dispatch(Core.set(CORE_B));
      });
      expect(get(KEY_B)).toEqual(CORE_B);
      act(() => {
        store.dispatch(Core.select(KEY_A));
      });
      expect(get()).toEqual(CORE_A);
    });

    it("should report whether any Core is selected on demand", () => {
      const store = createStore(stateWith([CORE_A]));
      const { result } = renderHook(() => Core.useGetIsAnySelected(), {
        wrapper: createWrapper(store),
      });
      const get = result.current;
      expect(get()).toBe(false);
      act(() => {
        store.dispatch(Core.select(KEY_A));
      });
      expect(get()).toBe(true);
    });
  });
});
