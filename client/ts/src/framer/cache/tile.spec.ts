// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Unreachable } from "@synnaxlabs/freighter";
import {
  DataType,
  type Reduction,
  Series,
  Size,
  sleep,
  type TimeRange,
  TimeSpan,
} from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { type channel } from "@/channel";
import { framer } from "@/framer";
import { rawKey } from "@/framer/cache/block";
import { tileRange } from "@/framer/cache/grid";
import { Store, type StoreProps } from "@/framer/cache/store";
import {
  tileKey,
  TileReader,
  type TileReaderProps,
  type TileSpec,
} from "@/framer/cache/tile";
import { UnexpectedError } from "@/index";
import { ManualClock } from "@/testutil";

interface Call {
  tr: TimeRange;
  keys: channel.Key[];
  reduction?: Reduction;
}

const SAMPLES = 4;

// Returns a reader whose remote fetch records each call and answers every key with
// samples float64 samples equal to the key.
const createReader = (
  { budget, staleThreshold, now, ...props }: Partial<TileReaderProps> & StoreProps = {},
  samples: number = SAMPLES,
): { reader: TileReader; calls: Call[]; store: Store } => {
  const calls: Call[] = [];
  const store = new Store({ budget, staleThreshold, now });
  const reader = new TileReader({
    store,
    now,
    batchDebounce: TimeSpan.milliseconds(5),
    readRemote: async (tr, keys, reduction) => {
      calls.push({ tr, keys, reduction });
      return new framer.Frame(
        keys,
        keys.map(
          (key) =>
            new Series({
              data: new Float64Array(samples).fill(key),
              timeRange: tr,
            }),
        ),
      );
    },
    ...props,
  });
  return { reader, calls, store };
};

// Budget specs read large tiles, so the fixed overhead of each tile does not change how
// many fit.
const LARGE_SAMPLES = 1024;
const LARGE_BYTES = LARGE_SAMPLES * DataType.FLOAT64.density.valueOf();

const createSpec = (overrides: Partial<TileSpec> = {}): TileSpec => ({
  key: 1,
  level: 10,
  index: 3,
  pointLimit: 100,
  aggregation: "min_max",
  ...overrides,
});

