// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { afterAll, beforeAll } from "vitest";

const RECT: DOMRect = {
  top: 0,
  left: 0,
  width: 100,
  height: 100,
  bottom: 100,
  right: 100,
  x: 0,
  y: 0,
  toJSON: () => "",
};

class SizeFiringResizeObserver implements ResizeObserver {
  private readonly callback: ResizeObserverCallback;

  constructor(callback: ResizeObserverCallback) {
    this.callback = callback;
  }

  observe(target: Element): void {
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
  }

  unobserve(): void {}

  disconnect(): void {}
}

/**
 * Replaces jsdom's zero-size layout with fake 100x100 geometry for the calling spec
 * file: every getBoundingClientRect returns a 100x100 rect and ResizeObserver fires a
 * matching size on observe. Call once at spec top level; installs in beforeAll and
 * restores in afterAll.
 *
 * Opt in only when the spec renders virtualized lists/trees that need a non-zero
 * container size to mount their items. A spec that opts in must never assert measured
 * pixel geometry — every measurement is fake.
 */
export const stubGeometry = (): void => {
  let originalGetBoundingClientRect: PropertyDescriptor | undefined;
  let OriginalResizeObserver: typeof ResizeObserver;
  beforeAll(() => {
    originalGetBoundingClientRect = Object.getOwnPropertyDescriptor(
      Element.prototype,
      "getBoundingClientRect",
    );
    OriginalResizeObserver = globalThis.ResizeObserver;
    Object.defineProperty(Element.prototype, "getBoundingClientRect", {
      configurable: true,
      writable: true,
      value: () => RECT,
    });
    globalThis.ResizeObserver = SizeFiringResizeObserver;
  });
  afterAll(() => {
    if (originalGetBoundingClientRect != null)
      Object.defineProperty(
        Element.prototype,
        "getBoundingClientRect",
        originalGetBoundingClientRect,
      );
    globalThis.ResizeObserver = OriginalResizeObserver;
  });
};
