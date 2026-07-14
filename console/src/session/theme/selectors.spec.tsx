// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { configureStore, type EnhancedStore } from "@reduxjs/toolkit";
import { renderHook, waitFor } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { Provider } from "react-redux";
import { afterEach, describe, expect, it, vi } from "vitest";

import { Theme } from "@/session/theme";

const storeWith = (mode: Theme.Mode = Theme.ZERO_SLICE_STATE.mode): EnhancedStore =>
  configureStore({
    reducer: { [Theme.SLICE_NAME]: Theme.reducer },
    preloadedState: { [Theme.SLICE_NAME]: { ...Theme.ZERO_SLICE_STATE, mode } },
  });

const wrapperFor = (store: EnhancedStore) => {
  const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
    <Provider store={store}>{children}</Provider>
  );
  return Wrapper;
};

const stubMatchMedia = (matches: boolean): void => {
  vi.stubGlobal(
    "matchMedia",
    vi.fn().mockReturnValue({
      matches,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
    }),
  );
};

describe("session theme selectors", () => {
  afterEach(() => vi.unstubAllGlobals());

  describe("useSelectMode", () => {
    it("returns the default system mode", () => {
      const { result } = renderHook(() => Theme.useSelectMode(), {
        wrapper: wrapperFor(storeWith()),
      });
      expect(result.current).toBe("system");
    });

    it("returns the mode from state", () => {
      const { result } = renderHook(() => Theme.useSelectMode(), {
        wrapper: wrapperFor(storeWith("dark")),
      });
      expect(result.current).toBe("dark");
    });

    it("falls back to system for legacy state without a mode", () => {
      const store = configureStore({
        reducer: { [Theme.SLICE_NAME]: Theme.reducer },
        preloadedState: {
          [Theme.SLICE_NAME]: { version: 0 } as Theme.SliceState,
        },
      });
      const { result } = renderHook(() => Theme.useSelectMode(), {
        wrapper: wrapperFor(store),
      });
      expect(result.current).toBe("system");
    });
  });

  describe("useProviderProps", () => {
    it("maps the light mode to the light theme key", () => {
      const { result } = renderHook(() => Theme.useProviderProps(), {
        wrapper: wrapperFor(storeWith("light")),
      });
      expect(result.current.theme).toEqual({ key: "synnaxLight" });
    });

    it("maps the dark mode to the dark theme key", () => {
      const { result } = renderHook(() => Theme.useProviderProps(), {
        wrapper: wrapperFor(storeWith("dark")),
      });
      expect(result.current.theme).toEqual({ key: "synnaxDark" });
    });

    it("follows the OS scheme in system mode", async () => {
      stubMatchMedia(true);
      const { result } = renderHook(() => Theme.useProviderProps(), {
        wrapper: wrapperFor(storeWith("system")),
      });
      await waitFor(() => expect(result.current.theme).toEqual({ key: "synnaxDark" }));
    });
  });
});
