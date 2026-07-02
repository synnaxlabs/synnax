// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { afterAll, beforeAll, vi } from "vitest";

class ResizeObserverMock {
  callback: ResizeObserverCallback;
  constructor(callback: ResizeObserverCallback) {
    this.callback = callback;
  }
  observe = vi.fn((target: Element) => {
    // Fire the callback so virtualizers see a non-zero container size
    this.callback(
      [
        {
          target,
          contentRect: target.getBoundingClientRect(),
          borderBoxSize: [{ blockSize: 100, inlineSize: 100 }],
          contentBoxSize: [{ blockSize: 100, inlineSize: 100 }],
          devicePixelContentBoxSize: [{ blockSize: 100, inlineSize: 100 }],
        },
      ],
      this,
    );
  });
  unobserve = vi.fn();
  disconnect = vi.fn();
}

class IntersectionObserverMock {
  observe = vi.fn();
  unobserve = vi.fn();
  disconnect = vi.fn();
}

// jsdom does not implement CSS.escape, which pluto relies on to build id selectors
// (e.g. Text.asyncEdit). Provide the standard CSSOM serialization algorithm.
const cssEscape = (value: string): string => {
  const string = String(value);
  const { length } = string;
  let result = "";
  let index = -1;
  const firstCodeUnit = string.charCodeAt(0);
  while (++index < length) {
    const codeUnit = string.charCodeAt(index);
    if (codeUnit === 0x0000) {
      result += "�";
      continue;
    }
    if (
      (codeUnit >= 0x0001 && codeUnit <= 0x001f) ||
      codeUnit === 0x007f ||
      (index === 0 && codeUnit >= 0x0030 && codeUnit <= 0x0039) ||
      (index === 1 &&
        codeUnit >= 0x0030 &&
        codeUnit <= 0x0039 &&
        firstCodeUnit === 0x002d)
    ) {
      result += `\\${codeUnit.toString(16)} `;
      continue;
    }
    if (index === 0 && length === 1 && codeUnit === 0x002d) {
      result += `\\${string.charAt(index)}`;
      continue;
    }
    if (
      codeUnit >= 0x0080 ||
      codeUnit === 0x002d ||
      codeUnit === 0x005f ||
      (codeUnit >= 0x0030 && codeUnit <= 0x0039) ||
      (codeUnit >= 0x0041 && codeUnit <= 0x005a) ||
      (codeUnit >= 0x0061 && codeUnit <= 0x007a)
    ) {
      result += string.charAt(index);
      continue;
    }
    result += `\\${string.charAt(index)}`;
  }
  return result;
};

beforeAll(() => {
  vi.stubGlobal("ResizeObserver", ResizeObserverMock);
  vi.stubGlobal("IntersectionObserver", IntersectionObserverMock);
  if (typeof globalThis.CSS === "undefined")
    vi.stubGlobal("CSS", { escape: cssEscape });
  else if (typeof globalThis.CSS.escape !== "function")
    globalThis.CSS.escape = cssEscape;
  // jsdom does not implement innerText; components that read/commit editable text
  // (pluto's Text.Editable) rely on it. Delegate to textContent.
  if (!Object.hasOwn(HTMLElement.prototype, "innerText"))
    Object.defineProperty(HTMLElement.prototype, "innerText", {
      configurable: true,
      get(this: HTMLElement) {
        return this.textContent ?? "";
      },
      set(this: HTMLElement, value: string) {
        this.textContent = value;
      },
    });
  Element.prototype.getBoundingClientRect = vi.fn().mockReturnValue({
    top: 0,
    left: 0,
    width: 100,
    height: 100,
    bottom: 100,
    right: 100,
    x: 0,
    y: 0,
    toJSON: () => "",
  });
});

afterAll(() => {
  vi.clearAllMocks();
});
