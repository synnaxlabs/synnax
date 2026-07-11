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
import { beforeEach, describe, expect, it } from "vitest";

import { Log } from "@/session/log";

const storeWith = (slice: Log.SliceState) =>
  configureStore({
    reducer: { [Log.SLICE_NAME]: Log.reducer },
    preloadedState: { [Log.SLICE_NAME]: slice },
  });

const KEY = "log-1";

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

describe("Log Slice", () => {
  let store: ReturnType<typeof storeWith>;

  beforeEach(() => {
    store = storeWith(Log.ZERO_SLICE_STATE);
  });

  const renderGetter = <G,>(useGetter: () => G, key: string = KEY): G =>
    renderHook(() => useGetter(), { wrapper: wrapperFor(store, key) }).result.current;

  describe("create", () => {
    it("should bootstrap session state from ZERO_STATE for the key", () => {
      const getState = renderGetter(Log.useGetState);
      act(() => void store.dispatch(Log.create({ key: KEY })));
      expect(getState()).toEqual(Log.ZERO_STATE);
    });

    it("should create multiple logs independently", () => {
      const getTab = renderGetter(Log.useGetSelectedToolbarTab);
      act(() => {
        store.dispatch(
          Log.create({ key: "log-1", toolbar: { selectedTab: "properties" } }),
        );
        store.dispatch(Log.create({ key: "log-2" }));
      });
      expect(getTab({ key: "log-1" })).toBe("properties");
      expect(getTab({ key: "log-2" })).toBe("channels");
    });

    it("should not overwrite an existing entry", () => {
      const getTab = renderGetter(Log.useGetSelectedToolbarTab);
      act(() => {
        store.dispatch(Log.create({ key: KEY }));
        store.dispatch(Log.setSelectedToolbarTab({ key: KEY, tab: "properties" }));
        store.dispatch(Log.create({ key: KEY }));
      });
      expect(getTab()).toBe("properties");
    });
  });

  describe("setActiveToolbarTab", () => {
    it("should set the active toolbar tab", () => {
      const getTab = renderGetter(Log.useGetSelectedToolbarTab);
      act(() => {
        store.dispatch(Log.create({ key: KEY }));
        store.dispatch(Log.setSelectedToolbarTab({ key: KEY, tab: "properties" }));
      });
      expect(getTab()).toBe("properties");
    });

    it("should provision state on first action for an unknown key", () => {
      const getTab = renderGetter(Log.useGetSelectedToolbarTab, "absent");
      act(
        () =>
          void store.dispatch(
            Log.setSelectedToolbarTab({ key: "absent", tab: "properties" }),
          ),
      );
      expect(getTab()).toBe("properties");
    });
  });

  describe("remove", () => {
    it("should remove a log by key", () => {
      const getState = renderGetter(Log.useGetState);
      act(() => {
        store.dispatch(
          Log.create({ key: KEY, toolbar: { selectedTab: "properties" } }),
        );
        store.dispatch(Log.remove({ keys: [KEY] }));
      });
      expect(getState()).toEqual(Log.ZERO_STATE);
    });

    it("should remove multiple logs at once", () => {
      const getState = renderGetter(Log.useGetState);
      act(() => {
        store.dispatch(
          Log.create({ key: "log-1", toolbar: { selectedTab: "properties" } }),
        );
        store.dispatch(
          Log.create({ key: "log-2", toolbar: { selectedTab: "properties" } }),
        );
        store.dispatch(Log.remove({ keys: ["log-1", "log-2"] }));
      });
      expect(getState({ key: "log-1" })).toEqual(Log.ZERO_STATE);
      expect(getState({ key: "log-2" })).toEqual(Log.ZERO_STATE);
    });
  });

  describe("stateZ schema", () => {
    it("should accept the zero state", () => {
      expect(() => Log.stateZ.parse(Log.ZERO_STATE)).not.toThrow();
    });

    it("should default the toolbar when missing", () => {
      const { toolbar: _toolbar, ...withoutToolbar } = Log.ZERO_STATE;
      expect(Log.stateZ.parse(withoutToolbar).toolbar).toEqual(Log.ZERO_STATE.toolbar);
    });

    it("should reject an incorrect slice version", () => {
      expect(() =>
        Log.sliceStateZ.parse({ ...Log.ZERO_SLICE_STATE, version: 1 }),
      ).toThrow();
    });
  });
});
