// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { act, renderHook } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { Theming } from "@/theming";

const renderTheming = (theme?: Theming.ProviderProps["theme"]) =>
  renderHook(() => Theming.useContext(), {
    wrapper: ({ children }: PropsWithChildren): ReactElement => (
      <Theming.Provider theme={theme}>{children}</Theming.Provider>
    ),
  });

describe("Theming", () => {
  it("should provide the dark theme by default", () => {
    const { result } = renderTheming();
    expect(result.current.theme.name).toBe("Synnax Dark");
  });

  it("should toggle a theme", () => {
    const { result } = renderTheming();
    expect(result.current.theme.name).toBe("Synnax Dark");
    act(() => result.current.toggleTheme());
    expect(result.current.theme.name).toBe("Synnax Light");
  });

  describe("OS color-scheme sync", () => {
    let matches = false;
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
    const fireOSChange = (next: boolean): void => {
      matches = next;
      listener?.({ matches: next } as MediaQueryListEvent);
    };

    afterEach(() => {
      matches = false;
      listener = null;
      vi.unstubAllGlobals();
    });

    it("follows OS changes when no theme prop is supplied", () => {
      vi.stubGlobal("matchMedia", vi.fn().mockReturnValue(mql));
      const { result } = renderTheming();
      expect(result.current.theme.name).toBe("Synnax Light");
      act(() => fireOSChange(true));
      expect(result.current.theme.name).toBe("Synnax Dark");
    });

    it("ignores OS changes once a caller pins an explicit theme", () => {
      vi.stubGlobal("matchMedia", vi.fn().mockReturnValue(mql));
      const { result } = renderTheming({ key: "synnaxLight" });
      expect(result.current.theme.name).toBe("Synnax Light");
      act(() => fireOSChange(true));
      expect(result.current.theme.name).toBe("Synnax Light");
    });
  });
});
