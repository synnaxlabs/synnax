// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { act, fireEvent, render } from "@testing-library/react";
import { type ReactElement, useState } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { Input } from "@/input";
import { Theming } from "@/theming";

const TestThemeContent = (): ReactElement => {
  const { theme, toggleTheme } = Theming.useContext();
  const [checked, setChecked] = useState(false);
  const handleChange = (value: boolean) => {
    setChecked(value);
    toggleTheme();
  };
  return (
    <div>
      {theme.name}
      <Input.Switch aria-label="theme-switch" value={checked} onChange={handleChange} />
    </div>
  );
};

const TestTheme = (): ReactElement => (
  <Theming.Provider>
    <TestThemeContent />
  </Theming.Provider>
);

describe("Theming", () => {
  it("should render a theme", () => {
    const { getByText } = render(<TestTheme />);
    expect(getByText("Synnax Dark")).toBeTruthy();
  });
  it("should toggle a theme", () => {
    const { getByText, getByLabelText } = render(<TestTheme />);
    expect(getByText("Synnax Dark")).toBeTruthy();
    const btn = getByLabelText("theme-switch");
    fireEvent.click(btn);
    expect(getByText("Synnax Light")).toBeTruthy();
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
      const { getByText } = render(
        <Theming.Provider>
          <TestThemeContent />
        </Theming.Provider>,
      );
      expect(getByText("Synnax Light")).toBeTruthy();
      act(() => fireOSChange(true));
      expect(getByText("Synnax Dark")).toBeTruthy();
    });

    it("ignores OS changes once a caller pins an explicit theme", () => {
      vi.stubGlobal("matchMedia", vi.fn().mockReturnValue(mql));
      const { getByText } = render(
        <Theming.Provider theme={{ key: "synnaxLight" }}>
          <TestThemeContent />
        </Theming.Provider>,
      );
      expect(getByText("Synnax Light")).toBeTruthy();
      act(() => fireOSChange(true));
      expect(getByText("Synnax Light")).toBeTruthy();
    });
  });
});
