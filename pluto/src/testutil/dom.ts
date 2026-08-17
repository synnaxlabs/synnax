// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type xy } from "@synnaxlabs/x";
import { createEvent, fireEvent } from "@testing-library/react";
import { type Mock, vi } from "vitest";

/**
 * Fires a drag event of the given type at the given cursor position on target. jsdom
 * has no DragEvent, so testing-library falls back to a plain Event and the coordinates
 * in the init are lost; they are defined on the event instead.
 */
export const fireDragEvent = (
  target: Element,
  type: "dragOver" | "drop",
  cursor: xy.XY,
): void => {
  const event = createEvent[type](target);
  Object.defineProperties(event, {
    clientX: { value: cursor.x },
    clientY: { value: cursor.y },
    screenX: { value: cursor.x },
    screenY: { value: cursor.y },
  });
  fireEvent(target, event);
};

/**
 * Gives every element a fixed box in jsdom, which otherwise reports zero for all of
 * them. A virtualized list measures its scroll container this way, so without it no
 * item is ever inside the window.
 */
export const mockGeometry = (width: number, height: number): void => {
  Element.prototype.getBoundingClientRect = mockBoundingClientRect(0, 0, width, height);
  Object.defineProperty(HTMLElement.prototype, "offsetHeight", {
    configurable: true,
    get: () => height,
  });
  Object.defineProperty(HTMLElement.prototype, "offsetWidth", {
    configurable: true,
    get: () => width,
  });
};

export const mockBoundingClientRect = (
  top: number,
  left: number,
  width: number,
  height: number,
): Mock<typeof HTMLElement.prototype.getBoundingClientRect> =>
  vi.fn().mockReturnValue({
    top,
    left,
    width,
    height,
    bottom: top + height,
    right: left + width,
    x: 0,
    y: 0,
    toJSON: () => "",
  });
