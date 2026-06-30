// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { configureStore, createSlice, type PayloadAction } from "@reduxjs/toolkit";
import { act, renderHook } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { Provider } from "react-redux";
import { describe, expect, it } from "vitest";

import { Select } from "@/session/select";

interface Item {
  key: string;
  value: number;
}

interface TestState {
  tracked: { value: number };
  untracked: { value: number };
  items: Record<string, Item>;
}

const ZERO_STATE: TestState = {
  tracked: { value: 1 },
  untracked: { value: 1 },
  items: { a: { key: "a", value: 1 }, b: { key: "b", value: 2 } },
};

const testSlice = createSlice({
  name: "test",
  initialState: ZERO_STATE,
  reducers: {
    setTracked: (state, { payload }: PayloadAction<number>) => {
      state.tracked = { value: payload };
    },
    setUntracked: (state, { payload }: PayloadAction<number>) => {
      state.untracked = { value: payload };
    },
    setItem: (state, { payload }: PayloadAction<Item>) => {
      state.items[payload.key] = payload;
    },
  },
});

const { setTracked, setUntracked, setItem } = testSlice.actions;

const makeStore = () => configureStore({ reducer: testSlice.reducer });

const wrapperFor = (store: ReturnType<typeof makeStore>) => {
  const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
    <Provider store={store}>{children}</Provider>
  );
  return Wrapper;
};

describe("Select.Select.useMemo", () => {
  it("returns a stable reference across a dispatch that does not touch the accessed state", () => {
    const store = makeStore();
    const { result } = renderHook(
      () => Select.useMemo((s: TestState) => s.tracked, []),
      {
        wrapper: wrapperFor(store),
      },
    );
    const first = result.current;
    act(() => {
      store.dispatch(setUntracked(99));
    });
    expect(result.current).toBe(first);
  });

  it("returns a new value when the accessed state changes", () => {
    const store = makeStore();
    const { result } = renderHook(
      () => Select.useMemo((s: TestState) => s.tracked, []),
      {
        wrapper: wrapperFor(store),
      },
    );
    const first = result.current;
    act(() => {
      store.dispatch(setTracked(2));
    });
    expect(result.current).not.toBe(first);
    expect(result.current).toEqual({ value: 2 });
  });

  it("keeps a constructed object stable until its inputs change", () => {
    const store = makeStore();
    const { result } = renderHook(
      () => Select.useMemo((s: TestState) => ({ doubled: s.tracked.value * 2 }), []),
      { wrapper: wrapperFor(store) },
    );
    const first = result.current;
    expect(first).toEqual({ doubled: 2 });
    act(() => {
      store.dispatch(setUntracked(99));
    });
    expect(result.current).toBe(first);
    act(() => {
      store.dispatch(setTracked(5));
    });
    expect(result.current).not.toBe(first);
    expect(result.current).toEqual({ doubled: 10 });
  });

  it("tracks only the accessed entry, not its siblings", () => {
    const store = makeStore();
    const { result } = renderHook(
      () => Select.useMemo((s: TestState) => s.items.a, []),
      {
        wrapper: wrapperFor(store),
      },
    );
    const first = result.current;
    act(() => {
      store.dispatch(setItem({ key: "b", value: 99 }));
    });
    expect(result.current).toBe(first);
    act(() => {
      store.dispatch(setItem({ key: "a", value: 7 }));
    });
    expect(result.current).toEqual({ key: "a", value: 7 });
  });

  it("re-points the selector when its dependencies change", () => {
    const store = makeStore();
    const { result, rerender } = renderHook(
      ({ key }: { key: string }) =>
        Select.useMemo((s: TestState) => s.items[key], [key]),
      { wrapper: wrapperFor(store), initialProps: { key: "a" } },
    );
    expect(result.current).toEqual({ key: "a", value: 1 });
    rerender({ key: "b" });
    expect(result.current).toEqual({ key: "b", value: 2 });
  });
});
