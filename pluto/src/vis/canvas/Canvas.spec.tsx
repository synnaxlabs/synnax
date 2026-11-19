// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { act, renderHook, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { render } from "@/testutil";
import { Canvas } from "@/vis/canvas";
import { canvas } from "@/vis/canvas/aether";
import { canvasTest } from "@/vis/render/testutil";

const Probe = () => <span>{Canvas.useGPUMissing() ? "missing" : "present"}</span>;

const offscreen = (context: unknown) =>
  Object.assign(new EventTarget(), { width: 0, height: 0, getContext: () => context });

// The canvas transfers its gl, upper 2D, and lower 2D canvases in that order.
const installOffscreen = (contexts: unknown[]) =>
  Object.defineProperty(HTMLCanvasElement.prototype, "transferControlToOffscreen", {
    configurable: true,
    value: () => offscreen(contexts.shift()),
  });

const renderCanvas = () =>
  render(
    <Canvas.Canvas resizeDebounce={0}>
      <Probe />
    </Canvas.Canvas>,
    { registry: canvas.REGISTRY },
  );

describe("Canvas", () => {
  const resizes: (() => void)[] = [];
  const BaseResizeObserver = globalThis.ResizeObserver;

  beforeEach(() => {
    vi.stubGlobal(
      "ResizeObserver",
      class {
        constructor(cb: () => void) {
          resizes.push(cb);
        }

        observe(): void {}
        disconnect(): void {}
      },
    );
  });

  afterEach(() => {
    resizes.length = 0;
    vi.stubGlobal("ResizeObserver", BaseResizeObserver);
    vi.restoreAllMocks();
    Reflect.deleteProperty(HTMLCanvasElement.prototype, "transferControlToOffscreen");
  });

  describe("useGPUMissing", () => {
    it("should return false outside a canvas", () => {
      const { result } = renderHook(() => Canvas.useGPUMissing());
      expect(result.current).toBe(false);
    });

    it("should return false when the canvas gets a WebGL context", async () => {
      const recorder = canvasTest.record();
      installOffscreen([canvasTest.createGL().gl, recorder.upper2d, recorder.lower2d]);
      renderCanvas();
      expect(await screen.findByText("present")).toBeDefined();
    });

    it("should return true when the canvas gets no WebGL context", async () => {
      installOffscreen([null, null, null]);
      renderCanvas();
      expect(await screen.findByText("missing")).toBeDefined();
    });

    it("should stay true after a resize", async () => {
      installOffscreen([null, null, null]);
      vi.spyOn(HTMLElement.prototype, "getBoundingClientRect").mockReturnValue(
        new DOMRect(0, 0, 200, 100),
      );
      renderCanvas();
      await screen.findByText("missing");
      await act(async () => {
        resizes.forEach((resize) => resize());
        await new Promise((resolve) => setTimeout(resolve, 10));
      });
      expect(screen.getByText("missing")).toBeDefined();
    });
  });
});
