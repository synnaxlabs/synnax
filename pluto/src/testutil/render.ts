// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

import { aether } from "@/aether/aether";
import { CSS } from "@/css";

/**
 * A vitest-free spy: a callable that records the arguments of every invocation on
 * `calls`. Specs that need vitest matchers can wrap one with `vi.fn` or assert on
 * `calls` directly.
 */
export interface RecordingFn {
  (...args: unknown[]): unknown;
  /** Arguments of each invocation, in call order. */
  calls: unknown[][];
}

/**
 * Creates a {@link RecordingFn} that records every call and delegates to impl for the
 * return value.
 */
export const createRecordingFn = (
  impl?: (...args: unknown[]) => unknown,
): RecordingFn => {
  const fn: RecordingFn = Object.assign(
    (...args: unknown[]): unknown => {
      fn.calls.push(args);
      return impl?.(...args);
    },
    { calls: [] as unknown[][] },
  );
  return fn;
};

export interface MockRenderContext {
  loop: { set: RecordingFn };
  erase: RecordingFn;
  scissor: RecordingFn;
  lower2d: Record<string, unknown>;
}

// Stubs for jsdom which doesn't implement CanvasRenderingContext2D.
// Used by aether components that render directly to a 2D canvas context.
export const mockCanvas2DContext = (): Record<string, unknown> => ({
  setLineDash: createRecordingFn(),
  beginPath: createRecordingFn(),
  closePath: createRecordingFn(),
  stroke: createRecordingFn(),
  fill: createRecordingFn(),
  rect: createRecordingFn(),
  roundRect: createRecordingFn(),
  fillText: createRecordingFn(),
  strokeRect: createRecordingFn(),
  fillRect: createRecordingFn(),
  save: createRecordingFn(),
  restore: createRecordingFn(),
  clip: createRecordingFn(),
  clearRect: createRecordingFn(),
  moveTo: createRecordingFn(),
  lineTo: createRecordingFn(),
  arc: createRecordingFn(),
  fillStyle: "",
  strokeStyle: "",
  lineWidth: 1,
  font: "",
  measureText: createRecordingFn(() => ({ width: 8 })),
});

export const mockRenderContext = (): MockRenderContext => {
  const ctx2d = mockCanvas2DContext();
  return {
    loop: { set: createRecordingFn() },
    erase: createRecordingFn(),
    scissor: createRecordingFn(() => createRecordingFn()),
    lower2d: {
      canvas: { width: 800, height: 600 },
      getContext: createRecordingFn(() => ctx2d),
      ...ctx2d,
      font: "",
      scissor: createRecordingFn(() => createRecordingFn()),
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
