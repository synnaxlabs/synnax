// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { EOF, type WebSocketClient } from "@synnaxlabs/freighter";
import {
  type Aggregation,
  DataType,
  type Series,
  TimeRange,
  TimeSpan,
  TimeStamp,
} from "@synnaxlabs/x";
import { describe, expect, test, vi } from "vitest";

import { channel } from "@/channel";
import { ValidationError } from "@/errors";
import { framer } from "@/framer";
import {
  createTestClient,
  newIndexedPair,
  randomSeries,
  secondsLinspace,
} from "@/testutil";

const client = createTestClient();

const MAX_FACTOR = 2 ** 32 - 1;

describe("Iterator", () => {
  describe("dispose", () => {
    test("should close the iterator once at the end of an await using block", async () => {
      const channels = await newIndexedPair(client);
      const iter = await client.openIterator(TimeRange.MAX, channels);
      const close = vi.spyOn(iter, "close");
      {
        await using _ = iter;
      }
      expect(close).toHaveBeenCalledTimes(1);
    });

    test("should dispose an iterator that was closed explicitly", async () => {
      const channels = await newIndexedPair(client);
      const iter = await client.openIterator(TimeRange.MAX, channels);
      const close = vi.spyOn(iter, "close");
      {
        await using i = iter;
        await i.close();
      }
      expect(close).toHaveBeenCalledTimes(2);
    });
  });

  test("happy path", async () => {
    const channels = await newIndexedPair(client);
    const writer = await client.openWriter({
      start: TimeStamp.seconds(1),
      channels,
    });
    const [idx_ch, data_ch] = channels;
    try {
      await writer.write({
        [idx_ch.key]: secondsLinspace(1, 10),
        [data_ch.key]: randomSeries(10, data_ch.dataType),
      });
      await writer.write({
        [idx_ch.key]: secondsLinspace(11, 10),
        [data_ch.key]: randomSeries(10, data_ch.dataType),
      });
      await writer.write({
        [idx_ch.key]: secondsLinspace(21, 10),
        [data_ch.key]: randomSeries(10, data_ch.dataType),
      });
      await writer.commit();
    } finally {
      await writer.close();
    }

    await using iter = await client.openIterator(
      new TimeRange(TimeStamp.seconds(1), TimeSpan.seconds(30)),
      channels,
    );
    expect(await iter.seekFirst()).toBe(true);
    let c = 0;
    while (await iter.next(TimeSpan.seconds(1))) {
      c++;
      expect(iter.value.get(idx_ch.key)).toHaveLength(1);
      expect(iter.value.get(data_ch.key)).toHaveLength(1);
    }
    expect(c).toEqual(29);
  });
  test("chunk size", async () => {
    const channels = await newIndexedPair(client);
    const [idx_ch, data_ch] = channels;
    const writer = await client.openWriter({
      start: TimeStamp.seconds(1),
      channels,
    });
    await writer.write({
      [idx_ch.key]: secondsLinspace(1, 10),
      [data_ch.key]: randomSeries(10, data_ch.dataType),
    });
    await writer.close();
    await using iter = await client.openIterator(TimeRange.MAX, channels, {
      chunkSize: 4,
    });
    expect(await iter.seekFirst()).toBe(true);

    expect(await iter.next(framer.AUTO_SPAN)).toBe(true);
    expect(iter.value.get(idx_ch.key).data).toEqual(
      new BigInt64Array(secondsLinspace(1, 4).map((v) => v.valueOf())),
    );

    expect(await iter.next(framer.AUTO_SPAN)).toBe(true);
    expect(iter.value.get(idx_ch.key).data).toEqual(
      new BigInt64Array(secondsLinspace(5, 4).map((v) => v.valueOf())),
    );

    expect(await iter.next(framer.AUTO_SPAN)).toBe(true);
    expect(iter.value.get(idx_ch.key).data).toEqual(
      new BigInt64Array(secondsLinspace(9, 2).map((v) => v.valueOf())),
    );

    expect(await iter.next(framer.AUTO_SPAN)).toBe(false);
  });
  describe("next after the Core closes the iterator with an error", () => {
    const time = new channel.Channel({
      key: 1,
      name: "time",
      dataType: DataType.TIMESTAMP,
      isIndex: true,
    });

    test("should throw the error the Core closed the iterator with", async () => {
      const state = { opened: false };
      const stream = {
        send: () => {
          if (state.opened) throw new EOF();
        },
        closeSend: () => {},
        received: () => false,
        receive: async () => {
          if (state.opened) throw new ValidationError("invalid read");
          state.opened = true;
          return {
            variant: framer.IteratorResponseVariant.Ack,
            ack: true,
            command: framer.IteratorCommand.Open,
          };
        },
      };
      const transport = { withCodec: () => transport, stream: async () => stream };
      const iter = await framer.Iterator._open(
        TimeRange.MAX,
        [time.key],
        async () => [time.payload],
        transport as unknown as WebSocketClient,
      );
      await expect(iter.next(framer.AUTO_SPAN)).rejects.toThrow(ValidationError);
    });
  });

  describe("stride", () => {
    test("should keep every second sample under a factor of 2", async () => {
      const channels = await newIndexedPair(client);
      const [idx_ch, data_ch] = channels;
      const writer = await client.openWriter({ start: TimeStamp.seconds(1), channels });
      await writer.write({
        [idx_ch.key]: secondsLinspace(1, 8),
        [data_ch.key]: new Float64Array([1, 2, 3, 4, 5, 6, 7, 8]),
      });
      await writer.close();
      await using iter = await client.openIterator(TimeRange.MAX, channels, {
        reduction: { variant: "stride", factor: 2 },
      });
      expect(await iter.seekFirst()).toBe(true);
      expect(await iter.next(framer.AUTO_SPAN)).toBe(true);
      const series = iter.value.get(data_ch.key);
      expect(series.data).toEqual(new Float64Array([1, 3, 5, 7]));
      expect(series.series[0].alignmentMultiple).toEqual(2n);
      expect(await iter.next(framer.AUTO_SPAN)).toBe(false);
    });
    test("should keep every third sample under a factor of 3", async () => {
      const channels = await newIndexedPair(client);
      const [idx_ch, data_ch] = channels;
      const writer = await client.openWriter({ start: TimeStamp.seconds(1), channels });
      await writer.write({
        [idx_ch.key]: secondsLinspace(1, 9),
        [data_ch.key]: new Float64Array([1, 2, 3, 4, 5, 6, 7, 8, 9]),
      });
      await writer.close();
      await using iter = await client.openIterator(TimeRange.MAX, channels, {
        reduction: { variant: "stride", factor: 3 },
      });
      expect(await iter.seekFirst()).toBe(true);
      expect(await iter.next(framer.AUTO_SPAN)).toBe(true);
      expect(iter.value.get(data_ch.key).data).toEqual(new Float64Array([1, 4, 7]));
      expect(await iter.next(framer.AUTO_SPAN)).toBe(false);
    });
    test("should keep every sample under a factor of 1", async () => {
      const channels = await newIndexedPair(client);
      const [idx_ch, data_ch] = channels;
      const writer = await client.openWriter({ start: TimeStamp.seconds(1), channels });
      await writer.write({
        [idx_ch.key]: secondsLinspace(1, 4),
        [data_ch.key]: new Float64Array([1, 2, 3, 4]),
      });
      await writer.close();
      await using iter = await client.openIterator(TimeRange.MAX, channels, {
        reduction: { variant: "stride", factor: 1 },
      });
      expect(await iter.seekFirst()).toBe(true);
      expect(await iter.next(framer.AUTO_SPAN)).toBe(true);
      expect(iter.value.get(data_ch.key).data).toEqual(new Float64Array([1, 2, 3, 4]));
      expect(await iter.next(framer.AUTO_SPAN)).toBe(false);
    });

    describe("config", () => {
      const stride = (factor: number) => ({
        reduction: { variant: "stride" as const, factor },
      });
      test("defaults to no reduction", () => {
        expect(framer.iteratorConfigZ.parse({}).reduction).toBeUndefined();
      });
      test.each([0, 1, 2, 10, MAX_FACTOR])("accepts %i", (factor) => {
        expect(framer.iteratorConfigZ.parse(stride(factor)).reduction).toEqual(
          stride(factor).reduction,
        );
      });
      // The wire field is unsigned, so a factor the schema lets through would be
      // reinterpreted by the Core instead of rejected.
      test.each([-1, 1.5, MAX_FACTOR + 1])("rejects %s", (factor) => {
        expect(() => framer.iteratorConfigZ.parse(stride(factor))).toThrow(/factor/);
      });
    });
  });
  describe("aggregation", () => {
    const readReduced = async (
      aggregation: Aggregation,
      pointLimit: number,
    ): Promise<Series> => {
      const channels = await newIndexedPair(client);
      const [idx_ch, data_ch] = channels;
      const writer = await client.openWriter({ start: TimeStamp.seconds(1), channels });
      await writer.write({
        [idx_ch.key]: secondsLinspace(1, 8),
        [data_ch.key]: new Float64Array([4, 1, 3, 8, 5, 2, 6, 7]),
      });
      await writer.close();
      await using iter = await client.openIterator(TimeRange.MAX, channels, {
        reduction: { variant: "limit", aggregation, pointLimit },
      });
      expect(await iter.seekFirst()).toBe(true);
      expect(await iter.next(framer.AUTO_SPAN)).toBe(true);
      const data = iter.value.get(data_ch.key).series[0];
      expect(iter.value.get(idx_ch.key).series[0].alignment).toEqual(data.alignment);
      expect(await iter.next(framer.AUTO_SPAN)).toBe(false);
      return data;
    };

    test("min_max keeps the extremes of each group in order", async () => {
      const data = await readReduced("min_max", 4);
      expect(Array.from(data)).toEqual([1, 8, 2, 7]);
      expect(data.alignmentMultiple).toEqual(2n);
    });

    test("average keeps the mean of each group", async () => {
      const data = await readReduced("average", 4);
      expect(Array.from(data)).toEqual([2.5, 5.5, 3.5, 6.5]);
      expect(data.alignmentMultiple).toEqual(2n);
    });

    test("decimate keeps the first sample of each group", async () => {
      const data = await readReduced("decimate", 2);
      expect(Array.from(data)).toEqual([4, 5]);
      expect(data.alignmentMultiple).toEqual(4n);
    });

    test("returns every sample under the point limit", async () => {
      const data = await readReduced("min_max", 8);
      expect(Array.from(data)).toEqual([4, 1, 3, 8, 5, 2, 6, 7]);
      expect(data.alignmentMultiple).toEqual(1n);
    });

    describe("config", () => {
      const limit = (pointLimit: number) => ({
        reduction: { variant: "limit" as const, pointLimit },
      });
      test("defaults the aggregation to min_max", () => {
        expect(framer.iteratorConfigZ.parse(limit(4)).reduction).toEqual({
          variant: "limit",
          aggregation: "min_max",
          pointLimit: 4,
        });
      });
      test.each([0, -1, 1.5, 2 ** 32])("rejects a point limit of %s", (pointLimit) => {
        expect(() => framer.iteratorConfigZ.parse(limit(pointLimit))).toThrow(
          /pointLimit/,
        );
      });
    });
  });
});
