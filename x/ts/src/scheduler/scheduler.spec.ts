// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { afterEach, describe, expect, it, vi } from "vitest";

import { scheduler } from "@/scheduler";

describe("scheduler", () => {
  describe("nextMicrotask", () => {
    it("should return a resolved promise", async () => {
      const result = scheduler.nextMicrotask();
      expect(result).toBeInstanceOf(Promise);
      await expect(result).resolves.toBeUndefined();
    });

    it("should flush microtasks before continuing", async () => {
      const order: number[] = [];
      void Promise.resolve().then(() => order.push(1));
      void Promise.resolve().then(() => order.push(2));
      await scheduler.nextMicrotask();
      expect(order).toEqual([1, 2]);
    });

    it("should resolve immediately without waiting for macrotasks", async () => {
      const order: number[] = [];
      setTimeout(() => order.push(1), 0);
      await scheduler.nextMicrotask();
      expect(order).toEqual([]);
    });
  });

  describe("nextTask", () => {
    it("should return a promise", async () => {
      const result = scheduler.nextTask();
      expect(result).toBeInstanceOf(Promise);
      await expect(result).resolves.toBeUndefined();
    });

    it("should flush macrotasks from setTimeout", async () => {
      const order: number[] = [];
      setTimeout(() => order.push(1), 0);
      setTimeout(() => order.push(2), 0);
      await scheduler.nextTask();
      expect(order).toEqual([1, 2]);
    });

    it("should execute after microtasks", async () => {
      const order: string[] = [];
      void Promise.resolve().then(() => order.push("micro"));
      setTimeout(() => order.push("macro"), 0);
      await scheduler.nextTask();
      expect(order).toEqual(["micro", "macro"]);
    });

    it("should handle nested flushes", async () => {
      const order: number[] = [];
      setTimeout(() => order.push(1), 0);
      await scheduler.nextTask();
      setTimeout(() => order.push(2), 0);
      await scheduler.nextTask();
      expect(order).toEqual([1, 2]);
    });
  });

  describe("flush order", () => {
    it("should flush microtasks before macrotasks", async () => {
      const order: string[] = [];
      setTimeout(() => order.push("macro"), 0);
      void Promise.resolve().then(() => order.push("micro"));
      await scheduler.nextMicrotask();
      expect(order).toEqual(["micro"]);
      await scheduler.nextTask();
      expect(order).toEqual(["micro", "macro"]);
    });
  });

  describe("nextFrame", () => {
    afterEach(() => {
      vi.unstubAllGlobals();
    });

    it("should resolve only once the next animation frame runs", async () => {
      const frames: FrameRequestCallback[] = [];
      vi.stubGlobal("requestAnimationFrame", (cb: FrameRequestCallback) =>
        frames.push(cb),
      );
      let resolved = false;
      const frame = scheduler.nextFrame().then(() => (resolved = true));
      await scheduler.nextTask();
      expect(resolved).toBe(false);
      expect(frames).toHaveLength(1);
      frames[0](0);
      await frame;
      expect(resolved).toBe(true);
    });
  });
});
