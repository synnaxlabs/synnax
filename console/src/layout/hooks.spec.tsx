// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { combineReducers, configureStore } from "@reduxjs/toolkit";
import { Drift } from "@synnaxlabs/drift";
import { renderHook } from "@testing-library/react";
import { act, type PropsWithChildren } from "react";
import { Provider, useStore } from "react-redux";
import { describe, expect, it } from "vitest";

import { select } from "@/layout/selectors";
import { reducer } from "@/layout/slice";
import { usePlacer } from "@/layout/usePlacer";
import { useRemover } from "@/layout/useRemover";

describe("placing & removing", () => {
  it("should place a layout within the store", () => {
    const store = configureStore({
      reducer: combineReducers({
        layout: reducer,
        drift: Drift.reducer,
      }),
    });
    const wrapper = ({ children }: PropsWithChildren) => (
      <Provider store={store}>{children}</Provider>
    );
    const { result } = renderHook(
      () => ({
        placer: usePlacer(),
        store: useStore(),
      }),
      { wrapper },
    );
    act(() => {
      result.current.placer({
        key: "test",
        location: "mosaic",
        type: "cat",
        name: "test",
        window: {
          title: "test",
        },
      });
    });
    const state = select(store.getState(), "test");
    expect(state).toBeDefined();
    expect(state?.key).toBe("test");
    expect(state?.location).toBe("mosaic");
    expect(state?.type).toBe("cat");
    expect(state?.name).toBe("test");
  });

  it("should remove a layout from the store", () => {
    const store = configureStore({
      reducer: combineReducers({
        layout: reducer,
        drift: Drift.reducer,
      }),
    });
    const wrapper = ({ children }: PropsWithChildren) => (
      <Provider store={store}>{children}</Provider>
    );
    const { result } = renderHook(
      () => ({
        placer: usePlacer(),
        store: useStore(),
        remover: useRemover(),
      }),
      { wrapper },
    );
    act(() => {
      result.current.placer({
        key: "test",
        location: "mosaic",
        type: "cat",
        name: "test",
        window: {
          title: "test",
        },
      });
    });
    act(() => {
      result.current.remover("test");
    });
    const state = select(store.getState(), "test");
    expect(state).toBeUndefined();
  });
});
