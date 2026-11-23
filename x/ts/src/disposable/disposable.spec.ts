// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { disposable } from "@/disposable";

class SyncResource {
  declare [Symbol.dispose]: () => void;
  declare [Symbol.asyncDispose]: () => Promise<void>;
  static {
    disposable.implement(this);
  }

  closes = 0;

  close(): void {
    this.closes++;
  }
}

class AsyncResource {
  declare [Symbol.asyncDispose]: () => Promise<void>;
  static {
    disposable.implementAsync(this);
  }

  closes = 0;

  async close(): Promise<void> {
    await Promise.resolve();
    this.closes++;
  }
}

describe("disposable", () => {
  describe("implement", () => {
    it("should close the resource at the end of a using block", () => {
      const r = new SyncResource();
      {
        using _ = r;
      }
      expect(r.closes).toBe(1);
    });

    it("should close the resource at the end of an await using block", async () => {
      const r = new SyncResource();
      {
        await using _ = r;
      }
      expect(r.closes).toBe(1);
    });

    it("should not make the dispose methods enumerable", () => {
      expect(Object.keys(new SyncResource())).toEqual(["closes"]);
    });
  });

  describe("implementAsync", () => {
    it("should await close at the end of an await using block", async () => {
      const r = new AsyncResource();
      {
        await using _ = r;
      }
      expect(r.closes).toBe(1);
    });

    it("should not give the instances a synchronous dispose method", () => {
      expect(Symbol.dispose in new AsyncResource()).toBe(false);
    });

    it("should propagate an error from close", async () => {
      class Failing {
        declare [Symbol.asyncDispose]: () => Promise<void>;
        static {
          disposable.implementAsync(this);
        }

        async close(): Promise<void> {
          throw new Error("close failure");
        }
      }
      await expect(async () => {
        await using _ = new Failing();
      }).rejects.toThrow("close failure");
    });
  });
});
