// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, id, TimeRange, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { describe, expect, it, test } from "vitest";

import { UnauthorizedError, ValidationError } from "@/errors";
import { ALWAYS_INDEX_PERSIST_ON_AUTO_COMMIT, WriterMode } from "@/framer/writer";
import { newIndexedPair } from "@/testutil/channels";
import { createTestClient } from "@/testutil/client";
import { secondsLinspace } from "@/testutil/telem";
import { randomSeries } from "@/util/telem";

const client = createTestClient();

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
      expect(true).toBe(true);
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
      expect(true).toBe(true);

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
      expect(true).toBe(true);
    });

    test("write with auto commit and a set interval", async () => {
      const channels = await newIndexedPair(client);
      const writer = await client.openWriter({
        start: TimeStamp.seconds(1),
        channels,
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
      expect(true).toBe(true);
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
        name: id.create(),
        dataType: DataType.TIMESTAMP,
        isIndex: true,
      });

      const dataCh = await client.channels.create({
        name: id.create(),
        dataType: DataType.FLOAT64,
        index: indexCh.key,
      });

      const writer = await client.openWriter({
        start: TimeStamp.now(),
        channels: [indexCh.key, dataCh.key],
      });

      await expect(async () => {
        for (let i = 0; i < 10; i++) {
          await new Promise((resolve) => setTimeout(resolve, 5));
          await writer.write({
            [indexCh.key]: new TimeStamp(i),
            [dataCh.key]: i,
          });
        }
      }).rejects.toThrow(ValidationError);
      await expect(async () => {
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
      });
      const w2 = await client.openWriter({
        start,
        channels,
        authorities: 20,
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
      });
      const w2 = await client.openWriter({
        start,
        channels,
        authorities: 20,
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
});
