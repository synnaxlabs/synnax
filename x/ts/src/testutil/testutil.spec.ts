// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { expectAlways, toString } from "@/testutil";

describe("testutil", () => {
  describe("toString", () => {
    it("should stringify regular values normally", () => {
      expect(toString("hello")).toBe('"hello"');
      expect(toString(123)).toBe("123");
      expect(toString(true)).toBe("true");
      expect(toString(null)).toBe("null");
      expect(toString(undefined)).toBeUndefined();
    });

    it("should handle arrays", () => {
      expect(toString([1, 2, 3])).toBe("[1,2,3]");
      expect(toString(["a", "b", "c"])).toBe('["a","b","c"]');
      expect(toString([])).toBe("[]");
    });

    it("should handle objects", () => {
      expect(toString({ a: 1, b: 2 })).toBe('{"a":1,"b":2}');
      expect(toString({ nested: { value: 42 } })).toBe('{"nested":{"value":42}}');
      expect(toString({})).toBe("{}");
    });

    it("should convert bigint to string", () => {
      const bigIntValue = BigInt(123456789012345678901234567890n);
      expect(toString(bigIntValue)).toBe('"123456789012345678901234567890"');
    });

    it("should handle objects with bigint values", () => {
      const obj = {
        regular: 123,
        big: BigInt(999999999999999999999n),
        nested: {
          value: BigInt(111111111111111111111n),
        },
      };
      expect(toString(obj)).toBe(
        '{"regular":123,"big":"999999999999999999999","nested":{"value":"111111111111111111111"}}',
      );
    });

    it("should handle arrays with bigint values", () => {
      const arr = [BigInt(1n), BigInt(2n), 3, BigInt(4n)];
      expect(toString(arr)).toBe('["1","2",3,"4"]');
    });

    it("should handle mixed complex structures", () => {
      const complex = {
        id: BigInt(123n),
        data: [1, BigInt(456n), { value: BigInt(789n) }],
        metadata: {
          count: 42,
          total: BigInt(999999n),
        },
      };
      expect(toString(complex)).toBe(
        '{"id":"123","data":[1,"456",{"value":"789"}],"metadata":{"count":42,"total":"999999"}}',
      );
    });
  });

  describe("expectAlways", () => {
    beforeEach(() => {
      vi.useFakeTimers();
    });

    afterEach(() => {
      vi.useRealTimers();
    });

    it("should call the function once per interval over the duration", async () => {
      const fn = vi.fn();
      const done = expectAlways(fn, 100, 20);
      await vi.runAllTimersAsync();
      await done;
      expect(fn).toHaveBeenCalledTimes(5);
    });

    it("should wait for async functions before sleeping", async () => {
      const start = Date.now();
      const calls: number[] = [];
      const asyncFn = async () => {
        calls.push(Date.now() - start);
        await new Promise((resolve) => setTimeout(resolve, 30));
      };
      const done = expectAlways(asyncFn, 100, 20);
      await vi.runAllTimersAsync();
      await done;
      expect(calls).toEqual([0, 50]);
    });

    it("should propagate errors from the function", async () => {
      const errorFn = vi.fn(() => {
        throw new Error("Test error");
      });
      await expect(expectAlways(errorFn, 50, 20)).rejects.toThrow("Test error");
      expect(errorFn).toHaveBeenCalledTimes(1);
    });

    it("should propagate errors from async functions", async () => {
      const asyncErrorFn = vi.fn(async () => {
        throw new Error("Async test error");
      });
      await expect(expectAlways(asyncErrorFn, 50, 20)).rejects.toThrow(
        "Async test error",
      );
      expect(asyncErrorFn).toHaveBeenCalledTimes(1);
    });

    it("should use default values when not provided", async () => {
      const fn = vi.fn();
      const done = expectAlways(fn);
      await vi.runAllTimersAsync();
      await done;
      expect(fn).toHaveBeenCalledTimes(10);
    });

    it("should sleep for the interval between calls", async () => {
      const fn = vi.fn();
      const start = Date.now();
      const done = expectAlways(fn, 100, 30);
      await vi.runAllTimersAsync();
      await done;
      expect(Date.now() - start).toBe(120);
      expect(fn).toHaveBeenCalledTimes(4);
    });

    it("should reject on the first failure instead of retrying", async () => {
      let callCount = 0;
      const fn = vi.fn(() => {
        callCount++;
        if (callCount < 3) throw new Error("Not ready yet");
      });
      await expect(expectAlways(fn, 80, 20)).rejects.toThrow("Not ready yet");
      expect(fn).toHaveBeenCalledTimes(1);
    });

    it("should fail when an expectation inside the function stops holding", async () => {
      let value = 0;
      const incrementer = setInterval(() => value++, 10);
      try {
        const rejects = expect(
          expectAlways(() => expect(value).toBeLessThan(3), 100, 20),
        ).rejects.toThrow();
        await vi.advanceTimersByTimeAsync(100);
        await rejects;
      } finally {
        clearInterval(incrementer);
      }
    });
  });
});
