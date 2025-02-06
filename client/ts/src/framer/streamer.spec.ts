// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, Rate, TimeStamp } from "@synnaxlabs/x/telem";
import { describe, expect, it, test } from "vitest";

import { type channel } from "@/channel";
import { newClient } from "@/setupspecs";

const client = newClient();

const newChannel = async (): Promise<channel.Channel> =>
  await client.channels.create({
    name: "test",
    leaseholder: 1,
    rate: Rate.hz(25),
    dataType: DataType.FLOAT64,
  });

describe("Streamer", () => {
  test("happy path", async () => {
    const ch = await newChannel();
    const streamer = await client.openStreamer(ch.key);
    await new Promise((resolve) => setTimeout(resolve, 100));
    const writer = await client.openWriter({
      start: TimeStamp.now(),
      channels: ch.key,
    });
    try {
      await writer.write(ch.key, new Float64Array([1, 2, 3]));
    } finally {
      await writer.close();
    }
    const d = await streamer.read();
    expect(Array.from(d.get(ch.key))).toEqual([1, 2, 3]);
  });
  test("open with config", async () => {
    const ch = await newChannel();
    await expect(client.openStreamer({ channels: ch.key })).resolves.not.toThrow();
  });
  it("should not throw an error when the streamer is opened with zero channels", async () => {
    await expect(client.openStreamer([])).resolves.not.toThrow();
  });
  it("should throw an error when the streamer is opened with a channel that does not exist", async () => {
    await expect(client.openStreamer([5678])).rejects.toThrow("not found");
  });
  test("downsample factor of 1", async () => {
    const ch = await newChannel();
    const streamer = await client.openStreamer({
      channels: ch.key,
      downsampleFactor: 1,
    });
    await new Promise((resolve) => setTimeout(resolve, 100));
    const writer = await client.openWriter({
      start: TimeStamp.now(),
      channels: ch.key,
    });
    try {
      await writer.write(ch.key, new Float64Array([1, 2, 3, 4, 5]));
    } finally {
      await writer.close();
    }
    const d = await streamer.read();
    expect(Array.from(d.get(ch.key))).toEqual([1, 2, 3, 4, 5]);
  });
  test("downsample factor of 2", async () => {
    const ch = await newChannel();
    const streamer = await client.openStreamer({
      channels: ch.key,
      downsampleFactor: 2,
    });
    await new Promise((resolve) => setTimeout(resolve, 100));
    const writer = await client.openWriter({
      start: TimeStamp.now(),
      channels: ch.key,
    });
    try {
      await writer.write(ch.key, new Float64Array([1, 2, 3, 4, 5, 6, 7, 8, 9, 10]));
    } finally {
      await writer.close();
    }
    const d = await streamer.read();
    expect(Array.from(d.get(ch.key))).toEqual([1, 3, 5, 7, 9]);
  });
  test("downsample factor of 10", async () => {
    const ch = await newChannel();
    const streamer = await client.openStreamer({
      channels: ch.key,
      downsampleFactor: 10,
    });
    await new Promise((resolve) => setTimeout(resolve, 100));
    const writer = await client.openWriter({
      start: TimeStamp.now(),
      channels: ch.key,
    });
    try {
      await writer.write(ch.key, new Float64Array([1, 2, 3, 4, 5, 6, 7, 8, 9, 10]));
    } finally {
      await writer.close();
    }
    const d = await streamer.read();
    expect(Array.from(d.get(ch.key))).toEqual([1]);
  });
});

