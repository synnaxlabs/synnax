// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { box, scale, xy } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { canvasTest } from "@/vis/render/test";

interface DrawableCtx {
  fillRect(x: number, y: number, w: number, h: number): void;
  strokeStyle: string;
  beginPath(): void;
  closePath(): void;
}

describe("canvasTest.record", () => {
  describe("2D context recording", () => {
    it("records a method call as { op, args }", () => {
      const r = canvasTest.record();
      const ctx = r.upper2d as unknown as DrawableCtx;
      ctx.fillRect(1, 2, 3, 4);
      expect(r.upper2d.calls).toEqual([{ op: "fillRect", args: [1, 2, 3, 4] }]);
    });

    it("records property assignments under set:<prop>", () => {
      const r = canvasTest.record();
      const ctx = r.lower2d as unknown as DrawableCtx;
      ctx.strokeStyle = "red";
      expect(r.lower2d.calls).toEqual([{ op: "set:strokeStyle", args: ["red"] }]);
    });

    it("preserves property reads after assignment", () => {
      const r = canvasTest.record();
      const ctx = r.upper2d as unknown as DrawableCtx;
      ctx.strokeStyle = "red";
      expect(ctx.strokeStyle).toBe("red");
    });

    it("records multiple calls in encounter order across both 2D contexts", () => {
      const r = canvasTest.record();
      const upper = r.upper2d as unknown as DrawableCtx;
      const lower = r.lower2d as unknown as DrawableCtx;
      upper.beginPath();
      lower.fillRect(0, 0, 10, 10);
      upper.closePath();
      expect(r.upper2d.calls).toEqual([
        { op: "beginPath", args: [] },
        { op: "closePath", args: [] },
      ]);
      expect(r.lower2d.calls).toEqual([{ op: "fillRect", args: [0, 0, 10, 10] }]);
    });

    it("records applyScale and returns the same recording surface", () => {
      const r = canvasTest.record();
      const scaled = r.lower2d.applyScale(scale.XY.IDENTITY);
      (scaled as unknown as DrawableCtx).fillRect(1, 1, 2, 2);
      expect(scaled).toBe(r.lower2d);
      expect(r.lower2d.calls).toEqual([
        { op: "applyScale", args: [scale.XY.IDENTITY] },
        { op: "fillRect", args: [1, 1, 2, 2] },
      ]);
    });

    it("records textDimensions and returns a length-derived size", () => {
      const r = canvasTest.record();
      expect(r.lower2d.textDimensions("abcd")).toEqual({ width: 32, height: 12 });
      expect(r.lower2d.calls).toEqual([{ op: "textDimensions", args: ["abcd"] }]);
    });
  });

  describe("scissor / erase / loop recording", () => {
    it("records scissor calls with region, overScan, and canvases", () => {
      const r = canvasTest.record();
      const region = box.construct({ x: 0, y: 0 }, { x: 10, y: 10 });
      r.scissor(region, xy.construct({ x: 1, y: 2 }), ["upper2d"]);
      expect(r.scissorCalls).toEqual([
        { region, overScan: { x: 1, y: 2 }, canvases: ["upper2d"] },
      ]);
    });

    it("records erase calls with default canvases when none supplied", () => {
      const r = canvasTest.record();
      const region = box.construct({ x: 0, y: 0 }, { x: 5, y: 5 });
      r.erase(region);
      expect(r.eraseCalls).toEqual([
        { region, overScan: xy.ZERO, canvases: ["upper2d", "lower2d", "gl"] },
      ]);
    });

    it("records loop.set calls with raw args", () => {
      const r = canvasTest.record();
      const fn = () => {};
      r.loop.set("key", fn);
      expect(r.loopCalls).toEqual([{ args: ["key", fn] }]);
    });
  });

  describe("clear", () => {
    it("resets every recording", () => {
      const r = canvasTest.record();
      const ctx = r.upper2d as unknown as DrawableCtx;
      ctx.fillRect(0, 0, 1, 1);
      r.scissor(box.ZERO);
      r.erase(box.ZERO);
      r.loop.set("k", () => {});
      r.clear();
      expect(r.upper2d.calls).toHaveLength(0);
      expect(r.scissorCalls).toHaveLength(0);
      expect(r.eraseCalls).toHaveLength(0);
      expect(r.loopCalls).toHaveLength(0);
    });
  });

  describe("resize", () => {
    it("updates region and dpr", () => {
      const r = canvasTest.record();
      const region = box.construct({ x: 0, y: 0 }, { x: 100, y: 200 });
      r.resize(region, 2);
      expect(r.region).toEqual(region);
      expect(r.dpr).toBe(2);
    });
  });
});
