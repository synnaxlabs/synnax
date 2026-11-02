// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Log as PLog } from "@synnaxlabs/pluto";
import { act, renderHook } from "@testing-library/react";
import { type FC, type PropsWithChildren, type ReactElement } from "react";
import { Provider } from "react-redux";
import { describe, expect, it } from "vitest";

import { Log } from "@/session/log";
import { createSliceStore, inWindow } from "@/session/window/testutil";

const KEY = "log-1";

const customState: Log.State = {
  hold: true,
  toolbar: { selectedTab: "properties" },
};

const storeWith = (slice: Log.SliceState) =>
  createSliceStore({
    name: Log.SLICE_NAME,
    reducer: Log.reducer,
    preloadedState: slice,
    middleware: Log.MIDDLEWARE,
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

const createCustomStore = () =>
  storeWith({ version: 0, windows: inWindow({ [KEY]: customState }) });

describe("log selector hooks", () => {
  it("should read the active toolbar tab", () => {
    const { result } = renderHook(() => Log.useSelectSelectedToolbarTab(), {
      wrapper: wrapperFor(createCustomStore(), KEY),
    });
    expect(result.current).toBe("properties");
  });

  it("should read the hold state", () => {
    const { result } = renderHook(() => Log.useSelectHold(), {
      wrapper: wrapperFor(createCustomStore(), KEY),
    });
    expect(result.current).toBe(true);
  });

  it("should default hold to false for a log with no stored state", () => {
    const { result } = renderHook(() => Log.useSelectHold({ key: "absent" }), {
      wrapper: wrapperFor(createCustomStore(), KEY),
    });
    expect(result.current).toBe(false);
  });

  it("should track hold across dispatches", () => {
    const store = storeWith(Log.ZERO_SLICE_STATE);
    const { result } = renderHook(() => Log.useSelectHold(), {
      wrapper: wrapperFor(store, KEY),
    });
    expect(result.current).toBe(false);
    act(() => {
      store.dispatch(Log.create({ key: KEY }));
      store.dispatch(Log.setHold({ key: KEY, hold: true }));
    });
    expect(result.current).toBe(true);
    act(() => {
      store.dispatch(Log.setHold({ key: KEY, hold: false }));
    });
    expect(result.current).toBe(false);
  });
});
