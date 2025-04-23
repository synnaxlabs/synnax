// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, TimeRange, TimeSpan, TimeStamp } from "@synnaxlabs/x/telem";
import { describe, expect, it, test } from "vitest";

import { UnauthorizedError, ValidationError } from "@/errors";
import { ALWAYS_INDEX_PERSIST_ON_AUTO_COMMIT, WriterMode } from "@/framer/writer";
import { newClient } from "@/setupspecs";
import { newIndexedPair } from "@/testutil/indexedPair";
import { secondsLinspace } from "@/testutil/telem";
import { randomSeries } from "@/util/telem";

const client = newClient();
describe("Writer", () => {
  describe("Writer", () => {
    test("basic write", async () => {
      const channels = await newIndexedPair(client);
      const start = TimeStamp.seconds(1);
      const writer = await client.openWriter({ start, channels });
      const [index, data] = channels;
      try {
        await writer.write({
          [index.key]: secondsLinspace(1, 10),
          [data.key]: randomSeries(10, data.dataType),
        });
        await writer.commit();
      } finally {
        await writer.close();
      }
      expect(true).toBeTruthy();
    });

    test("write to unknown channel key", async () => {
      const channels = await newIndexedPair(client);
      const writer = await client.openWriter({ start: TimeStamp.now(), channels });
      await expect(
        writer.write("billy bob", randomSeries(10, DataType.FLOAT64)),
      ).rejects.toThrow('Channel "billy bob" not found');
      await writer.close();
    });

    it("should not stream data when mode is set ot persist only", async () => {
      const channels = await newIndexedPair(client);
      const stream = await client.openStreamer(channels);
      const writer = await client.openWriter({
        start: TimeStamp.seconds(1),
        channels,
        mode: WriterMode.Persist,
      });
      const [index, data] = channels;
      try {
        await writer.write({
          [index.key]: secondsLinspace(1, 10),
          [data.key]: randomSeries(10, data.dataType),
        });
      } finally {
        await writer.close();
      }
      // Simulating a timeout.
      const v = await Promise.race([
        stream.read(),
        new Promise((resolve) => setTimeout(() => resolve(123), 250)),
      ]);
      expect(v).toEqual(123);
    });

    test("write with auto commit on", async () => {
      const channels = await newIndexedPair(client);
      const writer = await client.openWriter({
        start: TimeStamp.seconds(1),
        channels,
        enableAutoCommit: true,
      });
      const [index, data] = channels;
      try {
        await writer.write({
          [index.key]: secondsLinspace(1, 10),
          [data.key]: randomSeries(10, data.dataType),
        });
      } finally {
        await writer.close();
      }
      expect(true).toBeTruthy();

      const f = await client.read(
        new TimeRange(TimeStamp.seconds(1), TimeStamp.seconds(11)),
        index.key,
      );
      expect(f.length).toEqual(10);
    });

    test("write with auto commit and alwaysPersist", async () => {
      const channels = await newIndexedPair(client);
      const writer = await client.openWriter({
        start: TimeStamp.seconds(1),
        channels,
        enableAutoCommit: true,
        autoIndexPersistInterval: ALWAYS_INDEX_PERSIST_ON_AUTO_COMMIT,
      });
      const [index, data] = channels;
      try {
        await writer.write({
          [index.key]: secondsLinspace(1, 10),
          [data.key]: randomSeries(10, data.dataType),
        });
      } finally {
        await writer.close();
      }
      expect(true).toBeTruthy();
    });

    test("write with auto commit and a set interval", async () => {
      const channels = await newIndexedPair(client);
      const writer = await client.openWriter({
        start: TimeStamp.seconds(1),
        channels,
        enableAutoCommit: true,
        autoIndexPersistInterval: TimeSpan.milliseconds(100),
      });
      const [index, data] = channels;
      try {
        await writer.write({
          [index.key]: secondsLinspace(1, 10),
          [data.key]: randomSeries(10, data.dataType),
        });
      } finally {
        await writer.close();
      }
      expect(true).toBeTruthy();
    });

    test("write with auto-commit off and incorrect data length validation error", async () => {
      const channels = await newIndexedPair(client);
      const writer = await client.openWriter({
        start: TimeStamp.seconds(1),
        channels,
      });
      await expect(async () => {
        await writer.write({
          [channels[0].key]: secondsLinspace(1, 10),
          [channels[1].key]: randomSeries(11, channels[1].dataType),
        });
        await writer.commit();
        await writer.close();
      }).rejects.toThrow(ValidationError);
    });

    test("write with out of order timestamp", async () => {
      const indexCh = await client.channels.create({
        name: "idx",
        dataType: DataType.TIMESTAMP,
        isIndex: true,
      });

      const dataCh = await client.channels.create({
        name: "data",
        dataType: DataType.FLOAT64,
        index: indexCh.key,
      });

      const writer = await client.openWriter({
        start: TimeStamp.now(),
        channels: [indexCh.key, dataCh.key],
        enableAutoCommit: true,
      });

      await expect(async () => {
        for (let i = 0; i < 10; i++) {
          await new Promise((resolve) => setTimeout(resolve, 5));
          await writer.write({
            [indexCh.key]: BigInt(i),
            [dataCh.key]: i,
          });
        }
        await writer.close();
      }).rejects.toThrow(ValidationError);
    }, 5000000);

    test("write with errOnUnauthorized", async () => {
      const channels = await newIndexedPair(client);
      const w1 = await client.openWriter({
        start: new TimeStamp(TimeSpan.milliseconds(500)),
        channels,
      });

      await expect(
        client.openWriter({
          start: TimeStamp.now(),
          channels,
          errOnUnauthorized: true,
        }),
      ).rejects.toThrow(UnauthorizedError);
      await w1.close();
    });

    test("setAuthority", async () => {
      const channels = await newIndexedPair(client);
      const start = TimeStamp.seconds(5);
      const w1 = await client.openWriter({
        start,
        channels,
        authorities: 10,
        enableAutoCommit: true,
      });
      const w2 = await client.openWriter({
        start,
        channels,
        authorities: 20,
        enableAutoCommit: true,
      });
      const [index, data] = channels;
      await w1.write({
        [index.key]: secondsLinspace(5, 10),
        [data.key]: randomSeries(10, data.dataType),
      });
      let f = await index.read(TimeRange.MAX);
      expect(f.length).toEqual(0);

      await w1.setAuthority(100);
      await w1.write({
        [index.key]: secondsLinspace(5, 10),
        [data.key]: randomSeries(10, data.dataType),
      });
      await w1.close();
      await w2.close();
      f = await index.read(TimeRange.MAX);
      expect(f.length).toEqual(10);
    });

    test("setAuthority with name keys", async () => {
      const channels = await newIndexedPair(client);
      const start = TimeStamp.seconds(5);
      const w1 = await client.openWriter({
        start,
        channels,
        authorities: 10,
        enableAutoCommit: true,
      });
      const w2 = await client.openWriter({
        start,
        channels,
        authorities: 20,
        enableAutoCommit: true,
      });
      const [index, data] = channels;
      await w1.write({
        [index.key]: secondsLinspace(5, 10),
        [data.key]: randomSeries(10, data.dataType),
      });
      let f = await index.read(TimeRange.MAX);
      expect(f.length).toEqual(0);

      await w1.setAuthority({ [index.name]: 100, [data.name]: 100 });
      await w1.write({
        [index.key]: secondsLinspace(5, 10),
        [data.key]: randomSeries(10, data.dataType),
      });
      await w1.close();
      await w2.close();
      f = await index.read(TimeRange.MAX);
      expect(f.length).toEqual(10);
    });
  });

  // describe.skip("performance", async () => {
  //   const NUM_CHANNELS = 200;
  //   const ITERATIONS = 500;

  //   const idx = await client.channels.create(
  //     {
  //       name: "time",
  //       dataType: DataType.TIMESTAMP,
  //       isIndex: true,
  //     },
  //     { retrieveIfNameExists: true },
  //   );
  //   const channels = await client.channels.create(
  //     Array.from({ length: NUM_CHANNELS }, (_, i) => ({
  //       name: `ch-${i}`,
  //       dataType: DataType.FLOAT64,
  //       index: idx.key,
  //     })),
  //     { retrieveIfNameExists: true },
  //   );
  //   const channelKeys = channels.map((c) => c.key);
  //   const values = Object.fromEntries(channelKeys.map((k) => [k, 1]));
  //   channelKeys.push(idx.key);

  //   [true, false].forEach((reg) => {
  //     let wStart = TimeStamp.now();
  //     it(
  //       `should write 100,000 frames - reg codec - ${reg}`,
  //       async () => {
  //         if (!reg) wStart = wStart.sub(TimeSpan.days(100));
  //         const writer = await client.openWriter({
  //           start: wStart,
  //           channels: channelKeys,
  //           enableAutoCommit: true,
  //           useExperimentalCodec: false,
  //           mode: WriterMode.Stream,
  //         });
  //         const streamer = await client.openStreamer({
  //           channels: channelKeys,
  //           useExperimentalCodec: reg,
  //         });
  //         const start = performance.now();
  //         try {
  //           for (let i = 0; i < ITERATIONS; i++)
  //             // values[idx.key] = wStart.add(TimeSpan.seconds(i)).valueOf();
  //             await writer.write(values);
  //           // const d = await streamer.read();
  //         } finally {
  //           await writer.close();
  //           streamer.close();
  //         }
  //         console.log(
  //           `Experimental Codec: ${reg}`,
  //           TimeSpan.milliseconds(performance.now() - start).toString(),
  //         );
  //       },
  //       { timeout: 10000000 },
  //     );
  //   });
  // });
});
