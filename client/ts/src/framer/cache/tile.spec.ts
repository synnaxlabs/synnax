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
  TimeStamp,
} from "@synnaxlabs/x";
import { describe, expect, it, vi } from "vitest";

import { type channel } from "@/channel";
import { UnexpectedError } from "@/errors";
import {
  tileRange,
  TileReader,
  type TileReaderProps,
  tileSpan,
  type TileSpec,
} from "@/framer/cache/tile";
import { type Transform } from "@/framer/cache/transform";
import { Frame } from "@/framer/frame";

interface Call {
  tr: TimeRange;
  keys: channel.Key[];
  reduction: Reduction;
}

const SAMPLES = 4;

// Returns a reader whose remote fetch records each call and answers every key with
// SAMPLES float64 samples equal to the key.
const createReader = (
  props: Partial<TileReaderProps> = {},
): { reader: TileReader; calls: Call[] } => {
  const calls: Call[] = [];
  const reader = new TileReader({
    batchDebounce: TimeSpan.milliseconds(5),
    readRemote: async (tr, keys, reduction) => {
      calls.push({ tr, keys, reduction });
      return new Frame(
        keys,
        keys.map(
          (key) =>
            new Series({
              data: new Float64Array(SAMPLES).fill(key),
              timeRange: tr,
            }),
        ),
      );
    },
    ...props,
  });
  return { reader, calls };
};

const SERIES_BYTES = SAMPLES * DataType.FLOAT64.density.valueOf();

const spec = (overrides: Partial<TileSpec> = {}): TileSpec => ({
  key: 1,
  level: 10,
  index: 3,
  pointLimit: 100,
  aggregation: "min_max",
  ...overrides,
});

