// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { act, fireEvent, render } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { Notation } from "@/notation";

// Reports whether the labels give way to icons, as a narrow window does.
const showIcons = (icons: boolean): void => {
  vi.stubGlobal("matchMedia", (media: string): MediaQueryList =>
    Object.assign(new EventTarget(), {
      media,
      matches: icons,
      onchange: null,
      addListener: () => {},
      removeListener: () => {},
    }),
  );
};

// Gives the element a size, since a tooltip does not open over a zero-size element.
const BUTTON_RECT = {
  x: 0,
  y: 0,
  top: 0,
  left: 0,
  right: 80,
  bottom: 24,
  width: 80,
  height: 24,
  toJSON: () => ({}),
} as DOMRect;

const hover = (el: HTMLElement): string | null | undefined => {
  el.getBoundingClientRect = () => BUTTON_RECT;
  vi.useFakeTimers();
  fireEvent.pointerOver(el, { pointerType: "mouse" });
  act(() => {
    vi.advanceTimersByTime(2000);
  });
  vi.useRealTimers();
  return document.querySelector(".lyra-tooltip")?.textContent;
};

describe("Notation.Select", () => {
  afterEach(() => showIcons(false));

  it("should show no tooltip while its buttons show labels", () => {
    showIcons(false);
    const c = render(<Notation.Select value="standard" onChange={() => {}} />);
    expect(hover(c.getByRole("button", { name: /Engineering/ }))).toBeUndefined();
  });

  it("should name each button in a tooltip while it shows icons", () => {
    showIcons(true);
    const c = render(<Notation.Select value="standard" onChange={() => {}} />);
    expect(hover(c.getByRole("button", { name: /Engineering/ }))).toBe("Engineering");
  });
});
