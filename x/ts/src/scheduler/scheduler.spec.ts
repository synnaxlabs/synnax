// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { scheduler } from "@/scheduler";

describe("scheduler", () => {
  describe("flushMicrotasks", () => {
    it("should return a resolved promise", async () => {
      const result = scheduler.flushMicrotasks();
      expect(result).toBeInstanceOf(Promise);
      await expect(result).resolves.toBeUndefined();
    });

    it("should flush microtasks before continuing", async () => {
      const order: number[] = [];
      Promise.resolve().then(() => order.push(1));
      Promise.resolve().then(() => order.push(2));
      await scheduler.flushMicrotasks();
      expect(order).toEqual([1, 2]);
    });

    it("should resolve immediately without waiting for macrotasks", async () => {
      const order: number[] = [];
      setTimeout(() => order.push(1), 0);
      await scheduler.flushMicrotasks();
      expect(order).toEqual([]);
    });
  });

  describe("flushTaskQueue", () => {
    it("should return a promise", async () => {
      const result = scheduler.flushTaskQueue();
      expect(result).toBeInstanceOf(Promise);
      await expect(result).resolves.toBeUndefined();
    });

    it("should flush macrotasks from setTimeout", async () => {
      const order: number[] = [];
      setTimeout(() => order.push(1), 0);
      setTimeout(() => order.push(2), 0);
      await scheduler.flushTaskQueue();
      expect(order).toEqual([1, 2]);
    });

    it("should execute after microtasks", async () => {
      const order: string[] = [];
      Promise.resolve().then(() => order.push("micro"));
      setTimeout(() => order.push("macro"), 0);
      await scheduler.flushTaskQueue();
      expect(order).toEqual(["micro", "macro"]);
    });

    it("should handle nested flushes", async () => {
      const order: number[] = [];
      setTimeout(() => order.push(1), 0);
      await scheduler.flushTaskQueue();
      setTimeout(() => order.push(2), 0);
      await scheduler.flushTaskQueue();
      expect(order).toEqual([1, 2]);
    });
  });

  describe("flush order", () => {
    it("should flush microtasks before macrotasks", async () => {
      const order: string[] = [];
      setTimeout(() => order.push("macro"), 0);
      Promise.resolve().then(() => order.push("micro"));
      await scheduler.flushMicrotasks();
      expect(order).toEqual(["micro"]);
      await scheduler.flushTaskQueue();
      expect(order).toEqual(["micro", "macro"]);
    });
  });
});
