// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it, vi } from "vitest";

import { testutil } from "@/testutil";

describe("testutil", () => {
  describe("toString", () => {
    it("should stringify regular values normally", () => {
      expect(testutil.toString("hello")).toBe('"hello"');
      expect(testutil.toString(123)).toBe("123");
      expect(testutil.toString(true)).toBe("true");
      expect(testutil.toString(null)).toBe("null");
      expect(testutil.toString(undefined)).toBeUndefined();
    });

    it("should handle arrays", () => {
      expect(testutil.toString([1, 2, 3])).toBe("[1,2,3]");
      expect(testutil.toString(["a", "b", "c"])).toBe('["a","b","c"]');
      expect(testutil.toString([])).toBe("[]");
    });

    it("should handle objects", () => {
      expect(testutil.toString({ a: 1, b: 2 })).toBe('{"a":1,"b":2}');
      expect(testutil.toString({ nested: { value: 42 } })).toBe(
        '{"nested":{"value":42}}',
      );
      expect(testutil.toString({})).toBe("{}");
    });

    it("should convert bigint to string", () => {
      const bigIntValue = BigInt(123456789012345678901234567890n);
      expect(testutil.toString(bigIntValue)).toBe('"123456789012345678901234567890"');
    });

    it("should handle objects with bigint values", () => {
      const obj = {
        regular: 123,
        big: BigInt(999999999999999999999n),
        nested: {
          value: BigInt(111111111111111111111n),
        },
      };
      expect(testutil.toString(obj)).toBe(
        '{"regular":123,"big":"999999999999999999999","nested":{"value":"111111111111111111111"}}',
      );
    });

    it("should handle arrays with bigint values", () => {
      const arr = [BigInt(1n), BigInt(2n), 3, BigInt(4n)];
      expect(testutil.toString(arr)).toBe('["1","2",3,"4"]');
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
      expect(testutil.toString(complex)).toBe(
        '{"id":"123","data":[1,"456",{"value":"789"}],"metadata":{"count":42,"total":"999999"}}',
      );
    });
  });

  describe("testutil.expectAlways", () => {
    it("should call function multiple times over duration", async () => {
      const fn = vi.fn();
      await testutil.expectAlways(fn, 100, 20);
      expect(fn.mock.calls.length).toBeGreaterThanOrEqual(3);
      expect(fn.mock.calls.length).toBeLessThanOrEqual(7);
    });

    it("should handle async functions", async () => {
      let counter = 0;
      const asyncFn = vi.fn(async () => {
        counter++;
        await new Promise((resolve) => setTimeout(resolve, 5));
      });

      await testutil.expectAlways(asyncFn, 60, 20);
      expect(asyncFn).toHaveBeenCalled();
      expect(counter).toBeGreaterThan(0);
    });

    it("should propagate errors from the function", async () => {
      const errorFn = vi.fn(() => {
        throw new Error("Test error");
      });

      await expect(testutil.expectAlways(errorFn, 50, 20)).rejects.toThrow(
        "Test error",
      );
      expect(errorFn).toHaveBeenCalledTimes(1);
    });

    it("should propagate errors from async functions", async () => {
      const asyncErrorFn = vi.fn(async () => {
        throw new Error("Async test error");
      });

      await expect(testutil.expectAlways(asyncErrorFn, 50, 20)).rejects.toThrow(
        "Async test error",
      );
      expect(asyncErrorFn).toHaveBeenCalledTimes(1);
    });

    it("should use default values when not provided", async () => {
      const fn = vi.fn();
      await testutil.expectAlways(fn);
      expect(fn.mock.calls.length).toBeGreaterThanOrEqual(7);
      expect(fn.mock.calls.length).toBeLessThanOrEqual(12);
    });

    it("should respect custom interval", async () => {
      const fn = vi.fn();
      const start = Date.now();
      await testutil.expectAlways(fn, 100, 30);
      const elapsed = Date.now() - start;
      expect(elapsed).toBeGreaterThanOrEqual(90);
      expect(elapsed).toBeLessThan(150);
      expect(fn.mock.calls.length).toBeGreaterThanOrEqual(3);
      expect(fn.mock.calls.length).toBeLessThanOrEqual(5);
    });

    it("should handle functions that pass after initial failures", async () => {
      let callCount = 0;
      const fn = vi.fn(() => {
        callCount++;
        if (callCount < 3) throw new Error("Not ready yet");
      });

      await expect(testutil.expectAlways(fn, 80, 20)).rejects.toThrow("Not ready yet");
      expect(fn).toHaveBeenCalledTimes(1);
    });

    it("should work with expectations inside the function", async () => {
      let value = 0;
      const incrementer = setInterval(() => value++, 10);

      try {
        await testutil.expectAlways(
          () => {
            expect(value).toBeGreaterThanOrEqual(0);
          },
          50,
          10,
        );
      } finally {
        clearInterval(incrementer);
      }
    });
  });
});
