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
import { renderHook, waitFor } from "@testing-library/react";
import { act, type PropsWithChildren, useEffect, useState } from "react";
import { Provider, useStore } from "react-redux";
import { describe, expect, it, vi } from "vitest";

import { Layout } from "@/layout";
import { select } from "@/layout/selectors";
import { reducer } from "@/layout/slice";

describe("useSyncName", () => {
  it("should sync the name of a layout", async () => {
    const store = configureStore({
      reducer: combineReducers({
        layout: reducer,
        drift: Drift.reducer,
      }),
    });
    const wrapper = ({ children }: PropsWithChildren) => (
      <Provider store={store}>{children}</Provider>
    );
    const hook1 = renderHook(
      () => ({
        placer: Layout.usePlacer(),
        store: useStore(),
      }),
      { wrapper },
    );
    act(() => {
      hook1.result.current.placer({
        key: "test",
        location: "mosaic",
        type: "cat",
        name: "test",
        window: {
          title: "test",
        },
      });
    });
    hook1.unmount();
    const handleNameChange = vi.fn();
    const hook2 = renderHook(
      () => ({
        store: useStore(),
        useSyncName: Layout.useSyncName("test", "test", handleNameChange),
      }),
      { wrapper },
    );
    act(() => {
      store.dispatch(Layout.rename({ key: "test", name: "test2" }));
    });
    expect(handleNameChange).toHaveBeenCalledWith("test2");
    hook2.unmount();

    const hook3 = renderHook(
      () => {
        const [name, setName] = useState("test");
        useEffect(() => {
          const timeout = setTimeout(() => {
            setName("test3");
          }, 5);
          return () => {
            clearTimeout(timeout);
          };
        }, []);
        Layout.useSyncName("test", name, handleNameChange);
      },
      { wrapper },
    );
    await waitFor(() => {
      const name = select(store.getState(), "test")?.name;
      expect(name).toBe("test3");
    });
    hook3.unmount();
  });
});
