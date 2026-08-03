// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { configureStore } from "@reduxjs/toolkit";
import { Log as PLog } from "@synnaxlabs/pluto";
import { act, renderHook } from "@testing-library/react";
import { type FC, type PropsWithChildren, type ReactElement } from "react";
import { Provider } from "react-redux";
import { describe, expect, it } from "vitest";

import { Log } from "@/session/log";

const KEY = "log-1";

const customState = Log.stateZ.parse({ toolbar: { selectedTab: "properties" } });

const storeWith = (slice: Log.SliceState) =>
  configureStore({
    reducer: { [Log.SLICE_NAME]: Log.reducer },
    preloadedState: { [Log.SLICE_NAME]: slice },
  });

const wrapperFor = (
  store: ReturnType<typeof storeWith>,
  key: string,
): FC<PropsWithChildren> => {
  const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
    <Provider store={store}>
      <PLog.Scope.Provider value={key}>{children}</PLog.Scope.Provider>
    </Provider>
  );
  Wrapper.displayName = "Wrapper";
  return Wrapper;
};

const createCustomStore = () => storeWith({ version: 0, logs: { [KEY]: customState } });

describe("log selector hooks", () => {
  it("should resolve the key from the surrounding scope", () => {
    const { result } = renderHook(() => Log.useSelectState(), {
      wrapper: wrapperFor(createCustomStore(), KEY),
    });
    expect(result.current).toEqual(customState);
  });

  it("should let an explicit key override the scope", () => {
    const { result } = renderHook(() => Log.useSelectState({ key: "absent" }), {
      wrapper: wrapperFor(createCustomStore(), KEY),
    });
    expect(result.current).toEqual(Log.ZERO_STATE);
  });

  it("should read the active toolbar tab", () => {
    const { result } = renderHook(() => Log.useSelectSelectedToolbarTab(), {
      wrapper: wrapperFor(createCustomStore(), KEY),
    });
    expect(result.current).toBe("properties");
  });
});

describe("log getters", () => {
  it("should read a log's state on demand across dispatches", () => {
    const store = storeWith(Log.ZERO_SLICE_STATE);
    const { result } = renderHook(() => Log.useGetState(), {
      wrapper: wrapperFor(store, KEY),
    });
    const get = result.current;
    expect(get()).toEqual(Log.ZERO_STATE);
    act(() => {
      store.dispatch(Log.create({ key: KEY }));
      store.dispatch(Log.setSelectedToolbarTab({ key: KEY, tab: "properties" }));
    });
    expect(get().toolbar.selectedTab).toBe("properties");
  });
});