describe("tile", () => {
  describe("TileReader", () => {
    it("should read the tile range with the spec's options", async () => {
      const { reader, calls } = createReader();
      const s = createSpec();
      const series = await reader.read(s);
      expect(calls).toHaveLength(1);
      expect(calls[0].tr.equals(tileRange(s))).toBe(true);
      expect(calls[0].keys).toEqual([1]);
      expect(calls[0].reduction).toEqual({
        variant: "limit",
        aggregation: "min_max",
        pointLimit: 100,
      });
      expect(Array.from(series)).toEqual([1, 1, 1, 1]);
      reader.close();
    });

    it("should reject a read when the block under the tile's key is not a tile", async () => {
      const { reader, store } = createReader();
      store.set("1/10/3/100/min_max", store.raw(1, 0));
      const read = reader.read(createSpec());
      await expect(read).rejects.toThrow(UnexpectedError);
      await expect(read).rejects.toThrow("block 1/10/3/100/min_max is not a tile");
      reader.close();
    });

    it("should serve a second read of a tile from the cache", async () => {
      const { reader, calls } = createReader();
      const first = await reader.read(createSpec());
      const second = await reader.read(createSpec());
      expect(calls).toHaveLength(1);
      expect(second).toBe(first);
      reader.close();
    });

    it("should merge reads of one tile across channels into one request", async () => {
      const { reader, calls } = createReader();
      const [a, b] = await Promise.all([
        reader.read(createSpec({ key: 1 })),
        reader.read(createSpec({ key: 2 })),
      ]);
      expect(calls).toHaveLength(1);
      expect(calls[0].keys.sort()).toEqual([1, 2]);
      expect(Array.from(a)).toEqual([1, 1, 1, 1]);
      expect(Array.from(b)).toEqual([2, 2, 2, 2]);
      reader.close();
    });

    it("should share one fetch between concurrent reads of the same tile", async () => {
      const { reader, calls } = createReader();
      const [a, b] = await Promise.all([
        reader.read(createSpec()),
        reader.read(createSpec()),
      ]);
      expect(calls).toHaveLength(1);
      expect(calls[0].keys).toEqual([1]);
      expect(b).toBe(a);
      reader.close();
    });

    describe("a cached tile read with another channel of the tile", () => {
      it("should read the tile again in one request with the other channel", async () => {
        const calls: channel.Key[][] = [];
        // The groups the Core picks follow the samples written into the tile.
        let multiple = 2n;
        const reader = new TileReader({
          store: new Store(),
          batchDebounce: TimeSpan.milliseconds(5),
          readRemote: async (tr, keys) => {
            calls.push([...keys].sort());
            return new framer.Frame(
              keys,
              keys.map(
                () =>
                  new Series({
                    data: new Float64Array(SAMPLES),
                    timeRange: tr,
                    alignmentMultiple: multiple,
                  }),
              ),
            );
          },
        });
        await reader.read(createSpec({ key: 1 }));
        multiple = 4n;
        const [x, y] = await Promise.all([
          reader.read(createSpec({ key: 1 })),
          reader.read(createSpec({ key: 2 })),
        ]);
        expect(calls).toEqual([[1], [1, 2]]);
        expect(x.series[0].alignmentMultiple).toBe(4n);
        expect(y.series[0].alignmentMultiple).toBe(4n);
        expect(await reader.read(createSpec({ key: 1 }))).toBe(x);
        reader.close();
      });

      it("should resolve at once when the other read is of another tile", async () => {
        const { reader } = createReader({ batchDebounce: TimeSpan.milliseconds(20) });
        await reader.read(createSpec({ key: 1 }));
        let resolved = false;
        const cached = reader
          .read(createSpec({ key: 1 }))
          .then(() => (resolved = true));
        const fetched = reader.read(createSpec({ key: 2, index: 4 }));
        await sleep.sleep(TimeSpan.milliseconds(5));
        expect(resolved).toBe(true);
        await Promise.all([cached, fetched]);
        reader.close();
      });

      it("should reject when the reader closes before the fetch lands", async () => {
        const { reader } = createReader();
        await reader.read(createSpec({ key: 1 }));
        const cached = expect(reader.read(createSpec({ key: 1 }))).rejects.toThrow(
          "tile reader is closed",
        );
        const fetched = expect(reader.read(createSpec({ key: 2 }))).rejects.toThrow(
          "tile reader is closed",
        );
        reader.close();
        await Promise.all([cached, fetched]);
      });
    });

    it("should fetch tiles with different bounds or options separately", async () => {
      const { reader, calls } = createReader();
      const mid = tileRange(createSpec()).start.add(TimeSpan.milliseconds(512));
      await Promise.all([
        reader.read(createSpec({ index: 3 })),
        reader.read(createSpec({ index: 4 })),
        reader.read(createSpec({ level: 11 })),
        reader.read(createSpec({ pointLimit: 50 })),
        reader.read(createSpec({ aggregation: "average" })),
        reader.read(createSpec({ end: mid })),
        reader.read(createSpec({ start: mid })),
      ]);
      expect(calls).toHaveLength(7);
      reader.close();
    });

    it("should cache tiles with different options under separate keys", async () => {
      const { reader, calls } = createReader();
      await reader.read(createSpec({ aggregation: "min_max" }));
      await reader.read(createSpec({ aggregation: "average" }));
      await reader.read(createSpec({ aggregation: "min_max" }));
      await reader.read(createSpec({ aggregation: "average" }));
      expect(calls).toHaveLength(2);
      reader.close();
    });

    it("should never cache a partial tile", async () => {
      const { reader, calls } = createReader();
      const end = tileRange(createSpec()).start.add(TimeSpan.milliseconds(100));
      await reader.read(createSpec({ end }));
      await reader.read(createSpec({ end }));
      expect(calls).toHaveLength(2);
      expect(calls[0].tr.end.equals(end)).toBe(true);
      reader.close();
    });

    it("should never cache a tile cut at its start", async () => {
      const { reader, calls } = createReader();
      const start = tileRange(createSpec()).start.add(TimeSpan.milliseconds(100));
      await reader.read(createSpec({ start }));
      await reader.read(createSpec({ start }));
      expect(calls).toHaveLength(2);
      expect(calls[0].tr.start.equals(start)).toBe(true);
      reader.close();
    });

    it("should return an empty series for a channel with no data", async () => {
      const { reader } = createReader({
        readRemote: async () => new framer.Frame([], []),
      });
      const series = await reader.read(createSpec());
      expect(series.length).toBe(0);
      reader.close();
    });

    it("should evict the least recently used tile past the budget", async () => {
      const { reader, calls } = createReader(
        { budget: Size.bytes(2.5 * LARGE_BYTES) },
        LARGE_SAMPLES,
      );
      await reader.read(createSpec({ index: 1 }));
      await reader.read(createSpec({ index: 2 }));
      await reader.read(createSpec({ index: 1 }));
      await reader.read(createSpec({ index: 3 }));
      expect(calls).toHaveLength(3);
      await reader.read(createSpec({ index: 1 }));
      expect(calls).toHaveLength(3);
      await reader.read(createSpec({ index: 2 }));
      expect(calls).toHaveLength(4);
      reader.close();
    });

    it("should keep a tile a subscription pins past the budget", async () => {
      const { reader, calls, store } = createReader(
        { budget: Size.bytes(1.5 * LARGE_BYTES) },
        LARGE_SAMPLES,
      );
      const spec = createSpec({ index: 1 });
      await reader.read(spec);
      const tile = store.get(tileKey(spec))!;
      tile.pins++;
      await reader.read(createSpec({ index: 2 }));
      await reader.read(createSpec({ index: 1 }));
      expect(calls).toHaveLength(2);
      tile.pins--;
      await reader.read(createSpec({ index: 3 }));
      await reader.read(createSpec({ index: 1 }));
      expect(calls).toHaveLength(4);
      reader.close();
    });

    it("should count a tile read again with another channel once against the budget", async () => {
      const { reader, calls } = createReader(
        { budget: Size.bytes(3.5 * LARGE_BYTES) },
        LARGE_SAMPLES,
      );
      await reader.read(createSpec({ key: 1, index: 1 }));
      await Promise.all([
        reader.read(createSpec({ key: 1, index: 1 })),
        reader.read(createSpec({ key: 2, index: 1 })),
      ]);
      await reader.read(createSpec({ key: 1, index: 2 }));
      expect(calls).toHaveLength(3);
      await reader.read(createSpec({ key: 1, index: 1 }));
      expect(calls).toHaveLength(3);
      reader.close();
    });

    it("should evict tiles with no samples past the budget", async () => {
      let fetches = 0;
      const { reader } = createReader({
        budget: Size.bytes(LARGE_BYTES),
        readRemote: async () => {
          fetches++;
          return new framer.Frame([], []);
        },
      });
      await Promise.all(
        Array.from({ length: 1000 }, (_, index) => reader.read(createSpec({ index }))),
      );
      await reader.read(createSpec({ index: 0 }));
      expect(fetches).toBe(1001);
      reader.close();
    });

    it("should fetch a tile again once it is stale", async () => {
      const clock = new ManualClock();
      const { reader, calls } = createReader({
        staleThreshold: TimeSpan.milliseconds(20),
        now: clock.now,
      });
      await reader.read(createSpec());
      await reader.read(createSpec());
      expect(calls).toHaveLength(1);
      clock.advance(TimeSpan.milliseconds(20));
      await reader.read(createSpec());
      expect(calls).toHaveLength(2);
      reader.close();
    });

    describe("data type changes", () => {
      it("should read a cached tile again when the caller holds another data type", async () => {
        const { reader, calls } = createReader();
        await reader.read(createSpec());
        await reader.read(createSpec(), DataType.FLOAT64);
        expect(calls).toHaveLength(1);
        await reader.read(createSpec(), DataType.UINT8);
        expect(calls).toHaveLength(2);
        reader.close();
      });

      it("should drop a channel's cached tiles when a fetch returns a new data type", async () => {
        let dataType = DataType.FLOAT64;
        const { reader, calls } = createReader({
          readRemote: async (tr, keys) => {
            calls.push({ tr, keys });
            return new framer.Frame(
              keys,
              keys.map(
                (key) =>
                  new Series({
                    data: new Array(SAMPLES).fill(key),
                    dataType,
                    timeRange: tr,
                  }),
              ),
            );
          },
        });
        await reader.read(createSpec({ index: 1 }));
        await reader.read(createSpec({ index: 2 }));
        await reader.read(createSpec({ index: 1, key: 2 }));
        dataType = DataType.UINT8;
        const changed = await reader.read(createSpec({ index: 3 }));
        expect(changed.dataType.equals(DataType.UINT8)).toBe(true);
        expect(calls).toHaveLength(4);
        const refetched = await reader.read(createSpec({ index: 1 }));
        expect(refetched.dataType.equals(DataType.UINT8)).toBe(true);
        expect(calls).toHaveLength(5);
        const other = await reader.read(createSpec({ index: 1, key: 2 }));
        expect(other.dataType.equals(DataType.FLOAT64)).toBe(true);
        expect(calls).toHaveLength(5);
        reader.close();
      });

      it("should drop a channel's raw blocks when a tile fetch returns a new data type", async () => {
        const { reader, store } = createReader({
          readRemote: async (tr, keys) =>
            new framer.Frame(
              keys,
              keys.map(
                (key) =>
                  new Series({
                    data: new Array(SAMPLES).fill(key),
                    dataType: DataType.UINT8,
                    timeRange: tr,
                  }),
              ),
            ),
        });
        store.retainDataType(1, DataType.FLOAT64);
        const block = store.raw(1, 0);
        block.fill(
          [new Series({ data: new Float64Array([1]), timeRange: block.range })],
          block.range.end,
          1,
        );
        await reader.read(createSpec());
        expect(store.get(rawKey(1, 0))).toBeUndefined();
        reader.close();
      });
    });

    it("should reject every read in a failed fetch and cache nothing", async () => {
      let failing = true;
      const { reader } = createReader({
        readRemote: async () => {
          if (failing) throw new UnexpectedError("fetch failed");
          return new framer.Frame([1], [new Series(new Float64Array([1, 1, 1, 1]))]);
        },
      });
      await Promise.all([
        expect(reader.read(createSpec({ key: 1 }))).rejects.toThrow("fetch failed"),
        expect(reader.read(createSpec({ key: 2 }))).rejects.toThrow("fetch failed"),
      ]);
      failing = false;
      expect(Array.from(await reader.read(createSpec()))).toEqual([1, 1, 1, 1]);
      reader.close();
    });

    it("should reject a hung read with Unreachable after the fetch timeout", async () => {
      const { reader } = createReader({
        fetchTimeout: TimeSpan.milliseconds(20),
        readRemote: async () => await new Promise<framer.Frame>(() => {}),
      });
      const read = reader.read(createSpec());
      await expect(read).rejects.toSatisfy((e) => Unreachable.matches(e));
      await expect(read).rejects.toThrow("timed out after 20ms");
      reader.close();
    });

    it("should reject a read whose fetch throws synchronously", async () => {
      const { reader } = createReader({
        fetchTimeout: TimeSpan.milliseconds(20),
        readRemote: () => {
          throw new UnexpectedError("sync failure");
        },
      });
      await expect(reader.read(createSpec())).rejects.toThrow("sync failure");
      await sleep.sleep(TimeSpan.milliseconds(40));
      reader.close();
    });

    it("should reject pending reads when it closes", async () => {
      const { reader } = createReader({
        readRemote: async () => await new Promise<framer.Frame>(() => {}),
      });
      const queued = reader.read(createSpec({ index: 1 }));
      await sleep.sleep(TimeSpan.milliseconds(10));
      const inFlightRead = expect(queued).rejects.toThrow("tile reader is closed");
      const batched = expect(reader.read(createSpec({ index: 2 }))).rejects.toThrow(
        "tile reader is closed",
      );
      reader.close();
      await Promise.all([inFlightRead, batched]);
    });

    it("should reject reads after it closes", async () => {
      const { reader } = createReader();
      await reader.read(createSpec());
      reader.close();
      await expect(reader.read(createSpec())).rejects.toThrow("tile reader is closed");
    });
  });
});
