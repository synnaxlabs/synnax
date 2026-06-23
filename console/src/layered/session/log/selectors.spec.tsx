// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { configureStore } from "@reduxjs/toolkit";
import { Log as PlutoLog } from "@synnaxlabs/pluto";
import { renderHook } from "@testing-library/react";
import { type FC, type PropsWithChildren, type ReactElement } from "react";
import { Provider } from "react-redux";
import { describe, expect, it } from "vitest";

import { Log } from "@/layered/session/log";

const KEY = "log-1";

const customState = Log.stateZ.parse({ toolbar: { activeTab: "properties" } });

const storeState: Log.StoreState = {
  [Log.SLICE_NAME]: { logs: { [KEY]: customState } },
};

const params = { state: storeState, key: KEY };

describe("log selectors", () => {
  describe("selectSliceState", () => {
    it("should return the slice state", () => {
      expect(Log.selectSliceState(storeState)).toBe(storeState[Log.SLICE_NAME]);
    });
  });

  describe("selectOptional", () => {
    it("should return the entry when present", () => {
      expect(Log.selectOptional(params)).toBe(customState);
    });

    it("should return undefined for an unknown key", () => {
      expect(Log.selectOptional({ state: storeState, key: "absent" })).toBeUndefined();
    });
  });

  describe("selectState", () => {
    it("should return the state for the given key", () => {
      expect(Log.selectState(params)).toEqual(customState);
    });

    it("should fall back to ZERO_STATE for an unknown key", () => {
      expect(Log.selectState({ state: storeState, key: "absent" })).toEqual(
        Log.ZERO_STATE,
      );
    });
  });

  describe("selectExists", () => {
    it("should report whether the entry is present", () => {
      expect(Log.selectExists(params)).toBe(true);
      expect(Log.selectExists({ state: storeState, key: "absent" })).toBe(false);
    });
  });

  describe("selectActiveToolbarTab", () => {
    it("should read the active toolbar tab", () => {
      expect(Log.selectActiveToolbarTab(params)).toBe("properties");
    });
  });

  describe("selectVersion", () => {
    it("should read the version, undefined when absent", () => {
      expect(Log.selectVersion(params)).toBe(Log.ZERO_STATE.version);
      expect(Log.selectVersion({ state: storeState, key: "absent" })).toBeUndefined();
    });
  });
});

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
      <PlutoLog.Scope.Provider value={key}>{children}</PlutoLog.Scope.Provider>
    </Provider>
  );
  Wrapper.displayName = "Wrapper";
  return Wrapper;
};

describe("log selector hooks", () => {
  const store = (): ReturnType<typeof storeWith> =>
    storeWith({ logs: { [KEY]: customState } });

  it("should resolve the key from the surrounding scope", () => {
    const { result } = renderHook(() => Log.useSelect(), {
      wrapper: wrapperFor(store(), KEY),
    });
    expect(result.current).toEqual(customState);
  });

  it("should let an explicit key override the scope", () => {
    const { result } = renderHook(() => Log.useSelect({ key: "absent" }), {
      wrapper: wrapperFor(store(), KEY),
    });
    expect(result.current).toEqual(Log.ZERO_STATE);
  });

  it("should report whether the entry exists", () => {
    const { result } = renderHook(() => Log.useSelectExists(), {
      wrapper: wrapperFor(store(), KEY),
    });
    expect(result.current).toBe(true);
  });

  it("should read the active toolbar tab", () => {
    const { result } = renderHook(() => Log.useSelectActiveToolbarTab(), {
      wrapper: wrapperFor(store(), KEY),
    });
    expect(result.current).toBe("properties");
  });

  it("should read the version", () => {
    const { result } = renderHook(() => Log.useSelectVersion(), {
      wrapper: wrapperFor(store(), KEY),
    });
    expect(result.current).toBe(Log.ZERO_STATE.version);
  });
});
