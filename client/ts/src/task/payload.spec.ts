// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { task } from "@/task";

describe("newKey", () => {
  describe("basic key generation", () => {
    // Verifies that newKey correctly combines rack and task keys via bit shifting.
    // The rack key occupies the upper 32 bits, task key the lower 32.
    const cases = [
      { rackKey: 1, taskKey: 1, expected: ((1n << 32n) + 1n).toString() },
      { rackKey: 1, taskKey: 0, expected: (1n << 32n).toString() },
      { rackKey: 0, taskKey: 1, expected: "1" },
      { rackKey: 0, taskKey: 0, expected: "0" },
      { rackKey: 255, taskKey: 65535, expected: ((255n << 32n) + 65535n).toString() },
    ];
    cases.forEach(({ rackKey: r, taskKey: t, expected }) => {
      it(`should combine rackKey=${r} and taskKey=${t}`, () => {
        expect(task.newKey(r, t)).toBe(expected);
      });
    });
  });

  describe("default taskKey", () => {
    // Verifies that omitting taskKey defaults to 0, producing rackKey << 32.
    const rackKeys = [0, 1, 42, 1000];
    rackKeys.forEach((r) => {
      it(`should default taskKey to 0 for rackKey=${r}`, () => {
        expect(task.newKey(r)).toBe(task.newKey(r, 0));
      });
    });
  });

  describe("roundtrip with rackKey", () => {
    // Verifies that rackKey correctly extracts the rack portion from a combined key.
    // This ensures newKey and rackKey are inverse operations for the rack component.
    const cases = [
      { rack: 1, task: 1 },
      { rack: 1, task: 0 },
      { rack: 0, task: 1 },
      { rack: 100, task: 50000 },
      { rack: 2147483647, task: 2147483647 }, // max 32-bit signed integers
    ];
    cases.forEach(({ rack, task: tsk }) => {
      it(`should extract rack=${rack} from key generated with task=${tsk}`, () => {
        const key = task.newKey(rack, tsk);
        expect(task.rackKey(key)).toBe(rack);
      });
    });
  });

  describe("edge cases", () => {
    // Verifies behavior at boundary values to ensure no overflow or precision issues.
    it("should handle max 32-bit unsigned rack key", () => {
      const maxUint32 = 0xffffffff;
      const key = task.newKey(maxUint32, 0);
      expect(task.rackKey(key)).toBe(maxUint32);
    });

    it("should handle max 32-bit unsigned task key", () => {
      const maxUint32 = 0xffffffff;
      const key = task.newKey(1, maxUint32);
      expect(key).toBe(((1n << 32n) + BigInt(maxUint32)).toString());
    });

    it("should handle both max 32-bit values", () => {
      const maxUint32 = 0xffffffff;
      const key = task.newKey(maxUint32, maxUint32);
      const expected = ((BigInt(maxUint32) << 32n) + BigInt(maxUint32)).toString();
      expect(key).toBe(expected);
      expect(task.rackKey(key)).toBe(maxUint32);
    });
  });
});

describe("rackKey", () => {
  // Verifies rackKey extracts upper 32 bits from various key formats.
  // Keys can be string, bigint, or number per keyZ schema.
  describe("key format handling", () => {
    const rack = 42;
    const tsk = 100;
    const expectedKey = task.newKey(rack, tsk);

    it("should extract rack from string key", () => {
      expect(task.rackKey(expectedKey)).toBe(rack);
    });

    it("should extract rack from numeric string", () => {
      const numericKey = ((42n << 32n) + 100n).toString();
      expect(task.rackKey(numericKey)).toBe(42);
    });
  });

  describe("zero handling", () => {
    // Verifies correct behavior when rack or task portions are zero.
    it("should return 0 for key with zero rack", () => {
      expect(task.rackKey("100")).toBe(0);
    });

    it("should return rack when task portion is zero", () => {
      const key = task.newKey(5, 0);
      expect(task.rackKey(key)).toBe(5);
    });
  });
});
