// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { id, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { createTestClient } from "@/testutil/client";

const client = createTestClient();

describe("Client", () => {
  describe("read + write", () => {
    it("should correctly write and read a frame of data", async () => {
      const time = await client.channels.create({
        name: id.create(),
        dataType: "timestamp",
        isIndex: true,
      });
      const data = await client.channels.create({
        name: id.create(),
        dataType: "float32",
        index: time.key,
      });
      const start = TimeStamp.now();
      await client.write(start, { [time.key]: [start], [data.key]: [1] });
      const frame = await client.read({ start, end: start.add(TimeSpan.seconds(1)) }, [
        time.key,
        data.key,
      ]);
      expect(Array.from(frame.get(time.key))).toEqual([start.valueOf()]);
      expect(Array.from(frame.get(data.key))).toEqual([1]);
    });
    it("should correctly write a single series of data", async () => {
      const time = await client.channels.create({
        name: id.create(),
        dataType: "timestamp",
        isIndex: true,
      });
      const data = await client.channels.create({
        name: id.create(),
        dataType: "float32",
        index: time.key,
      });
      const start = TimeStamp.now();
      await client.write(start, time.key, TimeStamp.now());
      await client.write(start, data.key, 1);
    });
  });
  describe("retrieveGroup", () => {
    it("should correctly retrieve the main channel group", async () => {
      const group = await client.channels.retrieveGroup();
      expect(group.name).toEqual("Channels");
    });
  });
  describe("readLatestN", () => {
    it("should correctly read the latest N samples from a single channel", async () => {
      const time = await client.channels.create({
        name: id.create(),
        dataType: "timestamp",
        isIndex: true,
      });
      const data = await client.channels.create({
        name: id.create(),
        dataType: "float32",
        index: time.key,
      });
      const start = TimeStamp.now();
      const timeData = [
        start,
        start.add(TimeSpan.seconds(1)),
        start.add(TimeSpan.seconds(2)),
        start.add(TimeSpan.seconds(3)),
        start.add(TimeSpan.seconds(4)),
        start.add(TimeSpan.seconds(5)),
        start.add(TimeSpan.seconds(6)),
        start.add(TimeSpan.seconds(7)),
        start.add(TimeSpan.seconds(8)),
        start.add(TimeSpan.seconds(9)),
      ];
      const dataValues = [1, 2, 3, 4, 5, 6, 7, 8, 9, 10];
      await client.write(start, { [time.key]: timeData, [data.key]: dataValues });

      const result = await client.readLatest(data.key, 3);
      expect(Array.from(result)).toEqual([8, 9, 10]);
    });

    it("should correctly read the latest N samples from multiple channels", async () => {
      const time = await client.channels.create({
        name: id.create(),
        dataType: "timestamp",
        isIndex: true,
      });
      const data1 = await client.channels.create({
        name: id.create(),
        dataType: "float32",
        index: time.key,
      });
      const data2 = await client.channels.create({
        name: id.create(),
        dataType: "float32",
        index: time.key,
      });
      const start = TimeStamp.now();
      const timeData = [
        start,
        start.add(TimeSpan.seconds(1)),
        start.add(TimeSpan.seconds(2)),
        start.add(TimeSpan.seconds(3)),
        start.add(TimeSpan.seconds(4)),
        start.add(TimeSpan.seconds(5)),
        start.add(TimeSpan.seconds(6)),
        start.add(TimeSpan.seconds(7)),
        start.add(TimeSpan.seconds(8)),
        start.add(TimeSpan.seconds(9)),
      ];
      const data1Values = [1, 2, 3, 4, 5, 6, 7, 8, 9, 10];
      const data2Values = [10, 20, 30, 40, 50, 60, 70, 80, 90, 100];
      await client.write(start, {
        [time.key]: timeData,
        [data1.key]: data1Values,
        [data2.key]: data2Values,
      });

      const frame = await client.readLatest([time.key, data1.key, data2.key], 3);
      expect(Array.from(frame.get(data1.key))).toEqual([8, 9, 10]);
      expect(Array.from(frame.get(data2.key))).toEqual([80, 90, 100]);
      expect(Array.from(frame.get(time.key))).toEqual([
        timeData[7].valueOf(),
        timeData[8].valueOf(),
        timeData[9].valueOf(),
      ]);
    });

    it("should return empty series when no data exists", async () => {
      const time = await client.channels.create({
        name: id.create(),
        dataType: "timestamp",
        isIndex: true,
      });
      const data = await client.channels.create({
        name: id.create(),
        dataType: "float32",
        index: time.key,
      });

      const result = await client.readLatest(data.key, 5);
      expect(Array.from(result)).toEqual([]);
    });

    it("should correctly handle N larger than available data", async () => {
      const time = await client.channels.create({
        name: id.create(),
        dataType: "timestamp",
        isIndex: true,
      });
      const data = await client.channels.create({
        name: id.create(),
        dataType: "float32",
        index: time.key,
      });
      const start = TimeStamp.now();
      const timeData = [
        start,
        start.add(TimeSpan.seconds(1)),
        start.add(TimeSpan.seconds(2)),
      ];
      const dataValues = [1, 2, 3];
      await client.write(start, { [time.key]: timeData, [data.key]: dataValues });

      const result = await client.readLatest(data.key, 10);
      expect(Array.from(result)).toEqual([1, 2, 3]);
    });
  });
});
