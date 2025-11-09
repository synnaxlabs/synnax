// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, Series, TimeStamp } from "@synnaxlabs/x";
import { beforeAll, describe, expect, it } from "vitest";

import { type channel } from "@/channel";
import { ReadAdapter, WriteAdapter } from "@/framer/adapter";
import { Frame } from "@/index";
import { createTestClient } from "@/testutil/client";

const client = createTestClient();

describe("WriteFrameAdapter", () => {
  let timeCh: channel.Channel;
  let dataCh: channel.Channel;
  let adapter: WriteAdapter;

  beforeAll(async () => {
    timeCh = await client.channels.create({
      name: `time-${Math.random()}-${TimeStamp.now().toString()}`,
      dataType: DataType.TIMESTAMP,
      isIndex: true,
    });
    dataCh = await client.channels.create({
      name: `data-${Math.random()}-${TimeStamp.now().toString()}`,
      dataType: DataType.FLOAT32,
      index: timeCh.key,
    });

    adapter = await WriteAdapter.open(client.channels.retriever, [
      timeCh.key,
      dataCh.key,
    ]);
  });

  it("should correctly adapt a record of keys to single values", async () => {
    const ts = TimeStamp.now().valueOf();
    const res = await adapter.adapt({ [timeCh.key]: ts, [dataCh.key]: 1 });
    expect(res.columns).toHaveLength(2);
    expect(res.series).toHaveLength(2);
    expect(res.get(timeCh.key)).toHaveLength(1);
    expect(res.get(dataCh.key)).toHaveLength(1);
    expect(res.get(timeCh.key).at(0)).toEqual(ts);
    expect(res.get(dataCh.key).at(0)).toEqual(1);
  });

  it("should correctly adapt a record of names to single values", async () => {
    const ts = TimeStamp.now().valueOf();
    const res2 = await adapter.adapt({ [timeCh.name]: ts, [dataCh.name]: 1 });
    expect(res2.columns).toHaveLength(2);
    expect(res2.series).toHaveLength(2);
    expect(res2.get(timeCh.key)).toHaveLength(1);
    expect(res2.get(dataCh.key)).toHaveLength(1);
    expect(res2.get(timeCh.key).at(0)).toEqual(ts);
    expect(res2.get(dataCh.key).at(0)).toEqual(1);
  });

  it("should correctly adapt a single name to a single series", async () => {
    const res3 = await adapter.adapt(dataCh.name, new Series(1));
    expect(res3.columns).toHaveLength(1);
    expect(res3.series).toHaveLength(1);
    expect(res3.get(dataCh.key)).toHaveLength(1);
    expect(res3.get(dataCh.key).at(0)).toEqual(1);
  });

  it("should correctly adapt multiple names to multiple series", async () => {
    const ts = TimeStamp.now().valueOf();
    const res4 = await adapter.adapt(
      [timeCh.name, dataCh.name],
      [new Series(ts), new Series(1)],
    );
    expect(res4.get(timeCh.key)).toHaveLength(1);
    expect(res4.get(dataCh.key)).toHaveLength(1);
    expect(res4.get(timeCh.key).at(0)).toEqual(ts);
    expect(res4.get(dataCh.key).at(0)).toEqual(1);
  });

  it("should correctly adapt a frame keyed by name", async () => {
    const ts = TimeStamp.now().valueOf();
    const fr = new Frame({
      [timeCh.name]: new Series(ts),
      [dataCh.name]: new Series(1),
    });
    const res = await adapter.adapt(fr);
    expect(res.columns).toHaveLength(2);
    expect(res.series).toHaveLength(2);
    expect(res.get(timeCh.key).at(0)).toEqual(ts);
    expect(res.get(dataCh.key).at(0)).toEqual(1);
  });

  it("should not modify a frame keyed by key", async () => {
    const ts = TimeStamp.now().valueOf();
    const fr = new Frame({ [timeCh.key]: new Series(ts), [dataCh.key]: new Series(1) });
    const res = await adapter.adapt(fr);
    expect(res.columns).toHaveLength(2);
    expect(res.series).toHaveLength(2);
    expect(res.get(timeCh.key).at(0)).toEqual(ts);
    expect(res.get(dataCh.key).at(0)).toEqual(1);
  });

  it("should correctly adapt a map of series", async () => {
    const ts = TimeStamp.now().valueOf();
    const m = new Map();
    m.set(timeCh.key, new Series(ts));
    const res = await adapter.adapt(m);
    expect(res.columns).toHaveLength(1);
    expect(res.series).toHaveLength(1);
    expect(res.get(timeCh.key)).toHaveLength(1);
    expect(res.get(timeCh.key).at(0)).toEqual(ts);
  });

  it("should correctly adapt a name and JSON value", async () => {
    const jsonChannel = await client.channels.create({
      name: `json-${Math.random()}-${TimeStamp.now().toString()}`,
      dataType: DataType.JSON,
      virtual: true,
    });
    const adapter = await WriteAdapter.open(client.channels.retriever, [
      jsonChannel.key,
    ]);
    const res = await adapter.adapt(jsonChannel.name, [{ dog: "blue" }]);
    expect(res.columns).toHaveLength(1);
    expect(res.series).toHaveLength(1);
    expect(res.get(jsonChannel.key)).toHaveLength(1);
    expect(res.get(jsonChannel.key).at(0)).toEqual({ dog: "blue" });
  });

  it("should correctly adapt a name and a json typed series", async () => {
    const jsonChannel = await client.channels.create({
      name: `json-${Math.random()}-${TimeStamp.now().toString()}`,
      dataType: DataType.JSON,
      virtual: true,
    });
    const adapter = await WriteAdapter.open(client.channels.retriever, [
      jsonChannel.key,
    ]);
    const res = await adapter.adapt(jsonChannel.name, new Series([{ dog: "blue" }]));
    expect(res.columns).toHaveLength(1);
    expect(res.series).toHaveLength(1);
    expect(res.get(jsonChannel.key)).toHaveLength(1);
    expect(res.get(jsonChannel.key).at(0)).toEqual({ dog: "blue" });
  });

  it("should correctly adapt a numeric value to a BigInt keyed by key", async () => {
    const bigIntCh = await client.channels.create({
      name: `bigint-${Math.random()}-${TimeStamp.now().toString()}`,
      dataType: DataType.INT64,
      virtual: true,
    });
    const res = await adapter.adapt({
      [bigIntCh.key]: 12,
    });
    expect(res.get(bigIntCh.key).at(0)).toEqual(12n);
  });

  describe("adaptParams", () => {
    it("should correctly adapt generic object keys", async () => {
      const res = await adapter.adaptParams([timeCh.name, dataCh.name]);
      expect(res).toContain(timeCh.key);
      expect(res).toContain(dataCh.key);
    });
  });

  describe("update", () => {
    it("should return false when updating with the same channels", async () => {
      const hasChanged = await adapter.update([timeCh.key, dataCh.key]);
      expect(hasChanged).toBe(false);
    });

    it("should return true when adding a new channel", async () => {
      const newCh = await client.channels.create({
        name: `new-${Math.random()}-${TimeStamp.now().toString()}`,
        dataType: DataType.FLOAT32,
        index: timeCh.key,
      });
      const hasChanged = await adapter.update([timeCh.key, dataCh.key, newCh.key]);
      expect(hasChanged).toBe(true);
    });

    it("should return true when removing a channel", async () => {
      const hasChanged = await adapter.update([timeCh.key]);
      expect(hasChanged).toBe(true);
    });

    it("should return true when replacing channels", async () => {
      const newCh = await client.channels.create({
        name: `replacement-${Math.random()}-${TimeStamp.now().toString()}`,
        dataType: DataType.FLOAT32,
        index: timeCh.key,
      });
      const hasChanged = await adapter.update([timeCh.key, newCh.key]);
      expect(hasChanged).toBe(true);
    });

    it("should return false when updating with same channels in different order", async () => {
      await adapter.update([timeCh.key, dataCh.key]);
      const hasChanged = await adapter.update([dataCh.key, timeCh.key]);
      expect(hasChanged).toBe(false);
    });

    it("should return false when updating with channel names that resolve to same keys", async () => {
      await adapter.update([timeCh.key, dataCh.key]);
      const hasChanged = await adapter.update([timeCh.name, dataCh.name]);
      expect(hasChanged).toBe(false);
    });
  });
});

