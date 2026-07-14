// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { act, renderHook, waitFor } from "@testing-library/react";
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
  addEventListener: ReturnType<typeof vi.fn>;
}

const createMatchMedia = (initialMatches: boolean): FakeMediaQueryList => {
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
  return {
    mql,
    addEventListener,
    fireChange: (next: boolean) => {
      matches = next;
      listener?.({ matches: next } as MediaQueryListEvent);
    },
  };
};

describe("useOSTheme", () => {
  beforeEach(() => {
    mocks.engine = "web";
    mocks.label = "main";
    tauriWindow.theme.mockReset();
    tauriWindow.onThemeChanged.mockReset();
  });

  afterEach(() => vi.unstubAllGlobals());

  it("resolves the dark theme from the OS preference in the web engine", async () => {
    const { mql } = createMatchMedia(true);
    vi.stubGlobal("matchMedia", vi.fn().mockReturnValue(mql));
    const { result } = renderHook(() => Theme.useOSTheme(true));
    await waitFor(() => expect(result.current).toBe("synnaxDark"));
  });

  it("updates when the OS preference changes in the web engine", async () => {
    const { mql, fireChange } = createMatchMedia(false);
    vi.stubGlobal("matchMedia", vi.fn().mockReturnValue(mql));
    const { result } = renderHook(() => Theme.useOSTheme(true));
    await waitFor(() => expect(result.current).toBe("synnaxLight"));

    act(() => fireChange(true));
    await waitFor(() => expect(result.current).toBe("synnaxDark"));
  });

  it("resolves the theme from the Tauri window in the tauri engine", async () => {
    mocks.engine = "tauri";
    tauriWindow.theme.mockResolvedValue("dark");
    tauriWindow.onThemeChanged.mockResolvedValue(() => {});
    const { result } = renderHook(() => Theme.useOSTheme(true));
    await waitFor(() => expect(result.current).toBe("synnaxDark"));
  });

  it("updates when Tauri reports an OS theme change", async () => {
    mocks.engine = "tauri";
    tauriWindow.theme.mockResolvedValue("light");
    let onChange: ((event: { payload: "light" | "dark" }) => void) | null = null;
    tauriWindow.onThemeChanged.mockImplementation(async (cb: typeof onChange) => {
      onChange = cb;
      return () => {};
    });
    const { result } = renderHook(() => Theme.useOSTheme(true));
    await waitFor(() => expect(result.current).toBe("synnaxLight"));
    await waitFor(() => expect(onChange).not.toBeNull());

    act(() => onChange?.({ payload: "dark" }));
    await waitFor(() => expect(result.current).toBe("synnaxDark"));
  });

  it("does not subscribe while disabled", async () => {
    const { mql, addEventListener } = createMatchMedia(true);
    vi.stubGlobal("matchMedia", vi.fn().mockReturnValue(mql));
    const { result } = renderHook(() => Theme.useOSTheme(false));

    await act(async () => {});
    expect(addEventListener).not.toHaveBeenCalled();
    expect(result.current).toBe("synnaxDark");
  });

  it("stops updating once disabled", async () => {
    const { mql, fireChange } = createMatchMedia(false);
    vi.stubGlobal("matchMedia", vi.fn().mockReturnValue(mql));
    const { result, rerender } = renderHook(
      ({ enabled }) => Theme.useOSTheme(enabled),
      { initialProps: { enabled: true } },
    );
    await waitFor(() => expect(result.current).toBe("synnaxLight"));

    rerender({ enabled: false });
    await act(async () => {});
    act(() => fireChange(true));
    await act(async () => {});
    expect(result.current).toBe("synnaxLight");
  });
});
