// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { configureStore, type EnhancedStore } from "@reduxjs/toolkit";
import { act, renderHook } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { Provider } from "react-redux";
import { describe, expect, it } from "vitest";

import { Theme } from "@/session/theme";

const storeWith = (selected: string = Theme.ZERO_SLICE_STATE.selected): EnhancedStore =>
  configureStore({
    reducer: { [Theme.SLICE_NAME]: Theme.reducer },
    preloadedState: {
      [Theme.SLICE_NAME]: { ...Theme.ZERO_SLICE_STATE, selected },
    },
  });

const wrapperFor = (store: EnhancedStore) => {
  const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
    <Provider store={store}>{children}</Provider>
  );
  return Wrapper;
};

const selectedOf = (store: EnhancedStore): string =>
  store.getState()[Theme.SLICE_NAME].selected;

describe("session theme selectors", () => {
  describe("useSelectSelected", () => {
    it("returns the default selected theme", () => {
      const { result } = renderHook(() => Theme.useSelectSelected(), {
        wrapper: wrapperFor(storeWith()),
      });
      expect(result.current).toBe("synnaxLight");
    });

    it("returns the selected theme from state", () => {
      const { result } = renderHook(() => Theme.useSelectSelected(), {
        wrapper: wrapperFor(storeWith("synnaxDark")),
      });
      expect(result.current).toBe("synnaxDark");
    });
  });

  describe("useGetSelected", () => {
    it("should read the current selected theme on demand across dispatches", () => {
      const store = storeWith();
      const { result } = renderHook(() => Theme.useGetSelected(), {
        wrapper: wrapperFor(store),
      });
      const get = result.current;
      expect(get()).toBe("synnaxLight");
      act(() => {
        store.dispatch(Theme.select("synnaxDark"));
      });
      expect(get()).toBe("synnaxDark");
      act(() => {
        store.dispatch(Theme.toggle());
      });
      expect(get()).toBe("synnaxLight");
    });
  });

  describe("useProviderProps", () => {
    it("exposes the selected theme key", () => {
      const { result } = renderHook(() => Theme.useProviderProps(), {
        wrapper: wrapperFor(storeWith("synnaxDark")),
      });
      expect(result.current.theme).toEqual({ key: "synnaxDark" });
    });

    it("dispatches select when setTheme is called", () => {
      const store = storeWith();
      const { result } = renderHook(() => Theme.useProviderProps(), {
        wrapper: wrapperFor(store),
      });
      act(() => result.current.setTheme?.("synnaxDark"));
      expect(selectedOf(store)).toBe("synnaxDark");
    });

    it("dispatches toggle when toggleTheme is called", () => {
      const store = storeWith();
      const { result } = renderHook(() => Theme.useProviderProps(), {
        wrapper: wrapperFor(store),
      });
      expect(selectedOf(store)).toBe("synnaxLight");
      act(() => result.current.toggleTheme?.());
      expect(selectedOf(store)).toBe("synnaxDark");
      act(() => result.current.toggleTheme?.());
      expect(selectedOf(store)).toBe("synnaxLight");
    });
  });
});
