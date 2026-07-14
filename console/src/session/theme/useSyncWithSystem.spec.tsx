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

const mocks = vi.hoisted((): { engine: "web" | "tauri"; label: string } => ({
  engine: "web",
  label: "main",
}));

vi.mock("@/session/runtime/runtime", async (importOriginal) => {
  const { mockRuntimeEngine } = await import("@/testutil/runtime");
  return await mockRuntimeEngine(importOriginal, mocks);
});

const tauriWindow = {
  get label() {
    return mocks.label;
  },
  theme: vi.fn(),
  onThemeChanged: vi.fn(),
};

vi.mock("@tauri-apps/api/window", () => ({
  getCurrentWindow: () => tauriWindow,
}));

import { Theme } from "@/session/theme";

interface FakeMediaQueryList {
  mql: MediaQueryList;
  fireChange: (matches: boolean) => void;
}

const createMatchMedia = (initialMatches: boolean): FakeMediaQueryList => {
  let matches = initialMatches;
  let listener: ((e: MediaQueryListEvent) => void) | null = null;
  const mql = {
    get matches() {
      return matches;
    },
    addEventListener: (_: string, cb: (e: MediaQueryListEvent) => void) => {
      listener = cb;
    },
    removeEventListener: () => {
      listener = null;
    },
  } as unknown as MediaQueryList;
  return {
    mql,
    fireChange: (next: boolean) => {
      matches = next;
      listener?.({ matches: next } as MediaQueryListEvent);
    },
  };
};

const createStore = (
  syncWithSystem: boolean = Theme.ZERO_SLICE_STATE.syncWithSystem,
): EnhancedStore =>
  configureStore({
    reducer: { [Theme.SLICE_NAME]: Theme.reducer },
    preloadedState: {
      [Theme.SLICE_NAME]: { ...Theme.ZERO_SLICE_STATE, syncWithSystem },
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

describe("useSyncWithSystem", () => {
  beforeEach(() => {
    mocks.engine = "web";
    mocks.label = "main";
    tauriWindow.theme.mockReset();
    tauriWindow.onThemeChanged.mockReset();
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("sets the initial theme from the OS preference in the web engine", async () => {
    const { mql } = createMatchMedia(true);
    vi.stubGlobal("matchMedia", vi.fn().mockReturnValue(mql));
    const store = createStore();
    renderHook(() => Theme.useSyncWithSystem(), { wrapper: wrapperFor(store) });
    await waitFor(() => expect(selectedOf(store)).toBe("synnaxDark"));
  });

  it("updates the theme when the OS preference changes in the web engine", async () => {
    const { mql, fireChange } = createMatchMedia(false);
    vi.stubGlobal("matchMedia", vi.fn().mockReturnValue(mql));
    const store = createStore();
    renderHook(() => Theme.useSyncWithSystem(), { wrapper: wrapperFor(store) });
    await waitFor(() => expect(selectedOf(store)).toBe("synnaxLight"));

    act(() => fireChange(true));
    await waitFor(() => expect(selectedOf(store)).toBe("synnaxDark"));
  });

  it("sets the initial theme from the Tauri window in the tauri engine", async () => {
    mocks.engine = "tauri";
    tauriWindow.theme.mockResolvedValue("dark");
    tauriWindow.onThemeChanged.mockResolvedValue(() => {});
    const store = createStore();
    renderHook(() => Theme.useSyncWithSystem(), { wrapper: wrapperFor(store) });
    await waitFor(() => expect(selectedOf(store)).toBe("synnaxDark"));
  });

  it("updates the theme when Tauri reports an OS theme change", async () => {
    mocks.engine = "tauri";
    tauriWindow.theme.mockResolvedValue("light");
    let onChange: ((event: { payload: "light" | "dark" }) => void) | null = null;
    tauriWindow.onThemeChanged.mockImplementation(async (cb: typeof onChange) => {
      onChange = cb;
      return () => {};
    });
    const store = createStore();
    renderHook(() => Theme.useSyncWithSystem(), { wrapper: wrapperFor(store) });
    await waitFor(() => expect(selectedOf(store)).toBe("synnaxLight"));
    await waitFor(() => expect(onChange).not.toBeNull());

    act(() => onChange?.({ payload: "dark" }));
    await waitFor(() => expect(selectedOf(store)).toBe("synnaxDark"));
  });

  it("does not sync from a non-main tauri window", async () => {
    mocks.engine = "tauri";
    mocks.label = "child-window";
    const store = createStore();
    renderHook(() => Theme.useSyncWithSystem(), { wrapper: wrapperFor(store) });

    await act(async () => {});
    expect(tauriWindow.theme).not.toHaveBeenCalled();
    expect(selectedOf(store)).toBe("synnaxLight");
  });

  it("does not sync while syncWithSystem is disabled", async () => {
    const { mql } = createMatchMedia(true);
    vi.stubGlobal("matchMedia", vi.fn().mockReturnValue(mql));
    const store = createStore(false);
    renderHook(() => Theme.useSyncWithSystem(), { wrapper: wrapperFor(store) });

    await act(async () => {});
    expect(selectedOf(store)).toBe("synnaxLight");
  });

  it("stops syncing once syncWithSystem is toggled off", async () => {
    const { mql, fireChange } = createMatchMedia(false);
    vi.stubGlobal("matchMedia", vi.fn().mockReturnValue(mql));
    const store = createStore();
    renderHook(() => Theme.useSyncWithSystem(), { wrapper: wrapperFor(store) });
    await waitFor(() => expect(selectedOf(store)).toBe("synnaxLight"));

    act(() => {
      store.dispatch(Theme.toggleSyncWithSystem());
    });
    await act(async () => {});

    act(() => fireChange(true));
    await act(async () => {});
    expect(selectedOf(store)).toBe("synnaxLight");
  });
});
