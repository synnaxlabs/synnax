// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, id, Series, TimeRange, TimeStamp } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { type channel } from "@/channel";
import { Frame } from "@/framer/frame";
import { createTestClient, newIndexedPair, secondsLinspace } from "@/testutil";

const client = createTestClient();

const createInGroup = async (
  props: Omit<channel.New, "name" | "group">,
): Promise<[channel.Channel, string]> => {
  const g = await client.groups.create({ name: id.create(), scope: "channel" });
  const ch = await client.channels.create({
    ...props,
    name: id.create(),
    group: g.key,
  });
  return [ch, `${g.name}.${ch.name}`];
};

const float = (data: number[]): Series =>
  new Series({ data, dataType: DataType.FLOAT64 });

describe("Framer lookups, breaker 3", () => {
  describe("a frame keyed by the path of a held channel", () => {
    it("should write a Frame keyed by the path of a channel the writer holds by key", async () => {
      const [ch, path] = await createInGroup({
        dataType: DataType.FLOAT64,
        virtual: true,
      });
      const writer = await client.openWriter({
        start: TimeStamp.now(),
        channels: [ch.key],
      });
      try {
        await writer.write(path, [1, 2, 3]);
        await writer.write({ [path]: [1, 2, 3] });
        await writer.write(new Frame({ [path]: float([1, 2, 3]) }));
      } finally {
        await writer.close();
      }
    });

    it("should write a Map keyed by the path of a channel the writer holds by name", async () => {
      const [ch, path] = await createInGroup({
        dataType: DataType.FLOAT64,
        virtual: true,
      });
      const writer = await client.openWriter({
        start: TimeStamp.now(),
        channels: [ch.name],
      });
      try {
        await writer.write(new Map([[path, float([1, 2, 3])]]));
      } finally {
        await writer.close();
      }
    });
  });

  describe("one channel looked up twice", () => {
    it("should key a read by both the name and the path that looked up one channel", async () => {
      const [index, path] = await createInGroup({
        dataType: DataType.TIMESTAMP,
        isIndex: true,
      });
      await client.write(TimeStamp.seconds(1), index.key, secondsLinspace(1, 3));
      const fr = await client.read(TimeRange.MAX, [index.name, path]);
      expect(fr.get(path).length).toEqual(3);
      expect(fr.get(index.name).length).toEqual(3);
    });

    it("should key streamed frames by both the name and the path of one channel", async () => {
      const [ch, path] = await createInGroup({
        dataType: DataType.FLOAT64,
        virtual: true,
      });
      const streamer = await client.openStreamer([ch.name, path]);
      const writer = await client.openWriter({
        start: TimeStamp.now(),
        channels: [ch.key],
      });
      try {
        await writer.write(ch.key, [7]);
        const fr = await streamer.read();
        expect(Array.from(fr.get(path).series[0])).toEqual([7]);
        expect(Array.from(fr.get(ch.name).series[0])).toEqual([7]);
      } finally {
        await writer.close();
        streamer.close();
      }
    });

    it("should open a writer on a channel given by its key and its name with one authority each", async () => {
      const [index, data] = await newIndexedPair(client);
      const writer = await client.openWriter({
        start: TimeStamp.seconds(1),
        channels: [index.key, data.key, data.name],
        authorities: [100, 100, 100],
      });
      await writer.close();
    });
  });
});
