// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it, vi } from "vitest";

import { destructor } from "@/destructor";

describe("destructor", () => {
  describe("unwind", () => {
    it("should run destructors in reverse of the given order", () => {
      const order: number[] = [];
      destructor.unwind(
        () => order.push(1),
        () => order.push(2),
        () => order.push(3),
      );
      expect(order).toEqual([3, 2, 1]);
    });

    it("should skip nullish entries", () => {
      const undo = vi.fn();
      expect(() => destructor.unwind(undefined, undo, null)).not.toThrow();
      expect(undo).toHaveBeenCalledTimes(1);
    });

    it("should run the remaining destructors when one throws", () => {
      const consoleError = vi.spyOn(console, "error").mockImplementation(() => {});
      try {
        const first = vi.fn();
        destructor.unwind(first, () => {
          throw new Error("undo failure");
        });
        expect(first).toHaveBeenCalledTimes(1);
        expect(consoleError).toHaveBeenCalled();
      } finally {
        consoleError.mockRestore();
      }
    });

    it("should do nothing when given no destructors", () => {
      expect(() => destructor.unwind()).not.toThrow();
    });
  });

  describe("Chain", () => {
    describe("guard", () => {
      it("should return the call's result on success", async () => {
        const chain = new destructor.Chain();
        const result = await chain.guard(async () => 42);
        expect(result).toBe(42);
      });

      it("should not run destructors on success", async () => {
        const chain = new destructor.Chain();
        const undo = vi.fn();
        chain.add(undo);
        await chain.guard(async () => "ok");
        expect(undo).not.toHaveBeenCalled();
      });

      it("should run destructors in reverse order on failure", async () => {
        const chain = new destructor.Chain();
        const order: number[] = [];
        chain.add(() => order.push(1));
        chain.add(
          () => order.push(2),
          () => order.push(3),
        );
        await expect(
          chain.guard(async () => {
            throw new Error("network failure");
          }),
        ).rejects.toThrow("network failure");
        expect(order).toEqual([3, 2, 1]);
      });

      it("should rethrow non-Error rejections as Errors", async () => {
        const chain = new destructor.Chain();
        await expect(
          chain.guard(async () => {
            // eslint-disable-next-line @typescript-eslint/only-throw-error
            throw "string failure";
          }),
        ).rejects.toBeInstanceOf(Error);
      });

      it("should clear destructors after a failure", async () => {
        const chain = new destructor.Chain();
        const undo = vi.fn();
        chain.add(undo);
        await expect(
          chain.guard(async () => {
            throw new Error("first");
          }),
        ).rejects.toThrow("first");
        expect(undo).toHaveBeenCalledTimes(1);
        await expect(
          chain.guard(async () => {
            throw new Error("second");
          }),
        ).rejects.toThrow("second");
        expect(undo).toHaveBeenCalledTimes(1);
      });

      it("should run remaining destructors when one fails", async () => {
        const consoleError = vi.spyOn(console, "error").mockImplementation(() => {});
        try {
          const chain = new destructor.Chain();
          const undo = vi.fn();
          chain.add(undo);
          chain.add(() => {
            throw new Error("undo failure");
          });
          await expect(
            chain.guard(async () => {
              throw new Error("network failure");
            }),
          ).rejects.toThrow("network failure");
          expect(undo).toHaveBeenCalledTimes(1);
          expect(consoleError).toHaveBeenCalled();
        } finally {
          consoleError.mockRestore();
        }
      });

      it("should log instead of throwing when a destructor fails", async () => {
        const consoleError = vi.spyOn(console, "error").mockImplementation(() => {});
        try {
          const chain = new destructor.Chain();
          chain.add(() => {
            throw new Error("undo failure");
          });
          await expect(
            chain.guard(async () => {
              throw new Error("network failure");
            }),
          ).rejects.toThrow("network failure");
          expect(consoleError).toHaveBeenCalled();
        } finally {
          consoleError.mockRestore();
        }
      });
    });
  });
});