describe("ReadFrameAdapter", () => {
  let timeCh: channel.Channel;
  let dataCh: channel.Channel;
  let extraCh: channel.Channel;
  let adapter: ReadAdapter;

  beforeAll(async () => {
    timeCh = await client.channels.create({
      name: `read-time-${Math.random()}-${TimeStamp.now().toString()}`,
      dataType: DataType.TIMESTAMP,
      isIndex: true,
    });
    dataCh = await client.channels.create({
      name: `read-data-${Math.random()}-${TimeStamp.now().toString()}`,
      dataType: DataType.FLOAT32,
      index: timeCh.key,
    });
    extraCh = await client.channels.create({
      name: `read-extra-${Math.random()}-${TimeStamp.now().toString()}`,
      dataType: DataType.FLOAT64,
      index: timeCh.key,
    });

    adapter = await ReadAdapter.open(client.channels.retriever, [
      timeCh.key,
      dataCh.key,
    ]);
  });

  describe("adapt", () => {
    describe("with keys (no conversion)", () => {
      describe("hot path - exact channel match", () => {
        it("should return frame unchanged when all channels match", () => {
          // HOT PATH: Frame has exactly the channels registered with adapter
          const ts = TimeStamp.now().valueOf();
          const inputFrame = new Frame({
            [timeCh.key]: new Series([ts]),
            [dataCh.key]: new Series([1.5]),
          });

          const result = adapter.adapt(inputFrame);

          // Frame should be returned unchanged (zero allocations)
          expect(result).toBe(inputFrame); // Same object reference
          expect(result.columns).toHaveLength(2);
          expect(result.has(timeCh.key)).toBe(true);
          expect(result.has(dataCh.key)).toBe(true);
          expect(result.get(timeCh.key).at(0)).toEqual(ts);
          expect(result.get(dataCh.key).at(0)).toEqual(1.5);
        });

        it("should preserve series data types in hot path", () => {
          const ts = TimeStamp.now().valueOf();
          const inputFrame = new Frame({
            [timeCh.key]: new Series({ data: [ts], dataType: DataType.TIMESTAMP }),
            [dataCh.key]: new Series({ data: [1.5], dataType: DataType.FLOAT32 }),
          });

          const result = adapter.adapt(inputFrame);

          // Data types should be preserved
          expect(result.get(timeCh.key).dataType).toEqual(DataType.TIMESTAMP);
          expect(result.get(dataCh.key).dataType).toEqual(DataType.FLOAT32);
        });
      });

      describe("cold path - filtering needed", () => {
        it("should filter out extra channels in key mode", () => {
          // COLD PATH: Frame has extra channels not in adapter
          const ts = TimeStamp.now().valueOf();
          const inputFrame = new Frame({
            [timeCh.key]: new Series([ts]),
            [dataCh.key]: new Series([1.5]),
            [extraCh.key]: new Series([999.0]), // Extra channel
          });

          const result = adapter.adapt(inputFrame);

          // Should filter out extraCh
          expect(result).not.toBe(inputFrame); // Different object (filtered)
          expect(result.columns).toHaveLength(2);
          expect(result.has(timeCh.key)).toBe(true);
          expect(result.has(dataCh.key)).toBe(true);
          expect(result.has(extraCh.key)).toBe(false);
        });

        it("should handle partial matches in key mode", () => {
          // Frame has some matching and some extra channels
          const ts = TimeStamp.now().valueOf();
          const inputFrame = new Frame({
            [timeCh.key]: new Series([ts]),
            [extraCh.key]: new Series([999.0]),
          });

          const result = adapter.adapt(inputFrame);

          expect(result.columns).toHaveLength(1);
          expect(result.has(timeCh.key)).toBe(true);
          expect(result.has(extraCh.key)).toBe(false);
        });

        it("should return empty frame when no channels match in key mode", () => {
          const inputFrame = new Frame({
            [extraCh.key]: new Series([999.0]),
          });

          const result = adapter.adapt(inputFrame);

          expect(result.columns).toHaveLength(0);
          expect(result.series).toHaveLength(0);
        });
      });
    });

    describe("with names (conversion)", () => {
      let nameAdapter: ReadAdapter;

      beforeAll(async () => {
        // Create adapter with channel names (triggers key-to-name mapping)
        nameAdapter = await ReadAdapter.open(client.channels.retriever, [
          timeCh.name,
          dataCh.name,
        ]);
      });

      describe("hot path - exact match, only convert", () => {
        it("should convert channel keys to names when all channels match", () => {
          // HOT PATH: Frame has exactly the channels in adapter
          const ts = TimeStamp.now().valueOf();
          const inputFrame = new Frame({
            [timeCh.key]: new Series([ts]),
            [dataCh.key]: new Series([2.5]),
          });

          const result = nameAdapter.adapt(inputFrame);

          // Output should have names instead of keys (one allocation for conversion)
          expect(result.columns).toHaveLength(2);
          expect(result.has(timeCh.name)).toBe(true);
          expect(result.has(dataCh.name)).toBe(true);
          expect(result.get(timeCh.name).at(0)).toEqual(ts);
          expect(result.get(dataCh.name).at(0)).toEqual(2.5);
        });

        it("should handle multiple values in hot path", () => {
          const ts = TimeStamp.now().valueOf();
          const inputFrame = new Frame({
            [timeCh.key]: new Series([ts, ts + 1000n]),
            [dataCh.key]: new Series([1.0, 2.0]),
          });

          const result = nameAdapter.adapt(inputFrame);

          expect(result.columns).toHaveLength(2);
          expect(result.get(timeCh.name)).toHaveLength(2);
          expect(result.get(dataCh.name)).toHaveLength(2);
          expect(result.get(timeCh.name).at(0)).toEqual(ts);
          expect(result.get(timeCh.name).at(1)).toEqual(ts + 1000n);
          expect(result.get(dataCh.name).at(0)).toEqual(1.0);
          expect(result.get(dataCh.name).at(1)).toEqual(2.0);
        });

        it("should preserve data types during name conversion", () => {
          const ts = TimeStamp.now().valueOf();
          const inputFrame = new Frame({
            [timeCh.key]: new Series({ data: [ts], dataType: DataType.TIMESTAMP }),
            [dataCh.key]: new Series({ data: [3.5], dataType: DataType.FLOAT32 }),
          });

          const result = nameAdapter.adapt(inputFrame);

          expect(result.get(timeCh.name).dataType).toEqual(DataType.TIMESTAMP);
          expect(result.get(dataCh.name).dataType).toEqual(DataType.FLOAT32);
        });
      });

      describe("cold path - filter and convert", () => {
        it("should filter out extra channels while converting", async () => {
          // COLD PATH: Frame has extra channels that need filtering
          const ts = TimeStamp.now().valueOf();
          const inputFrame = new Frame({
            [timeCh.key]: new Series([ts]),
            [dataCh.key]: new Series([1.5]),
            [extraCh.key]: new Series([999.0]), // Extra channel
          });

          const result = nameAdapter.adapt(inputFrame);

          // Should filter extraCh and convert remaining keys to names
          expect(result.columns).toHaveLength(2);
          expect(result.has(timeCh.name)).toBe(true);
          expect(result.has(dataCh.name)).toBe(true);
          expect(result.has(extraCh.key)).toBe(false);
        });

        it("should handle partial matches while converting", async () => {
          const filterAdapter = await ReadAdapter.open(client.channels.retriever, [
            timeCh.name,
          ]);

          const ts = TimeStamp.now().valueOf();
          const inputFrame = new Frame({
            [timeCh.key]: new Series([ts]),
            [extraCh.key]: new Series([999.0]),
          });

          const result = filterAdapter.adapt(inputFrame);

          expect(result.columns).toHaveLength(1);
          expect(result.has(timeCh.name)).toBe(true);
          expect(result.has(extraCh.key)).toBe(false);
          expect(result.get(timeCh.name).at(0)).toEqual(ts);
        });

        it("should return empty frame when no channels match", async () => {
          const filterAdapter = await ReadAdapter.open(client.channels.retriever, [
            timeCh.name,
          ]);

          const inputFrame = new Frame({
            [extraCh.key]: new Series([999.0]),
          });

          const result = filterAdapter.adapt(inputFrame);

          expect(result.columns).toHaveLength(0);
          expect(result.series).toHaveLength(0);
        });
      });
    });

    describe("edge cases", () => {
      it("should handle empty frames", () => {
        const inputFrame = new Frame({});

        const result = adapter.adapt(inputFrame);

        expect(result.columns).toHaveLength(0);
        expect(result.series).toHaveLength(0);
      });

      it("should handle frames with empty series", () => {
        const inputFrame = new Frame({
          [timeCh.key]: new Series({ data: [], dataType: DataType.TIMESTAMP }),
          [dataCh.key]: new Series({ data: [], dataType: DataType.FLOAT32 }),
        });

        const result = adapter.adapt(inputFrame);

        expect(result.columns).toHaveLength(2);
        expect(result.get(timeCh.key)).toHaveLength(0);
        expect(result.get(dataCh.key)).toHaveLength(0);
      });
    });

    describe("data integrity", () => {
      it("should preserve series values across multiple data types", async () => {
        const int64Ch = await client.channels.create({
          name: `read-int64-${Math.random()}-${TimeStamp.now().toString()}`,
          dataType: DataType.INT64,
          index: timeCh.key,
        });

        const testAdapter = await ReadAdapter.open(client.channels.retriever, [
          timeCh.key,
          dataCh.key,
          int64Ch.key,
        ]);

        const ts = TimeStamp.now().valueOf();
        const inputFrame = new Frame({
          [timeCh.key]: new Series([ts, ts + 1000n]),
          [dataCh.key]: new Series([1.5, 2.5]),
          [int64Ch.key]: new Series([100n, 200n]),
        });

        const result = testAdapter.adapt(inputFrame);

        // Verify all values preserved
        expect(result.get(timeCh.key).at(0)).toEqual(ts);
        expect(result.get(timeCh.key).at(1)).toEqual(ts + 1000n);
        expect(result.get(dataCh.key).at(0)).toEqual(1.5);
        expect(result.get(dataCh.key).at(1)).toEqual(2.5);
        expect(result.get(int64Ch.key).at(0)).toEqual(100n);
        expect(result.get(int64Ch.key).at(1)).toEqual(200n);
      });

      it("should preserve series lengths after filtering", () => {
        const ts = TimeStamp.now().valueOf();
        const inputFrame = new Frame({
          [timeCh.key]: new Series([ts, ts + 1000n, ts + 2000n]),
          [dataCh.key]: new Series([1.0, 2.0, 3.0]),
          [extraCh.key]: new Series([999.0, 888.0, 777.0]),
        });

        const result = adapter.adapt(inputFrame);

        // Lengths should be preserved for included channels
        expect(result.get(timeCh.key)).toHaveLength(3);
        expect(result.get(dataCh.key)).toHaveLength(3);
      });

      it("should preserve series order", () => {
        const ts = TimeStamp.now().valueOf();
        // Create frame with explicit column order
        const inputFrame = new Frame(
          [dataCh.key, timeCh.key],
          [new Series([1.0, 2.0, 3.0]), new Series([ts, ts + 1000n, ts + 2000n])],
        );

        const result = adapter.adapt(inputFrame);

        // Order should be preserved (dataCh first, then timeCh)
        expect(result.columns[0]).toEqual(dataCh.key);
        expect(result.columns[1]).toEqual(timeCh.key);
      });
    });

    describe("state management", () => {
      it("should handle multiple sequential updates correctly", async () => {
        // Start with NAME mode to enable filtering
        const newAdapter = await ReadAdapter.open(client.channels.retriever, [
          timeCh.name,
        ]);

        // Initial state: only timeCh registered
        const ts = TimeStamp.now().valueOf();
        const inputFrame = new Frame({
          [timeCh.key]: new Series([ts]),
          [dataCh.key]: new Series([1.5]),
        });

        // Should filter out dataCh and convert timeCh key to name
        let result = newAdapter.adapt(inputFrame);
        expect(result.columns).toHaveLength(1);
        expect(result.has(timeCh.name)).toBe(true);
        expect(result.has(dataCh.name)).toBe(false);

        // Update to include dataCh
        await newAdapter.update([timeCh.name, dataCh.name]);

        // Should now include both channels (converted to names)
        result = newAdapter.adapt(inputFrame);
        expect(result.columns).toHaveLength(2);
        expect(result.has(timeCh.name)).toBe(true);
        expect(result.has(dataCh.name)).toBe(true);
      });
    });

    describe("codec integration", () => {
      it("should update codec when channels change", async () => {
        const codecAdapter = await ReadAdapter.open(client.channels.retriever, [
          timeCh.key,
        ]);

        // Check initial codec state (keys array)
        expect(codecAdapter.keys).toHaveLength(1);

        // Update channels
        await codecAdapter.update([timeCh.key, dataCh.key]);

        // Codec should be updated (reflected in keys array)
        expect(codecAdapter.keys).toHaveLength(2);
      });
    });
  });

  describe("update", () => {
    it("should return false when updating with the same channels", async () => {
      const hasChanged = await adapter.update([timeCh.key, dataCh.key]);
      expect(hasChanged).toBe(false);
    });

    it("should return true when adding a new channel", async () => {
      const newCh = await client.channels.create({
        name: `read-new-${Math.random()}-${TimeStamp.now().toString()}`,
        dataType: DataType.FLOAT32,
        index: timeCh.key,
      });
      const hasChanged = await adapter.update([timeCh.key, dataCh.key, newCh.key]);
      expect(hasChanged).toBe(true);
    });

    it("should return true when removing a channel", async () => {
      const hasChanged = await adapter.update([timeCh.key]);
      expect(hasChanged).toBe(true);
    });

    it("should return true when replacing channels", async () => {
      const newCh = await client.channels.create({
        name: `read-replacement-${Math.random()}-${TimeStamp.now().toString()}`,
        dataType: DataType.FLOAT32,
        index: timeCh.key,
      });
      const hasChanged = await adapter.update([timeCh.key, newCh.key]);
      expect(hasChanged).toBe(true);
    });

    it("should return false when updating with same channels in different order", async () => {
      await adapter.update([timeCh.key, dataCh.key]);
      const hasChanged = await adapter.update([dataCh.key, timeCh.key]);
      expect(hasChanged).toBe(false);
    });

    it("should return false when updating with channel names that resolve to same keys", async () => {
      await adapter.update([timeCh.key, dataCh.key]);
      const hasChanged = await adapter.update([timeCh.name, dataCh.name]);
      expect(hasChanged).toBe(false);
    });
  });
});
