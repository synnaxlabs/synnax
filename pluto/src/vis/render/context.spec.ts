// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { box, type runtime } from "@synnaxlabs/x";
import { afterEach, assert, beforeEach, describe, expect, it, vi } from "vitest";

import { aetherTest } from "@/aether/test";
import { buildStack } from "@/testutil/providers";
import { canvas } from "@/vis/canvas/aether";
import { canvasTest } from "@/vis/render/test";

const { GL_ENUMS } = canvasTest;

/** Stands in for a transferred canvas, which jsdom does not provide. */
class FakeCanvas extends EventTarget {
  width = 0;
  height = 0;
  private readonly context: unknown;

  constructor(context: unknown) {
    super();
    this.context = context;
  }

  getContext(): unknown {
    return this.context;
  }
}

const teardowns: (() => void)[] = [];

const mount = (os: runtime.OS = "macOS") => {
  const returns: Record<string, () => unknown> = {};
  const { gl, calls } = canvasTest.createGL(returns);
  const glCanvas = new FakeCanvas(gl);
  const recorder = canvasTest.record();
  const stack = buildStack({ registry: canvas.REGISTRY });
  teardowns.push(() => stack.driver.delete([aetherTest.ROOT_KEY]));
  const path = [...stack.basePath, "canvas"];
  stack.driver.update(path, canvas.Canvas.TYPE, {
    bootstrap: true,
    glCanvas,
    lower2dCanvas: new FakeCanvas(recorder.lower2d),
    upper2dCanvas: new FakeCanvas(recorder.upper2d),
    region: box.construct(0, 0, 100, 100),
    dpr: 1,
    os,
  });
  const { renderCtx } = stack.driver.find<canvas.Canvas>(path);
  assert(renderCtx != null);
  const statuses = () => stack.providers.status?.state.statuses ?? [];
  const count = (op: string) => calls.filter((c) => c.op === op).length;
  const lose = (): Event => {
    const event = new Event("webglcontextlost", { cancelable: true });
    glCanvas.dispatchEvent(event);
    return event;
  };
  const restore = () => glCanvas.dispatchEvent(new Event("webglcontextrestored"));
  return { renderCtx, calls, returns, statuses, count, lose, restore };
};

const glRequest = () => ({
  key: "plot",
  priority: "high" as const,
  canvases: ["gl" as const],
  render: vi.fn(),
});

describe("render.Context", () => {
  beforeEach(() => {
    vi.useFakeTimers({ toFake: ["requestAnimationFrame"] });
  });

  afterEach(() => {
    for (const teardown of teardowns) teardown();
    teardowns.length = 0;
    vi.useRealTimers();
  });

  describe("when the WebGL context is lost", () => {
    it("should cancel the event so that the browser restores the context", () => {
      const m = mount();
      expect(m.lose().defaultPrevented).toBe(true);
    });

    it("should add an error status", () => {
      const m = mount();
      m.lose();
      expect(m.statuses()).toEqual([
        expect.objectContaining({
          variant: "error",
          message: "WebGL context lost",
          description: "Plots draw again once the browser restores the context.",
        }),
      ]);
    });

    it("should hold back renders that draw on the GL canvas", () => {
      const m = mount();
      m.lose();
      const req = glRequest();
      m.renderCtx.loop.set(req);
      vi.advanceTimersToNextFrame();
      expect(req.render).not.toHaveBeenCalled();
    });
  });

  describe("when the WebGL context is restored", () => {
    it("should increment the generation", () => {
      const m = mount();
      m.lose();
      m.restore();
      expect(m.renderCtx.generation).toBe(1);
    });

    it("should compile the line programs again", () => {
      const m = mount();
      const compiled = m.count("createProgram");
      m.lose();
      m.restore();
      expect(m.count("createProgram")).toBe(compiled + 2);
    });

    it("should compile the clear program again on Windows", () => {
      const m = mount("Windows");
      const compiled = m.count("createProgram");
      m.lose();
      m.restore();
      expect(m.count("createProgram")).toBe(compiled + 3);
    });

    it("should disable the GL state that a new context enables", () => {
      const m = mount();
      m.lose();
      const start = m.calls.length;
      m.restore();
      const disabled = m.calls
        .slice(start)
        .filter((c) => c.op === "disable")
        .map((c) => c.args[0]);
      expect(disabled).toEqual([
        GL_ENUMS.DEPTH_TEST,
        GL_ENUMS.STENCIL_TEST,
        GL_ENUMS.CULL_FACE,
        GL_ENUMS.DITHER,
      ]);
    });

    it("should call each subscribed restore handler", () => {
      const m = mount();
      const handler = vi.fn();
      const removed = vi.fn();
      m.renderCtx.onRestore(handler);
      m.renderCtx.onRestore(removed)();
      m.lose();
      m.restore();
      expect(handler).toHaveBeenCalledOnce();
      expect(removed).not.toHaveBeenCalled();
    });

    it("should render the held GL renders", () => {
      const m = mount();
      m.lose();
      const req = glRequest();
      m.renderCtx.loop.set(req);
      vi.advanceTimersToNextFrame();
      expect(req.render).not.toHaveBeenCalled();
      m.restore();
      vi.advanceTimersToNextFrame();
      expect(req.render).toHaveBeenCalledOnce();
    });

    it("should replace the error status with a success status", () => {
      const m = mount();
      m.lose();
      const [lost] = m.statuses();
      m.restore();
      expect(m.statuses()).toEqual([
        expect.objectContaining({
          key: lost.key,
          variant: "success",
          message: "WebGL context restored",
        }),
      ]);
    });

    it("should keep holding back GL renders when the restore fails", () => {
      const m = mount();
      m.lose();
      m.returns.createProgram = () => null;
      m.restore();
      const req = glRequest();
      m.renderCtx.loop.set(req);
      vi.advanceTimersToNextFrame();
      expect(req.render).not.toHaveBeenCalled();
      expect(m.statuses().at(-1)).toEqual(
        expect.objectContaining({
          variant: "error",
          message: "Failed to restore WebGL context",
        }),
      );
    });
  });
});
