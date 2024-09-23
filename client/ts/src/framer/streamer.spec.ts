// Copyright 2024 Synnax Labs, Inc.
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
  test("downsample factor of 1", async () => {
    const ch = await newChannel();
    const streamer = await client.openStreamer({channels: ch.key, downsampleFactor: 1});
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
    const streamer = await client.openStreamer({channels: ch.key, downsampleFactor: 2});
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
    const streamer = await client.openStreamer({channels: ch.key, downsampleFactor: 10});
    await new Promise((resolve) => setTimeout(resolve, 100));
    const writer = await client.openWriter({
      start: TimeStamp.now(),
      channels: ch.key,
    });
    try {
      await writer.write(ch.key, new Float64Array([1, 2, 3, 4, 5, 6,7,8,9,10]));
    } finally {
      await writer.close();
    }
    const d = await streamer.read();
    expect(Array.from(d.get(ch.key))).toEqual([1]);
  });
});
