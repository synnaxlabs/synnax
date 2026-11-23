// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { act } from "@testing-library/react";
import { vi } from "vitest";

/** The width of the elements that a spec lays out with {@link stubWidth}. */
export interface Width {
  /** Changes the width and notifies every resize observer. */
  set: (width: number) => void;
  /** Puts back the 100px layout and the resize observer of the suite. */
  restore: () => void;
}

/**
 * Lays out each element with the class at the given width, and every other element
 * at the 100px of the suite. Each resize observer reports a change to the width.
 */
export const stubWidth = (className: string, initial: number): Width => {
  let width = initial;
  const notifiers = new Set<() => void>();
  class Observer {
    private readonly notify: () => void;
    constructor(callback: ResizeObserverCallback) {
      this.notify = () => callback([], this);
    }
    observe(): void {
      notifiers.add(this.notify);
      this.notify();
    }
    unobserve(): void {}
    disconnect(): void {
      notifiers.delete(this.notify);
    }
  }
  const previous = globalThis.ResizeObserver;
  vi.stubGlobal("ResizeObserver", Observer);
  const measure = vi
    .spyOn(Element.prototype, "getBoundingClientRect")
    .mockImplementation(function (this: Element) {
      return new DOMRect(0, 0, this.classList.contains(className) ? width : 100, 100);
    });
  return {
    set: (next) => {
      width = next;
      act(() => notifiers.forEach((notify) => notify()));
    },
    restore: () => {
      measure.mockRestore();
      vi.stubGlobal("ResizeObserver", previous);
    },
  };
};
