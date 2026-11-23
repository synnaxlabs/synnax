// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, id, TimeRange, TimeStamp } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { type channel } from "@/channel";
import {
  createTestClient,
  newIndexedPair,
  secondsLinspace,
  spyOnSend,
} from "@/testutil";

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

describe("breaker2 framer lookups", () => {
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
        expect(
          send.mock.calls.filter(([target]) => target === "/channel/retrieve"),
        ).toHaveLength(0);
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
        expect(
          send.mock.calls.filter(([target]) => target === "/channel/retrieve"),
        ).toHaveLength(0);
      } finally {
        send.mockRestore();
      }
    });
  });
});
