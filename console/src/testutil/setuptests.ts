// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { configure } from "@testing-library/react";
import { afterAll, beforeAll, vi } from "vitest";

import { installTestWebSocket } from "@/testutil/websocket";

// Live-core round-trips share the single test cluster with the rest of the suite, so
// allow more than the 1s waitFor default. Full-suite runs saturate the CPU, so local
// polls also need generous headroom.
configure({ asyncUtilTimeout: 15_000 });

// jsdom does not implement ResizeObserver, and pluto's useResize constructs one
// unconditionally. Provide an inert polyfill so construction succeeds; it never
// fires, so observed elements keep jsdom's real (zero) geometry. Specs that render
// virtualized lists opt into fake geometry via stubGeometry() from @/testutil.
class ResizeObserverStub implements ResizeObserver {
  observe(): void {}
  unobserve(): void {}
  disconnect(): void {}
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
  installTestWebSocket();
  vi.stubGlobal("ResizeObserver", ResizeObserverStub);
  vi.stubGlobal("IntersectionObserver", IntersectionObserverMock);
  if (typeof globalThis.CSS === "undefined")
    vi.stubGlobal("CSS", { escape: cssEscape });
  else if (typeof globalThis.CSS.escape !== "function")
    globalThis.CSS.escape = cssEscape;
  // jsdom's Blob does not implement arrayBuffer; file-import paths read it to decode
  // file contents. Back it with the real bytes via FileReader.
  if (typeof Blob.prototype.arrayBuffer !== "function")
    Object.defineProperty(Blob.prototype, "arrayBuffer", {
      configurable: true,
      writable: true,
      value(this: Blob): Promise<ArrayBuffer> {
        return new Promise((resolve, reject) => {
          const reader = new FileReader();
          reader.onload = () => resolve(reader.result as ArrayBuffer);
          reader.onerror = () =>
            reject(reader.error ?? new Error("failed to read blob"));
          reader.readAsArrayBuffer(this);
        });
      },
    });
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
});

afterAll(() => {
  vi.clearAllMocks();
  vi.unstubAllGlobals();
});
