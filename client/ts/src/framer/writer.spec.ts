// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, Rate, TimeRange, TimeStamp } from "@synnaxlabs/x";
import { describe, expect, test } from "vitest";

import { type channel } from "@/channel";
import { WriterMode } from "@/framer/writer";
import { newClient } from "@/setupspecs";
import { randomSeries } from "@/util/telem";

const client = newClient();

const newChannel = async (): Promise<channel.Channel> =>
  await client.channels.create({
    name: "test",
    leaseholder: 1,
    rate: Rate.hz(1),
    dataType: DataType.FLOAT64,
  });

describe("Writer", () => {
  describe("Writer", () => {
    test("basic write", async () => {
      const ch = await newChannel();
      const writer = await client.telem.newWriter({ start: 0, channels: ch.key });
      try {
        await writer.write(ch.key, randomSeries(10, ch.dataType));
        await writer.commit();
      } finally {
        await writer.close();
      }
      expect(true).toBeTruthy();
    });
    test("write to unknown channel key", async () => {
      const ch = await newChannel();
      const writer = await client.telem.newWriter({ start: 0, channels: ch.key });
      await expect(
        writer.write("billy bob", randomSeries(10, DataType.FLOAT64)),
      ).rejects.toThrow("Channel billy bob not found");
      await writer.close();
    });
    test("stream when mode is set ot persist only", async () => {
      const ch = await newChannel();
      const stream = await client.telem.newStreamer(ch.key);
      const writer = await client.telem.newWriter({
        start: 0,
        channels: ch.key,
        mode: WriterMode.PersistOnly,
      });
      try {
        await writer.write(ch.key, randomSeries(10, ch.dataType));
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
  });
  describe("Client", () => {
    test("Client - basic write", async () => {
      const ch = await newChannel();
      const data = randomSeries(10, ch.dataType);
      await client.telem.write(ch.key, TimeStamp.seconds(1), data);
      const res = await client.telem.read(TimeRange.MAX, ch.key);
      expect(res.length).toEqual(data.length);
      expect(res.data).toEqual(data);
    });
  });
});
