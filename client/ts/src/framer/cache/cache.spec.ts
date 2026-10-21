// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, MultiSeries, Series, TimeStamp } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { channel } from "@/channel";
import { UnexpectedError } from "@/errors";
import { Cache } from "@/framer/cache/cache";

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

  describe("resolve", () => {
    const payload = (
      p: Partial<channel.Payload> & Pick<channel.Payload, "key">,
    ): channel.Payload =>
      channel.payloadZ.parse({ name: `ch-${p.key}`, dataType: DataType.FLOAT32, ...p });
    const INDEX = payload({ key: 1, dataType: DataType.TIMESTAMP, isIndex: true });
    const DATA = payload({ key: 2, index: 1 });
    const VIRTUAL = payload({ key: 3, virtual: true });
    const CALCULATED = payload({ key: 4, index: 1, virtual: true, expression: "a" });

    const createCache = (channels: channel.Payload[]) => {
      const calls: channel.Key[][] = [];
      const cache = new Cache({
        retrieveChannels: async (keys) => {
          calls.push(keys as channel.Key[]);
          return channels.filter((ch) => (keys as channel.Key[]).includes(ch.key));
        },
      });
      return { cache, calls };
    };

    it("should point an index channel at itself", async () => {
      const { cache } = createCache([INDEX]);
      await cache.resolve([1]);
      expect(cache.indexOf(1)).toBe(1);
      expect(cache.get(1).index).toBe(cache.get(1));
      cache.close();
    });

    it("should point a data channel at its index", async () => {
      const { cache } = createCache([INDEX, DATA]);
      await cache.resolve([2]);
      expect(cache.indexOf(2)).toBe(1);
      expect(cache.get(2).index).toBe(cache.get(1));
      cache.close();
    });

    it("should give a virtual channel no index", async () => {
      const { cache } = createCache([VIRTUAL]);
      await cache.resolve([3]);
      expect(cache.indexOf(3)).toBeNull();
      expect(cache.get(3).index).toBeNull();
      cache.close();
    });

    it("should give a calculated channel its index", async () => {
      const { cache } = createCache([INDEX, CALCULATED]);
      await cache.resolve([4]);
      expect(cache.indexOf(4)).toBe(1);
      cache.close();
    });

    it("should be null before a key resolves", () => {
      const { cache } = createCache([INDEX]);
      expect(cache.indexOf(1)).toBeNull();
      cache.close();
    });

    it("should resolve a missing key to no index and not ask again", async () => {
      const { cache, calls } = createCache([]);
      await cache.resolve([5]);
      await cache.resolve([5]);
      expect(cache.indexOf(5)).toBeNull();
      expect(calls).toEqual([[5]]);
      cache.close();
    });

    it("should ask only for unresolved keys", async () => {
      const { cache, calls } = createCache([INDEX, DATA]);
      await cache.resolve([1]);
      await cache.resolve([1, 2]);
      await cache.resolve([1, 2]);
      expect(calls).toEqual([[1], [2]]);
      cache.close();
    });

    it("should time a data sample through the resolved index", async () => {
      const { cache } = createCache([INDEX, DATA]);
      await cache.resolve([2]);
      const t0 = TimeStamp.seconds(1_700_000_000).valueOf();
      cache.get(1).writeDynamic(
        new MultiSeries([
          new Series({
            data: new BigInt64Array([t0, t0 + 1n]),
            dataType: DataType.TIMESTAMP,
            timeRange: new TimeStamp(t0).range(new TimeStamp(t0 + 2n)),
            alignment: 10n,
          }),
        ]),
      );
      expect(cache.get(2).timeAt(11n)).toEqual(new TimeStamp(t0 + 1n));
      cache.close();
    });

    it("should throw after close", async () => {
      const { cache } = createCache([INDEX]);
      cache.close();
      await expect(cache.resolve([1])).rejects.toThrow(UnexpectedError);
    });
  });
});