describe("Streamer - Calculated Channels", () => {
  test("basic calculated channel streaming", async () => {
    // Create a timestamp index channel
    const timeChannel = await client.channels.create({
      name: "calc_test_time",
      isIndex: true,
      dataType: DataType.TIMESTAMP,
    });

    // Create source channels with the timestamp index
    const [channelA, channelB] = await client.channels.create([
      {
        name: "test_a",
        dataType: DataType.FLOAT64,
        index: timeChannel.key,
      },
      {
        name: "test_b",
        dataType: DataType.FLOAT64,
        index: timeChannel.key,
      },
    ]);

    // Create calculated channel that adds the two source channels
    const calcChannel = await client.channels.create({
      name: "test_calc",
      dataType: DataType.FLOAT64,
      index: timeChannel.key,
      virtual: true,
      expression: "return test_a + test_b",
      requires: [channelA.key, channelB.key],
    });

    // Set up streamer to listen for calculated results
    const streamer = await client.openStreamer(calcChannel.key);

    // Give streamer time to initialize
    await new Promise((resolve) => setTimeout(resolve, 100));

    // Write test data
    const startTime = TimeStamp.now();
    const writer = await client.openWriter({
      start: startTime,
      channels: [timeChannel.key, channelA.key, channelB.key],
    });

    try {
      // Write test values - each source gets 2.5 so sum should be 5.0
      await writer.write({
        [timeChannel.key]: [startTime],
        [channelA.key]: new Float64Array([2.5]),
        [channelB.key]: new Float64Array([2.5]),
      });

      // Read from streamer
      const frame = await streamer.read();

      // Verify calculated results
      const calcData = Array.from(frame.get(calcChannel.key));
      expect(calcData).toEqual([5.0]);
    } finally {
      await writer.close();
      streamer.close();
    }
  });
  test("calculated channel with constant", async () => {
    // Create an index channel for timestamps
    const timeChannel = await client.channels.create({
      name: "calc_const_time",
      isIndex: true,
      dataType: DataType.TIMESTAMP,
    });

    // Create base channel with index
    const baseChannel = await client.channels.create({
      name: "base_channel",
      dataType: DataType.FLOAT64,
      index: timeChannel.key,
    });

    // Create calculated channel that adds 5
    const calcChannel = await client.channels.create({
      name: "calc_const_channel",
      dataType: DataType.FLOAT64,
      index: timeChannel.key,
      virtual: true,
      expression: `return ${baseChannel.name} + 5`,
      requires: [baseChannel.key],
    });

    const streamer = await client.openStreamer(calcChannel.key);
    await new Promise((resolve) => setTimeout(resolve, 100));

    const startTime = TimeStamp.now();
    const writer = await client.openWriter({
      start: startTime,
      channels: [timeChannel.key, baseChannel.key],
    });

    try {
      const timestamps = [
        startTime,
        new TimeStamp(startTime.valueOf() + BigInt(1000000000)),
        new TimeStamp(startTime.valueOf() + BigInt(2000000000)),
      ];

      await writer.write({
        [timeChannel.key]: timestamps,
        [baseChannel.key]: new Float64Array([1, 2, 3]),
      });

      const frame = await streamer.read();
      const calcData = Array.from(frame.get(calcChannel.key));
      expect(calcData).toEqual([6, 7, 8]); // Original values + 5
    } finally {
      await writer.close();
      streamer.close();
    }
  });

  test("calculated channel with multiple operations", async () => {
    // Create timestamp channel
    const timeChannel = await client.channels.create({
      name: "calc_multi_time",
      isIndex: true,
      dataType: DataType.TIMESTAMP,
    });

    // Create source channels
    const [channelA, channelB] = await client.channels.create([
      {
        name: "multi_test_a",
        dataType: DataType.FLOAT64,
        index: timeChannel.key,
      },
      {
        name: "multi_test_b",
        dataType: DataType.FLOAT64,
        index: timeChannel.key,
      },
    ]);

    // Create calculated channel with multiple operations
    const calcChannel = await client.channels.create({
      name: "multi_calc",
      dataType: DataType.FLOAT64,
      index: timeChannel.key,
      virtual: true,
      expression: "return (multi_test_a * 2) + (multi_test_b / 2)",
      requires: [channelA.key, channelB.key],
    });

    const streamer = await client.openStreamer(calcChannel.key);
    await new Promise((resolve) => setTimeout(resolve, 100));

    const startTime = TimeStamp.now();
    const writer = await client.openWriter({
      start: startTime,
      channels: [timeChannel.key, channelA.key, channelB.key],
    });

    try {
      await writer.write({
        [timeChannel.key]: [startTime],
        [channelA.key]: new Float64Array([2.0]), // Will be multiplied by 2 = 4.0
        [channelB.key]: new Float64Array([4.0]), // Will be divided by 2 = 2.0
      });

      const frame = await streamer.read();
      const calcData = Array.from(frame.get(calcChannel.key));
      expect(calcData).toEqual([6.0]); // (2.0 * 2) + (4.0 / 2) = 4.0 + 2.0 = 6.0
    } finally {
      await writer.close();
      streamer.close();
    }
  });
});
