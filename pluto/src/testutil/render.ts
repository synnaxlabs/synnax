// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { vi } from "vitest";

// Stubs for jsdom which doesn't implement CanvasRenderingContext2D.
// Used by aether components that render directly to a 2D canvas context.
export const mockCanvas2DContext = () => ({
  setLineDash: vi.fn(),
  beginPath: vi.fn(),
  closePath: vi.fn(),
  stroke: vi.fn(),
  fill: vi.fn(),
  rect: vi.fn(),
  roundRect: vi.fn(),
  fillText: vi.fn(),
  strokeRect: vi.fn(),
  fillRect: vi.fn(),
  save: vi.fn(),
  restore: vi.fn(),
  clip: vi.fn(),
  clearRect: vi.fn(),
  moveTo: vi.fn(),
  lineTo: vi.fn(),
  arc: vi.fn(),
  fillStyle: "",
  strokeStyle: "",
  lineWidth: 1,
  font: "",
  measureText: vi.fn(() => ({ width: 8 })),
});

export const mockRenderContext = () => {
  const ctx2d = mockCanvas2DContext();
  return {
    loop: { set: vi.fn() },
    erase: vi.fn(),
    scissor: vi.fn(() => vi.fn()),
    lower2d: {
      canvas: { width: 800, height: 600 },
      getContext: vi.fn(() => ctx2d),
      ...ctx2d,
      font: "",
      scissor: vi.fn(() => vi.fn()),
    },
  };
};
