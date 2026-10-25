// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { configureStore, type EnhancedStore } from "@reduxjs/toolkit";
import { act, renderHook, waitFor } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { Provider } from "react-redux";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted((): { engine: "web" | "tauri" } => ({ engine: "web" }));

vi.mock("@/session/runtime/runtime", async (importOriginal) => {
  const { mockRuntimeEngine } = await import("@/testutil/runtime");
  return await mockRuntimeEngine(importOriginal, mocks);
});

const tauriWindow = { theme: vi.fn(), onThemeChanged: vi.fn(), setTheme: vi.fn() };

vi.mock("@tauri-apps/api/window", () => ({ getCurrentWindow: () => tauriWindow }));

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

interface FakeMediaQueryList {
  mql: MediaQueryList;
  fireChange: (matches: boolean) => void;
  addEventListener: ReturnType<typeof vi.fn>;
}

const stubMatchMedia = (initialMatches: boolean): FakeMediaQueryList => {
  let matches = initialMatches;
  let listener: ((e: MediaQueryListEvent) => void) | null = null;
  const addEventListener = vi.fn((_: string, cb: (e: MediaQueryListEvent) => void) => {
    listener = cb;
  });
  const mql = {
    get matches() {
      return matches;
    },
    addEventListener,
    removeEventListener: () => {
      listener = null;
    },
  } as unknown as MediaQueryList;
  vi.stubGlobal("matchMedia", vi.fn().mockReturnValue(mql));
  return {
    mql,
    addEventListener,
    fireChange: (next: boolean) => {
      matches = next;
      listener?.({ matches: next } as MediaQueryListEvent);
    },
  };
};

const themeKeyOf = (result: {
  current: { theme?: { key?: string } };
}): string | undefined => result.current.theme?.key;

describe("session theme selectors", () => {
  beforeEach(() => {
    mocks.engine = "web";
    tauriWindow.theme.mockReset();
    tauriWindow.onThemeChanged.mockReset();
    tauriWindow.setTheme.mockReset();
  });

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
  });

  describe("useProviderProps", () => {
    it("maps the light mode to the light theme key", () => {
      const { result } = renderHook(() => Theme.useProviderProps(), {
        wrapper: wrapperFor(storeWith("light")),
      });
      expect(themeKeyOf(result)).toBe("synnaxLight");
    });

    it("maps the dark mode to the dark theme key", () => {
      const { result } = renderHook(() => Theme.useProviderProps(), {
        wrapper: wrapperFor(storeWith("dark")),
      });
      expect(themeKeyOf(result)).toBe("synnaxDark");
    });

    it("follows the OS scheme in system mode on the web", async () => {
      stubMatchMedia(true);
      const { result } = renderHook(() => Theme.useProviderProps(), {
        wrapper: wrapperFor(storeWith("system")),
      });
      await waitFor(() => expect(themeKeyOf(result)).toBe("synnaxDark"));
    });

    it("updates when the OS scheme changes on the web", async () => {
      const { fireChange } = stubMatchMedia(false);
      const { result } = renderHook(() => Theme.useProviderProps(), {
        wrapper: wrapperFor(storeWith("system")),
      });
      await waitFor(() => expect(themeKeyOf(result)).toBe("synnaxLight"));

      act(() => fireChange(true));
      await waitFor(() => expect(themeKeyOf(result)).toBe("synnaxDark"));
    });

    it("follows the OS scheme in system mode in the tauri engine", async () => {
      mocks.engine = "tauri";
      tauriWindow.theme.mockResolvedValue("dark");
      tauriWindow.onThemeChanged.mockResolvedValue(() => {});
      const { result } = renderHook(() => Theme.useProviderProps(), {
        wrapper: wrapperFor(storeWith("system")),
      });
      await waitFor(() => expect(themeKeyOf(result)).toBe("synnaxDark"));
    });

    it("updates when Tauri reports an OS scheme change", async () => {
      mocks.engine = "tauri";
      tauriWindow.theme.mockResolvedValue("light");
      let onChange: ((event: { payload: "light" | "dark" }) => void) | null = null;
      tauriWindow.onThemeChanged.mockImplementation(async (cb: typeof onChange) => {
        onChange = cb;
        return () => {};
      });
      const { result } = renderHook(() => Theme.useProviderProps(), {
        wrapper: wrapperFor(storeWith("system")),
      });
      await waitFor(() => expect(themeKeyOf(result)).toBe("synnaxLight"));
      await waitFor(() => expect(onChange).not.toBeNull());

      act(() => onChange?.({ payload: "dark" }));
      await waitFor(() => expect(themeKeyOf(result)).toBe("synnaxDark"));
    });

    it("sets the window theme to a fixed mode in the tauri engine", async () => {
      mocks.engine = "tauri";
      renderHook(() => Theme.useProviderProps(), {
        wrapper: wrapperFor(storeWith("dark")),
      });
      await waitFor(() => expect(tauriWindow.setTheme).toHaveBeenCalledWith("dark"));
      expect(tauriWindow.theme).not.toHaveBeenCalled();
    });

    it("clears the window theme before reading the OS theme", async () => {
      mocks.engine = "tauri";
      tauriWindow.theme.mockResolvedValue("dark");
      tauriWindow.onThemeChanged.mockResolvedValue(() => {});
      renderHook(() => Theme.useProviderProps(), {
        wrapper: wrapperFor(storeWith("system")),
      });
      await waitFor(() => expect(tauriWindow.theme).toHaveBeenCalled());
      expect(tauriWindow.setTheme).toHaveBeenCalledWith(null);
      expect(tauriWindow.setTheme.mock.invocationCallOrder[0]).toBeLessThan(
        tauriWindow.theme.mock.invocationCallOrder[0],
      );
    });

    it("sets the window theme again when the mode changes", async () => {
      mocks.engine = "tauri";
      tauriWindow.theme.mockResolvedValue("light");
      tauriWindow.onThemeChanged.mockResolvedValue(() => {});
      const store = storeWith("system");
      renderHook(() => Theme.useProviderProps(), { wrapper: wrapperFor(store) });
      await waitFor(() => expect(tauriWindow.setTheme).toHaveBeenCalledWith(null));
      act(() => {
        store.dispatch(Theme.set("light"));
      });
      await waitFor(() =>
        expect(tauriWindow.setTheme).toHaveBeenLastCalledWith("light"),
      );
    });

    it("does not listen to the OS scheme outside system mode", async () => {
      const { addEventListener } = stubMatchMedia(true);
      const { result } = renderHook(() => Theme.useProviderProps(), {
        wrapper: wrapperFor(storeWith("light")),
      });
      await act(async () => {});
      expect(addEventListener).not.toHaveBeenCalled();
      expect(themeKeyOf(result)).toBe("synnaxLight");
    });

    it("stops following the OS scheme when the mode changes to a fixed one", async () => {
      const { fireChange } = stubMatchMedia(false);
      const store = storeWith("system");
      const { result } = renderHook(() => Theme.useProviderProps(), {
        wrapper: wrapperFor(store),
      });
      await waitFor(() => expect(themeKeyOf(result)).toBe("synnaxLight"));

      act(() => {
        store.dispatch(Theme.set("dark"));
      });
      await waitFor(() => expect(themeKeyOf(result)).toBe("synnaxDark"));

      act(() => fireChange(true));
      await act(async () => {});
      expect(themeKeyOf(result)).toBe("synnaxDark");
    });
  });
});
