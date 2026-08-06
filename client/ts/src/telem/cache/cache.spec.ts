// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, MultiSeries, Series } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { UnexpectedError } from "@/errors";
import { Cache } from "@/telem/cache/cache";

describe("Cache", () => {
  describe("get", () => {
    it("should create an entry on first access", () => {
      const cache = new Cache();
      const entry = cache.get(1);
      expect(entry).toBeDefined();
      expect(entry.leadingBuffer).toBeNull();
      cache.close();
    });
    it("should return the same entry on repeated access", () => {
      const cache = new Cache();
      const entry = cache.get(1);
      expect(cache.get(1)).toBe(entry);
      cache.close();
    });
    it("should keep entries for distinct keys independent", () => {
      const cache = new Cache();
      const one = cache.get(1);
      const two = cache.get(2);
      one.writeDynamic(
        new MultiSeries([
          new Series({ data: new Float32Array([1]), dataType: DataType.FLOAT32 }),
        ]),
      );
      expect(one.leadingBuffer).not.toBeNull();
      expect(two.leadingBuffer).toBeNull();
      cache.close();
    });
  });
  describe("close", () => {
    it("should throw on access after close", () => {
      const cache = new Cache();
      cache.get(1);
      cache.close();
      expect(() => cache.get(1)).toThrow(UnexpectedError);
    });
    it("should close its entries", () => {
      const cache = new Cache();
      const entry = cache.get(1);
      cache.close();
      expect(() => entry.read(new MultiSeries([]).timeRange)).toThrow(UnexpectedError);
    });
  });
});
