// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, id, Series, TimeRange, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { describe, expect, it, test } from "vitest";

import { UnauthorizedError, ValidationError } from "@/errors";
import { ALWAYS_INDEX_PERSIST_ON_AUTO_COMMIT, WriterMode } from "@/framer/writer";
import { createTestClient, newIndexedPair, secondsLinspace } from "@/testutil";
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
        writer.write("nonexistent_channel", randomSeries(10, DataType.FLOAT64)),
      ).rejects.toThrow('Channel "nonexistent_channel" not found');
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

    test("auto-index generates server-side timestamps for data-only writes", async () => {
      const [index, data] = await newIndexedPair(client);
      const before = TimeStamp.now();
      const writer = await client.openWriter({
        start: before,
        channels: [data.key],
        autoIndex: true,
      });
      try {
        await writer.write(data.key, randomSeries(5, data.dataType));
        await writer.commit();
      } finally {
        await writer.close();
      }
      const f = await client.read(
        new TimeRange(before, TimeStamp.now().add(TimeSpan.seconds(1))),
        index.key,
      );
      expect(f.length).toEqual(5);
    });

    test("auto-index mixes user-provided and generated timestamps in one writer", async () => {
      const [index, data] = await newIndexedPair(client);
      const userStamps = secondsLinspace(200, 2);
      const beforeAuto = TimeStamp.now();
      const writer = await client.openWriter({
        start: TimeStamp.seconds(200),
        channels: [index.key, data.key],
        autoIndex: true,
      });
      try {
        await writer.write({
          [index.key]: userStamps,
          [data.key]: randomSeries(2, data.dataType),
        });
        await writer.write(data.key, randomSeries(3, data.dataType));
        await writer.commit();
      } finally {
        await writer.close();
      }
      const userF = await client.read(
        new TimeRange(TimeStamp.seconds(200), TimeStamp.seconds(202)),
        index.key,
      );
      expect(userF.data).toEqual(new BigInt64Array(userStamps.map((v) => v.valueOf())));
      const autoF = await client.read(
        new TimeRange(beforeAuto, TimeStamp.now().add(TimeSpan.seconds(1))),
        index.key,
      );
      expect(autoF.length).toEqual(3);
    });

    test("auto-index leaves user-provided index data untouched", async () => {
      const [index, data] = await newIndexedPair(client);
      const writer = await client.openWriter({
        start: TimeStamp.seconds(200),
        channels: [index.key, data.key],
        autoIndex: true,
      });
      try {
        await writer.write({
          [index.key]: secondsLinspace(200, 3),
          [data.key]: randomSeries(3, data.dataType),
        });
        await writer.commit();
      } finally {
        await writer.close();
      }
      const f = await client.read(
        new TimeRange(TimeStamp.seconds(200), TimeStamp.seconds(203)),
        index.key,
      );
      expect(f.data).toEqual(
        new BigInt64Array(secondsLinspace(200, 3).map((v) => v.valueOf())),
      );
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
    });

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

  describe("Variable Channels", () => {
    test("write and read string data", async () => {
      const index = await client.channels.create({
        name: id.create(),
        isIndex: true,
        dataType: DataType.TIMESTAMP,
        leaseholder: 1,
      });
      const data = await client.channels.create({
        name: id.create(),
        index: index.key,
        dataType: DataType.STRING,
        leaseholder: 1,
      });
      const writer = await client.openWriter({
        start: TimeStamp.seconds(1),
        channels: [index, data],
      });
      try {
        await writer.write({
          [index.key]: secondsLinspace(1, 3),
          [data.key]: new Series({
            data: ["hello", "world", "foo"],
            dataType: DataType.STRING,
          }),
        });
        await writer.commit();
      } finally {
        await writer.close();
      }
      const f = await data.read(TimeRange.MAX);
      expect(f.toStrings()).toEqual(["hello", "world", "foo"]);
    });

    test("write and read JSON data", async () => {
      const index = await client.channels.create({
        name: id.create(),
        isIndex: true,
        dataType: DataType.TIMESTAMP,
        leaseholder: 1,
      });
      const data = await client.channels.create({
        name: id.create(),
        index: index.key,
        dataType: DataType.JSON,
        leaseholder: 1,
      });
      const writer = await client.openWriter({
        start: TimeStamp.seconds(1),
        channels: [index, data],
      });
      try {
        await writer.write({
          [index.key]: secondsLinspace(1, 2),
          [data.key]: new Series({
            data: [{ key: "value" }, { num: 42 }],
            dataType: DataType.JSON,
          }),
        });
        await writer.commit();
      } finally {
        await writer.close();
      }
      const f = await data.read(TimeRange.MAX);
      expect(f.length).toEqual(2);
    });

    test("write mixed fixed and variable channels", async () => {
      const index = await client.channels.create({
        name: id.create(),
        isIndex: true,
        dataType: DataType.TIMESTAMP,
        leaseholder: 1,
      });
      const floatCh = await client.channels.create({
        name: id.create(),
        index: index.key,
        dataType: DataType.FLOAT64,
        leaseholder: 1,
      });
      const strCh = await client.channels.create({
        name: id.create(),
        index: index.key,
        dataType: DataType.STRING,
        leaseholder: 1,
      });
      const writer = await client.openWriter({
        start: TimeStamp.seconds(1),
        channels: [index, floatCh, strCh],
      });
      try {
        await writer.write({
          [index.key]: secondsLinspace(1, 3),
          [floatCh.key]: new Float64Array([1.1, 2.2, 3.3]),
          [strCh.key]: new Series({
            data: ["a", "b", "c"],
            dataType: DataType.STRING,
          }),
        });
        await writer.commit();
      } finally {
        await writer.close();
      }
      const floatData = await floatCh.read(TimeRange.MAX);
      expect(floatData.length).toEqual(3);
      const strData = await strCh.read(TimeRange.MAX);
      expect(strData.toStrings()).toEqual(["a", "b", "c"]);
    });

    test("write strings with embedded newlines", async () => {
      const index = await client.channels.create({
        name: id.create(),
        isIndex: true,
        dataType: DataType.TIMESTAMP,
        leaseholder: 1,
      });
      const data = await client.channels.create({
        name: id.create(),
        index: index.key,
        dataType: DataType.STRING,
        leaseholder: 1,
      });
      const writer = await client.openWriter({
        start: TimeStamp.seconds(1),
        channels: [index, data],
      });
      try {
        await writer.write({
          [index.key]: secondsLinspace(1, 2),
          [data.key]: new Series({
            data: ["line1\nline2", "no newline"],
            dataType: DataType.STRING,
          }),
        });
        await writer.commit();
      } finally {
        await writer.close();
      }
      const f = await data.read(TimeRange.MAX);
      expect(f.toStrings()).toEqual(["line1\nline2", "no newline"]);
    });
  });
});