describe("tile", () => {
  describe("tileSpan", () => {
    it("should span one millisecond at level zero", () => {
      expect(tileSpan(0).equals(TimeSpan.milliseconds(1))).toBe(true);
    });

    it("should double the span with each level", () => {
      expect(tileSpan(10).equals(TimeSpan.milliseconds(1024))).toBe(true);
    });
  });

  describe("tileRange", () => {
    it("should start the tile index spans after the epoch", () => {
      const tr = tileRange(spec({ level: 2, index: 3 }));
      expect(tr.start.equals(TimeStamp.milliseconds(12))).toBe(true);
      expect(tr.end.equals(TimeStamp.milliseconds(16))).toBe(true);
    });

    it("should cut the range short at the end of the spec", () => {
      const end = TimeStamp.milliseconds(14);
      const tr = tileRange(spec({ level: 2, index: 3, end }));
      expect(tr.end.equals(end)).toBe(true);
    });

    it("should cut the range short at the start of the spec", () => {
      const start = TimeStamp.milliseconds(13);
      const tr = tileRange(spec({ level: 2, index: 3, start }));
      expect(tr.start.equals(start)).toBe(true);
      expect(tr.end.equals(TimeStamp.milliseconds(16))).toBe(true);
    });

    it("should ignore a start before the tile", () => {
      const start = TimeStamp.milliseconds(4);
      const tr = tileRange(spec({ level: 2, index: 3, start }));
      expect(tr.start.equals(TimeStamp.milliseconds(12))).toBe(true);
    });

    it("should ignore an end after the tile", () => {
      const end = TimeStamp.milliseconds(20);
      const tr = tileRange(spec({ level: 2, index: 3, end }));
      expect(tr.end.equals(TimeStamp.milliseconds(16))).toBe(true);
    });
  });

  describe("TileReader", () => {
    it("should read the tile range with the spec's options", async () => {
      const { reader, calls } = createReader();
      const s = spec();
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

    it("should serve a second read of a tile from the cache", async () => {
      const { reader, calls } = createReader();
      const first = await reader.read(spec());
      const second = await reader.read(spec());
      expect(calls).toHaveLength(1);
      expect(second).toBe(first);
      reader.close();
    });

    it("should merge reads of one tile across channels into one request", async () => {
      const { reader, calls } = createReader();
      const [a, b] = await Promise.all([
        reader.read(spec({ key: 1 })),
        reader.read(spec({ key: 2 })),
      ]);
      expect(calls).toHaveLength(1);
      expect(calls[0].keys.sort()).toEqual([1, 2]);
      expect(Array.from(a)).toEqual([1, 1, 1, 1]);
      expect(Array.from(b)).toEqual([2, 2, 2, 2]);
      reader.close();
    });

    it("should share one fetch between concurrent reads of the same tile", async () => {
      const { reader, calls } = createReader();
      const [a, b] = await Promise.all([reader.read(spec()), reader.read(spec())]);
      expect(calls).toHaveLength(1);
      expect(calls[0].keys).toEqual([1]);
      expect(b).toBe(a);
      reader.close();
    });

    it("should fetch tiles with different bounds or options separately", async () => {
      const { reader, calls } = createReader();
      await Promise.all([
        reader.read(spec({ index: 3 })),
        reader.read(spec({ index: 4 })),
        reader.read(spec({ level: 11 })),
        reader.read(spec({ pointLimit: 50 })),
        reader.read(spec({ aggregation: "average" })),
      ]);
      expect(calls).toHaveLength(5);
      reader.close();
    });

    it("should cache tiles with different options under separate keys", async () => {
      const { reader, calls } = createReader();
      await reader.read(spec({ aggregation: "min_max" }));
      await reader.read(spec({ aggregation: "average" }));
      await reader.read(spec({ aggregation: "min_max" }));
      await reader.read(spec({ aggregation: "average" }));
      expect(calls).toHaveLength(2);
      reader.close();
    });

    it("should never cache a partial tile", async () => {
      const { reader, calls } = createReader();
      const end = tileRange(spec()).start.add(TimeSpan.milliseconds(100));
      await reader.read(spec({ end }));
      await reader.read(spec({ end }));
      expect(calls).toHaveLength(2);
      expect(calls[0].tr.end.equals(end)).toBe(true);
      expect(reader.size.valueOf()).toBe(0);
      reader.close();
    });

    it("should never cache a tile cut at its start", async () => {
      const { reader, calls } = createReader();
      const start = tileRange(spec()).start.add(TimeSpan.milliseconds(100));
      await reader.read(spec({ start }));
      await reader.read(spec({ start }));
      expect(calls).toHaveLength(2);
      expect(calls[0].tr.start.equals(start)).toBe(true);
      expect(reader.size.valueOf()).toBe(0);
      reader.close();
    });

    it("should return an empty series for a channel with no data", async () => {
      const { reader } = createReader({
        readRemote: async () => new Frame([], []),
      });
      const series = await reader.read(spec());
      expect(series.length).toBe(0);
      reader.close();
    });

    it("should apply the transform to every fetched series", async () => {
      const convert = vi.fn((s: Series) => s.convert(DataType.FLOAT32));
      const transform: Transform = { resolveDataType: () => DataType.FLOAT32, convert };
      const { reader } = createReader({ transform });
      const series = await reader.read(spec());
      expect(convert).toHaveBeenCalledTimes(1);
      expect(series.dataType.equals(DataType.FLOAT32)).toBe(true);
      reader.close();
    });

    it("should track the byte size of the cached tiles", async () => {
      const { reader } = createReader();
      await reader.read(spec({ index: 1 }));
      await reader.read(spec({ index: 2 }));
      expect(reader.size.valueOf()).toBe(2 * SERIES_BYTES);
      reader.close();
    });

    it("should evict the least recently used tile past the budget", async () => {
      const { reader, calls } = createReader({ budget: Size.bytes(2 * SERIES_BYTES) });
      await reader.read(spec({ index: 1 }));
      await reader.read(spec({ index: 2 }));
      await reader.read(spec({ index: 1 }));
      await reader.read(spec({ index: 3 }));
      expect(reader.size.valueOf()).toBe(2 * SERIES_BYTES);
      expect(calls).toHaveLength(3);
      await reader.read(spec({ index: 1 }));
      expect(calls).toHaveLength(3);
      await reader.read(spec({ index: 2 }));
      expect(calls).toHaveLength(4);
      reader.close();
    });

    it("should keep a tile a caller holds past the budget", async () => {
      const { reader, calls } = createReader({ budget: Size.bytes(SERIES_BYTES) });
      const heldTile = await reader.read(spec({ index: 1 }));
      heldTile.acquire();
      await reader.read(spec({ index: 2 }));
      expect(reader.size.valueOf()).toBe(SERIES_BYTES);
      await reader.read(spec({ index: 1 }));
      expect(calls).toHaveLength(2);
      heldTile.release();
      await reader.read(spec({ index: 3 }));
      expect(reader.size.valueOf()).toBe(SERIES_BYTES);
      await reader.read(spec({ index: 1 }));
      expect(calls).toHaveLength(4);
      reader.close();
    });

    it("should fetch a tile again once it is stale", async () => {
      const { reader, calls } = createReader({
        staleThreshold: TimeSpan.milliseconds(20),
      });
      await reader.read(spec());
      await reader.read(spec());
      expect(calls).toHaveLength(1);
      await sleep.sleep(TimeSpan.milliseconds(30));
      await reader.read(spec());
      expect(calls).toHaveLength(2);
      expect(reader.size.valueOf()).toBe(SERIES_BYTES);
      reader.close();
    });

    it("should reject every read in a failed fetch and cache nothing", async () => {
      let failing = true;
      const { reader } = createReader({
        readRemote: async () => {
          if (failing) throw new UnexpectedError("fetch failed");
          return new Frame([1], [new Series(new Float64Array([1, 1, 1, 1]))]);
        },
      });
      await Promise.all([
        expect(reader.read(spec({ key: 1 }))).rejects.toThrow("fetch failed"),
        expect(reader.read(spec({ key: 2 }))).rejects.toThrow("fetch failed"),
      ]);
      expect(reader.size.valueOf()).toBe(0);
      failing = false;
      expect(Array.from(await reader.read(spec()))).toEqual([1, 1, 1, 1]);
      reader.close();
    });

    it("should reject a hung read with Unreachable after the fetch timeout", async () => {
      const { reader } = createReader({
        fetchTimeout: TimeSpan.milliseconds(20),
        readRemote: async () => await new Promise<Frame>(() => {}),
      });
      await expect(reader.read(spec())).rejects.toSatisfy((e) =>
        Unreachable.matches(e),
      );
      reader.close();
    });

    it("should reject a read whose fetch throws synchronously", async () => {
      const { reader } = createReader({
        fetchTimeout: TimeSpan.milliseconds(20),
        readRemote: () => {
          throw new UnexpectedError("sync failure");
        },
      });
      await expect(reader.read(spec())).rejects.toThrow("sync failure");
      await sleep.sleep(TimeSpan.milliseconds(40));
      reader.close();
    });

    it("should reject pending reads when it closes", async () => {
      const { reader } = createReader({
        readRemote: async () => await new Promise<Frame>(() => {}),
      });
      const queued = reader.read(spec({ index: 1 }));
      await sleep.sleep(TimeSpan.milliseconds(10));
      const inFlightRead = expect(queued).rejects.toThrow(UnexpectedError);
      const batched = expect(reader.read(spec({ index: 2 }))).rejects.toThrow(
        UnexpectedError,
      );
      reader.close();
      await Promise.all([inFlightRead, batched]);
    });

    it("should reject reads after it closes and drop its tiles", async () => {
      const { reader } = createReader();
      await reader.read(spec());
      reader.close();
      expect(reader.size.valueOf()).toBe(0);
      await expect(reader.read(spec())).rejects.toThrow(UnexpectedError);
    });
  });
});
