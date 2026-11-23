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
import { ValidationError } from "@/errors";
import { Frame } from "@/framer/frame";
import {
  createTestClient,
  newIndexedPair,
  secondsLinspace,
  spyOnSend,
} from "@/testutil";

const client = createTestClient();

const float = (data: number[]): Series =>
  new Series({ data, dataType: DataType.FLOAT64 });

/** @returns, as JSON, each channel retrieve that send saw for any of keys. */
const retrievesOf = (
  send: ReturnType<typeof spyOnSend>,
  keys: channel.Key[],
): string[] => {
  const mention = new RegExp(`\\b(${keys.join("|")})\\b`);
  return send.mock.calls
    .filter(([target]) => target === "/channel/retrieve")
    .map(([, req]) => JSON.stringify(req))
    .filter((req) => mention.test(req));
};

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

describe("Framer lookups", () => {
  describe("client.write record", () => {
    it("should write a record keyed by a channel key and a channel name", async () => {
      const [index, data] = await newIndexedPair(client);
      await client.write(TimeStamp.seconds(1), {
        [index.key]: secondsLinspace(1, 3),
        [data.name]: [1, 2, 3],
      });
      const fr = await client.read(TimeRange.MAX, [data.key]);
      expect(Array.from(fr.get(data.key).series[0])).toEqual([1, 2, 3]);
    });
  });

  describe("paths", () => {
    it("should open a writer on a channel given by its path", async () => {
      const [ch, path] = await createInGroup({
        dataType: DataType.FLOAT64,
        virtual: true,
      });
      expect((await client.channels.retrieve(path)).key).toEqual(ch.key);
      const writer = await client.openWriter({
        start: TimeStamp.now(),
        channels: [path],
      });
      await writer.close();
    });

    it("should open a streamer on a channel given by its path", async () => {
      const [ch, path] = await createInGroup({
        dataType: DataType.FLOAT64,
        virtual: true,
      });
      expect((await client.channels.retrieve(path)).key).toEqual(ch.key);
      const streamer = await client.openStreamer([path]);
      expect(streamer.keys).toEqual([ch.key]);
      streamer.close();
    });

    it("should read a channel given by its path", async () => {
      const [index, path] = await createInGroup({
        dataType: DataType.TIMESTAMP,
        isIndex: true,
      });
      await client.write(TimeStamp.seconds(1), index.key, secondsLinspace(1, 3));
      const series = await client.read(TimeRange.MAX, path);
      expect(series.length).toEqual(3);
    });

    it("should set authority on one channel given by its path", async () => {
      const [ch, path] = await createInGroup({
        dataType: DataType.FLOAT64,
        virtual: true,
      });
      const writer = await client.openWriter({
        start: TimeStamp.now(),
        channels: [ch.key],
      });
      try {
        await writer.setAuthority(path, 100);
        await writer.write(path, [1, 2, 3]);
      } finally {
        await writer.close();
      }
    });
  });

  describe("digit names beside keys", () => {
    it("should stream the channel named by another channel's key", async () => {
      const other = await client.channels.create({
        name: id.create(),
        dataType: DataType.FLOAT64,
        virtual: true,
      });
      const named = await client.channels.create({
        name: String(other.key),
        dataType: DataType.FLOAT64,
        virtual: true,
      });
      const streamer = await client.openStreamer([named.name]);
      const writer = await client.openWriter({
        start: TimeStamp.now(),
        channels: [other.key, named.key],
      });
      try {
        await writer.write([other.key, named.key], [[1], [2]]);
        const fr = await streamer.read();
        expect(fr.columns).toEqual([named.name]);
        expect(Array.from(fr.get(named.name).series[0])).toEqual([2]);
      } finally {
        await writer.close();
        streamer.close();
      }
    });

    it("should set authority by a record that mixes a digit name and a key", async () => {
      const [index, data] = await newIndexedPair(client);
      const name = `${Date.now()}${Math.floor(Math.random() * 1e6)}`;
      await client.channels.rename(data.key, name);
      const start = TimeStamp.seconds(5);
      const low = await client.openWriter({
        start,
        channels: [index.key, data.key],
        authorities: 10,
      });
      const high = await client.openWriter({
        start,
        channels: [index.key, data.key],
        authorities: 20,
      });
      try {
        await low.setAuthority({ [name]: 100, [index.key]: 100 });
        await low.write({
          [index.key]: secondsLinspace(5, 3),
          [data.key]: [1, 2, 3],
        });
      } finally {
        await low.close();
        await high.close();
      }
      expect((await data.read(TimeRange.MAX)).length).toEqual(3);
    });
  });

  describe("key-only lookups and the record store", () => {
    it("should open a writer on one cached key with no channel retrieve", async () => {
      const local = createTestClient();
      await local.connect();
      const ch = await local.channels.create({
        name: id.create(),
        dataType: DataType.FLOAT64,
        virtual: true,
      });
      const send = spyOnSend(local);
      try {
        const writer = await local.openWriter({
          start: TimeStamp.now(),
          channels: [ch.key],
        });
        await writer.close();
        expect(retrievesOf(send, [ch.key])).toEqual([]);
      } finally {
        send.mockRestore();
      }
    });

    it("should open a writer on two cached keys with no channel retrieve", async () => {
      const local = createTestClient();
      await local.connect();
      const a = await local.channels.create({
        name: id.create(),
        dataType: DataType.FLOAT64,
        virtual: true,
      });
      const b = await local.channels.create({
        name: id.create(),
        dataType: DataType.FLOAT64,
        virtual: true,
      });
      const send = spyOnSend(local);
      try {
        const writer = await local.openWriter({
          start: TimeStamp.now(),
          channels: [a.key, b.key],
        });
        await writer.close();
        expect(retrievesOf(send, [a.key, b.key])).toEqual([]);
      } finally {
        send.mockRestore();
      }
    });
  });

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
    it("should refuse a read that names one channel by its name and its path", async () => {
      const [index, path] = await createInGroup({
        dataType: DataType.TIMESTAMP,
        isIndex: true,
      });
      await expect(client.read(TimeRange.MAX, [index.name, path])).rejects.toThrow(
        new ValidationError(`"${index.name}" and "${path}" name one channel`),
      );
    });

    it("should refuse a streamer that names one channel by its name and its path", async () => {
      const [ch, path] = await createInGroup({
        dataType: DataType.FLOAT64,
        virtual: true,
      });
      await expect(client.openStreamer([ch.name, path])).rejects.toThrow(
        new ValidationError(`"${ch.name}" and "${path}" name one channel`),
      );
    });

    it("should refuse a writer that names one channel by its key and its name", async () => {
      const [index, data] = await newIndexedPair(client);
      await expect(
        client.openWriter({
          start: TimeStamp.seconds(1),
          channels: [index.key, data.key, data.name],
          authorities: [100, 100, 100],
        }),
      ).rejects.toThrow(
        new ValidationError(`${data.key} and "${data.name}" name one channel`),
      );
    });

    it("should read a channel that a list names twice by one name", async () => {
      const [index] = await newIndexedPair(client);
      await client.write(TimeStamp.seconds(1), index.key, secondsLinspace(1, 3));
      const fr = await client.read(TimeRange.MAX, [index.name, index.name]);
      expect(fr.get(index.name).length).toEqual(3);
    });
  });

  describe("one retrieve per open", () => {
    it("should resolve the names and the paths of a writer in one retrieve", async () => {
      const [, aPath] = await createInGroup({
        dataType: DataType.FLOAT64,
        virtual: true,
      });
      const [, bPath] = await createInGroup({
        dataType: DataType.FLOAT64,
        virtual: true,
      });
      const named = await client.channels.create({
        name: id.create(),
        dataType: DataType.FLOAT64,
        virtual: true,
      });
      const send = spyOnSend(client);
      try {
        const writer = await client.openWriter({
          start: TimeStamp.now(),
          channels: [aPath, bPath, named.name],
        });
        await writer.close();
        const lookups = [aPath, bPath, named.name];
        const mine = send.mock.calls
          .filter(([target]) => target === "/channel/retrieve")
          .map(([, req]) => JSON.stringify(req))
          .filter((req) => lookups.some((l) => req.includes(`"${l}"`)));
        expect(mine).toHaveLength(1);
        for (const l of lookups) expect(mine[0]).toContain(`"${l}"`);
      } finally {
        send.mockRestore();
      }
    });
  });
});
