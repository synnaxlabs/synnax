// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Mock, vi } from "vitest";
import { z } from "zod";

import { aether } from "@/aether/aether";
import { CSS } from "@/css";

export interface MockRenderContext {
  loop: { set: Mock };
  erase: Mock;
  scissor: Mock;
  lower2d: Record<string, unknown>;
}

// Stubs for jsdom which doesn't implement CanvasRenderingContext2D.
// Used by aether components that render directly to a 2D canvas context.
export const mockCanvas2DContext = (): Record<string, unknown> => ({
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

export const mockRenderContext = (): MockRenderContext => {
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

const RENDER_CONTEXT_KEY = CSS.B("render-context");

const mockRenderContexts = new Map<string, MockRenderContext>();

/**
 * Registers a mock render context under key so a {@link MockRenderContextProvider}
 * whose state carries the same key can seed it into the aether tree.
 */
export const registerMockRenderContext = (
  key: string,
  ctx: MockRenderContext,
): void => {
  mockRenderContexts.set(key, ctx);
};

export const mockRenderContextProviderStateZ = z.object({ contextKey: z.string() });

/**
 * Aether composite that publishes a registered {@link MockRenderContext} to its
 * descendants under the render context key, standing in for the real canvas provider.
 */
export class MockRenderContextProvider extends aether.Composite<
  typeof mockRenderContextProviderStateZ
> {
  static readonly TYPE = "mock-render-context-provider";
  schema = mockRenderContextProviderStateZ;

  afterUpdate(ctx: aether.Context): void {
    const mock = mockRenderContexts.get(this.state.contextKey);
    if (mock == null)
      throw new Error(
        `no mock render context registered for key ${this.state.contextKey}`,
      );
    ctx.set(RENDER_CONTEXT_KEY, mock);
  }
}

export const MOCK_RENDER_CONTEXT_REGISTRY: aether.ComponentRegistry = {
  [MockRenderContextProvider.TYPE]: MockRenderContextProvider,
};
