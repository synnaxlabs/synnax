// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, MultiSeries, Series, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { channel } from "@/channel";
import { UnexpectedError } from "@/errors";
import { Cache } from "@/framer/cache/cache";
import { IDENTITY_TRANSFORM, type Transform } from "@/framer/cache/transform";
import { glTransform } from "@/testutil/telem";

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

  const payload = (
    p: Partial<channel.Payload> & Pick<channel.Payload, "key">,
  ): channel.Payload =>
    channel.payloadZ.parse({ name: `ch-${p.key}`, dataType: DataType.FLOAT32, ...p });
  const INDEX = payload({
    key: 1,
    index: 1,
    dataType: DataType.TIMESTAMP,
    isIndex: true,
  });
  const DATA = payload({ key: 2, index: 1 });
  const VIRTUAL = payload({ key: 3, virtual: true });
  const CALCULATED = payload({ key: 4, index: 1, virtual: true, expression: "a" });
  const OTHER = payload({ key: 5, index: 1 });

  describe("resolve", () => {
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

    it("should resolve an index found through its data channel", async () => {
      const { cache, calls } = createCache([INDEX, DATA]);
      await cache.resolve([2]);
      expect(cache.indexOf(1)).toBe(1);
      expect(cache.get(1).index).toBe(cache.get(1));
      await cache.resolve([1]);
      expect(calls).toEqual([[2]]);
      cache.close();
    });

    it("should give a virtual channel no index", async () => {
      const { cache } = createCache([VIRTUAL]);
      await cache.resolve([3]);
      expect(cache.indexOf(3)).toBeNull();
      expect(cache.get(3).index).toBeNull();
      cache.close();
    });

    it("should give only a channel with no index write stamps", async () => {
      const { cache } = createCache([INDEX, DATA, VIRTUAL]);
      await cache.resolve([1, 2, 3]);
      expect(cache.get(1).stamps).toBeNull();
      expect(cache.get(2).stamps).toBeNull();
      expect(cache.get(3).stamps).not.toBeNull();
      cache.close();
    });

    it("should time a virtual sample through the write that carried it", async () => {
      const { cache } = createCache([VIRTUAL]);
      await cache.resolve([3]);
      const t0 = TimeStamp.seconds(1_700_000_000);
      cache.get(3).writeDynamic(
        new MultiSeries([
          new Series({
            data: new Float32Array([1, 2]),
            dataType: DataType.FLOAT32,
            timeRange: t0.range(t0.add(TimeSpan.NANOSECOND)),
            alignment: 10n,
          }),
        ]),
      );
      expect(cache.timeAt(3, 11n)).toEqual(t0);
      expect(cache.lastWrite(3)).toEqual(t0);
      cache.close();
    });

    it("should keep as many write stamps as configured", async () => {
      const cache = new Cache({
        retrieveChannels: async () => [VIRTUAL],
        writeStamps: 1,
      });
      await cache.resolve([3]);
      const t0 = TimeStamp.seconds(1_700_000_000);
      const t1 = t0.add(TimeSpan.seconds(1));
      const write = (alignment: bigint, stamp: TimeStamp): MultiSeries =>
        new MultiSeries([
          new Series({
            data: new Float32Array([1]),
            dataType: DataType.FLOAT32,
            timeRange: stamp.range(stamp.add(TimeSpan.NANOSECOND)),
            alignment,
          }),
        ]);
      cache.get(3).writeDynamic(write(10n, t0));
      cache.get(3).writeDynamic(write(11n, t1));
      expect(cache.timeAt(3, 10n)).toBeNull();
      expect(cache.timeAt(3, 11n)).toEqual(t1);
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

    it("should skip an index an earlier key in the same pass recorded", async () => {
      const { cache, calls } = createCache([INDEX, DATA]);
      await cache.resolve([2, 1]);
      expect(calls).toEqual([[2, 1]]);
      expect(cache.indexOf(1)).toBe(1);
      expect(cache.indexOf(2)).toBe(1);
      expect(cache.get(2).index).toBe(cache.get(1));
      cache.close();
    });

    it("should settle overlapping lookups of one key consistently", async () => {
      const { cache } = createCache([INDEX, DATA]);
      await Promise.all([cache.resolve([2]), cache.resolve([2])]);
      expect(cache.indexOf(2)).toBe(1);
      expect(cache.indexOf(1)).toBe(1);
      expect(cache.get(2).index).toBe(cache.get(1));
      expect(cache.get(1).index).toBe(cache.get(1));
      cache.close();
    });

    it("should point sibling data channels at one index entry", async () => {
      const { cache } = createCache([INDEX, DATA, OTHER]);
      await cache.resolve([2, 5]);
      expect(cache.get(2).index).toBe(cache.get(1));
      expect(cache.get(5).index).toBe(cache.get(1));
      cache.close();
    });

    it("should trust a payload's index without fetching the index channel", async () => {
      const { cache, calls } = createCache([DATA]);
      await cache.resolve([2]);
      expect(cache.indexOf(1)).toBe(1);
      expect(cache.get(1).index).toBe(cache.get(1));
      await cache.resolve([1]);
      expect(calls).toEqual([[2]]);
      cache.close();
    });

    it("should record nothing when closed during the lookup", async () => {
      let release: (() => void) | undefined;
      const cache = new Cache({
        retrieveChannels: async () => {
          await new Promise<void>((resolve) => {
            release = resolve;
          });
          return [INDEX, DATA];
        },
      });
      const pending = cache.resolve([2]);
      cache.close();
      release?.();
      await expect(pending).resolves.toBeUndefined();
      expect(cache.indexOf(2)).toBeNull();
      expect(cache.indexOf(1)).toBeNull();
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
      expect(cache.timeAt(2, 11n)).toEqual(new TimeStamp(t0 + 1n));
      expect(cache.timeAt(1, 11n)).toEqual(new TimeStamp(t0 + 1n));
      expect(cache.lastWrite(1)).toEqual(new TimeStamp(t0 + 1n));
      expect(cache.lastWrite(2)).toBeNull();
      cache.close();
    });

    it("should throw from the accessors after close", () => {
      const { cache } = createCache([INDEX]);
      cache.close();
      expect(() => cache.timeAt(1, 0n)).toThrow(UnexpectedError);
      expect(() => cache.lastWrite(1)).toThrow(UnexpectedError);
    });

    it("should throw after close", async () => {
      const { cache } = createCache([INDEX]);
      cache.close();
      await expect(cache.resolve([1])).rejects.toThrow(UnexpectedError);
    });
  });

  describe("gc", () => {
    const T0 = TimeStamp.seconds(1_700_000_000).valueOf();
    const T1 = new TimeStamp(T0 + 1n);
    const TR = new TimeStamp(T0).range(new TimeStamp(T0 + 2n));
    const LEADING = (BigInt(0xffffffff) - 1_000_000n) << 32n;
    const stamps = (alignment = 10n): MultiSeries =>
      new MultiSeries([
        new Series({
          data: new BigInt64Array([T0, T0 + 1n]),
          dataType: DataType.TIMESTAMP,
          timeRange: TR,
          alignment,
        }),
      ]);
    const values = (alignment = 10n): MultiSeries =>
      new MultiSeries([
        new Series({
          data: new Float32Array([1, 2]),
          dataType: DataType.FLOAT32,
          timeRange: TR,
          alignment,
        }),
      ]);
    // Collects every 10 ms with no grace period, so only retention keeps an entry.
    const createCache = (transform: Transform = glTransform): Cache =>
      new Cache({
        retrieveChannels: async (keys) =>
          [INDEX, DATA, VIRTUAL, OTHER].filter((ch) =>
            (keys as channel.Key[]).includes(ch.key),
          ),
        gcInterval: TimeSpan.milliseconds(10),
        staleEntryThreshold: TimeSpan.ZERO,
        transform,
      });
    const acquire = (cache: Cache, key: channel.Key): (() => void) => {
      const held = cache.get(key).read(TR).series.series;
      held.forEach((s) => s.acquire());
      return () => held.forEach((s) => s.release());
    };
    const hold = (cache: Cache, key: channel.Key): (() => void) => {
      cache.get(key).writeStatic(values());
      return acquire(cache, key);
    };
    const tick = (): void => {
      vi.advanceTimersByTime(10);
    };

    beforeEach(() => vi.useFakeTimers());
    afterEach(() => vi.useRealTimers());

    it("should keep the stamps of held samples past the stale threshold", async () => {
      const cache = createCache();
      await cache.resolve([2, 5]);
      cache.get(1).writeStatic(stamps());
      const releaseData = hold(cache, 2);
      const releaseOther = hold(cache, 5);
      tick();
      expect(cache.timeAt(2, 11n)).toEqual(T1);
      releaseData();
      tick();
      expect(cache.timeAt(2, 11n)).toEqual(T1);
      releaseOther();
      tick();
      expect(cache.timeAt(2, 11n)).toBeNull();
      cache.close();
    });

    it("should keep the stamps of held streamed samples through a fetch", async () => {
      const cache = createCache();
      await cache.resolve([2]);
      cache.get(1).writeDynamic(stamps(LEADING));
      cache.get(1).flushDynamic();
      cache.get(2).writeDynamic(values(LEADING));
      cache.get(2).flushDynamic();
      const release = acquire(cache, 2);
      cache.get(1).writeStatic(stamps());
      cache.get(2).writeStatic(values());
      tick();
      expect(cache.timeAt(2, LEADING + 1n)).toEqual(T1);
      release();
      tick();
      expect(cache.timeAt(2, LEADING + 1n)).toBeNull();
      cache.close();
    });

    it("should purge the narrowed index copy while its stamps stay", async () => {
      const cache = createCache();
      await cache.resolve([2]);
      cache.get(1).writeStatic(stamps());
      hold(cache, 2);
      tick();
      expect(cache.get(1).read(TR).series.series).toHaveLength(0);
      expect(cache.timeAt(2, 11n)).toEqual(T1);
      cache.close();
    });

    it("should keep stamps held through the index channel itself", async () => {
      const cache = createCache();
      await cache.resolve([2]);
      cache.get(1).writeStatic(stamps());
      const release = acquire(cache, 1);
      tick();
      expect(cache.timeAt(2, 11n)).toEqual(T1);
      release();
      tick();
      expect(cache.timeAt(2, 11n)).toBeNull();
      cache.close();
    });

    it("should keep stamps under a held leading buffer", async () => {
      const cache = createCache();
      await cache.resolve([2]);
      cache.get(1).writeDynamic(stamps());
      cache.get(1).flushDynamic();
      cache.get(2).writeDynamic(values());
      const leading = cache.get(2).leadingBuffer;
      expect(leading).not.toBeNull();
      leading?.acquire();
      tick();
      expect(cache.timeAt(2, 11n)).toEqual(T1);
      leading?.release();
      tick();
      expect(cache.timeAt(2, 11n)).toBeNull();
      cache.close();
    });

    it("should ignore held samples of channels without an index", async () => {
      const cache = createCache();
      await cache.resolve([2, 3]);
      cache.get(1).writeStatic(stamps());
      hold(cache, 3);
      hold(cache, 9);
      tick();
      expect(cache.timeAt(2, 11n)).toBeNull();
      cache.close();
    });

    it("should keep stamps in the main copy when nothing narrows them", async () => {
      const cache = createCache(IDENTITY_TRANSFORM);
      await cache.resolve([2]);
      cache.get(1).writeStatic(stamps());
      const release = hold(cache, 2);
      tick();
      expect(cache.get(1).read(TR).series.series).toHaveLength(1);
      expect(cache.timeAt(2, 11n)).toEqual(T1);
      release();
      tick();
      expect(cache.timeAt(2, 11n)).toBeNull();
      cache.close();
    });
  });
});
