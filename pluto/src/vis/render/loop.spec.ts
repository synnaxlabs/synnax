// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { type CanvasVariant } from "@/vis/render/context";
import { type Cleanup, Loop, type Request } from "@/vis/render/loop";

const create = () => {
  const afterRender = vi.fn();
  const loop = new Loop({ afterRender, handleError: vi.fn() });
  return { loop, afterRender };
};

const request = (
  key: string,
  canvases: CanvasVariant[],
  cleanup?: Cleanup,
): Request & { render: ReturnType<typeof vi.fn> } => ({
  key,
  priority: "high",
  canvases,
  render: vi.fn(() => cleanup),
});

describe("Loop", () => {
  beforeEach(() => {
    vi.useFakeTimers({ toFake: ["requestAnimationFrame"] });
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  describe("while the GL context is lost", () => {
    it("should hold back only the requests that draw on the GL canvas", () => {
      const { loop } = create();
      const plot = request("plot", ["upper2d", "gl"]);
      const diagram = request("diagram", ["upper2d", "lower2d"]);
      loop.glLost = true;
      loop.set(plot);
      loop.set(diagram);
      vi.advanceTimersToNextFrame();
      expect(plot.render).not.toHaveBeenCalled();
      expect(diagram.render).toHaveBeenCalledOnce();
    });

    it("should not call afterRender", () => {
      const { loop, afterRender } = create();
      loop.glLost = true;
      loop.set(request("diagram", ["lower2d"]));
      vi.advanceTimersToNextFrame();
      expect(afterRender).not.toHaveBeenCalled();
    });

    it("should hold back the cleanup of a held request", () => {
      const { loop } = create();
      const cleanup = vi.fn();
      loop.set(request("plot", ["gl"], cleanup));
      vi.advanceTimersToNextFrame();
      loop.glLost = true;
      loop.set(request("plot", ["gl"]));
      vi.advanceTimersToNextFrame();
      expect(cleanup).not.toHaveBeenCalled();
    });
  });

  describe("once the GL context is restored", () => {
    it("should run the cleanup and the render of each held request", () => {
      const { loop, afterRender } = create();
      const cleanup = vi.fn();
      loop.set(request("plot", ["gl"], cleanup));
      vi.advanceTimersToNextFrame();
      loop.glLost = true;
      const held = request("plot", ["gl"]);
      loop.set(held);
      vi.advanceTimersToNextFrame();
      expect(held.render).not.toHaveBeenCalled();
      loop.glLost = false;
      vi.advanceTimersToNextFrame();
      expect(cleanup).toHaveBeenCalledExactlyOnceWith(held);
      expect(held.render).toHaveBeenCalledOnce();
      expect(afterRender).toHaveBeenCalledTimes(2);
    });
  });
});
