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
import { NotFoundError, ValidationError } from "@/errors";
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

const drain = async (stream: ReadableStream<Uint8Array>): Promise<string> => {
  const reader = stream.getReader();
  const decoder = new TextDecoder();
  let out = "";
  while (true) {
    const { done, value } = await reader.read();
    if (done) break;
    out += decoder.decode(value);
  }
  return out;
};

const channelRetrieves = (send: ReturnType<typeof spyOnSend>, mention: string[]) =>
  send.mock.calls
    .filter(([target]) => target === "/channel/retrieve")
    .map(([, req]) => JSON.stringify(req))
    .filter((req) => mention.some((m) => req.includes(m)));

describe("Framer lookups, breaker 4", () => {
  describe("a file read names one channel twice", () => {
    it("should refuse a CSV read that names one channel by its name and its path", async () => {
      const [index, path] = await createInGroup({
        dataType: DataType.TIMESTAMP,
        isIndex: true,
      });
      await client.write(TimeStamp.seconds(1), index.key, secondsLinspace(1, 3));
      await expect(
        client.readCSV(TimeRange.MAX, [index.name, path]).then(drain),
      ).rejects.toThrow(
        new ValidationError(`"${index.name}" and "${path}" name one channel`),
      );
    });

    it("should refuse a TDMS read that names one channel by its key and its name", async () => {
      const [index] = await newIndexedPair(client);
      await client.write(TimeStamp.seconds(1), index.key, secondsLinspace(1, 3));
      await expect(
        client.readTDMS(TimeRange.MAX, [index.key, index.name]).then(drain),
      ).rejects.toThrow(
        new ValidationError(`${index.key} and "${index.name}" name one channel`),
      );
    });

    it("should refuse a CSV read of a name that finds no channel", async () => {
      const [index] = await newIndexedPair(client);
      await client.write(TimeStamp.seconds(1), index.key, secondsLinspace(1, 3));
      const missing = id.create();
      await expect(
        client.readCSV(TimeRange.MAX, [index.name, missing]).then(drain),
      ).rejects.toThrow(NotFoundError);
    });
  });

  describe("setAuthority names one channel twice", () => {
    it("should refuse a record that gives one channel two authorities by its key and its name", async () => {
      const [ch] = await createInGroup({ dataType: DataType.FLOAT64, virtual: true });
      const writer = await client.openWriter({
        start: TimeStamp.now(),
        channels: [ch.key],
        authorities: 50,
      });
      try {
        await expect(
          writer.setAuthority({ [ch.key]: 10, [ch.name]: 200 }),
        ).rejects.toThrow(ValidationError);
      } finally {
        await writer.close();
      }
    });
  });

  describe("one retrieve per lookup set", () => {
    it("should resolve the paths of a setAuthority record in one retrieve", async () => {
      const [a, aPath] = await createInGroup({
        dataType: DataType.FLOAT64,
        virtual: true,
      });
      const [b, bPath] = await createInGroup({
        dataType: DataType.FLOAT64,
        virtual: true,
      });
      const writer = await client.openWriter({
        start: TimeStamp.now(),
        channels: [a.key, b.key],
      });
      const send = spyOnSend(client);
      try {
        await writer.setAuthority({ [aPath]: 100, [bPath]: 100 });
        expect(channelRetrieves(send, [aPath, bPath])).toHaveLength(1);
      } finally {
        send.mockRestore();
        await writer.close();
      }
    });

    it("should resolve the paths of a record write in one retrieve", async () => {
      const [a, aPath] = await createInGroup({
        dataType: DataType.FLOAT64,
        virtual: true,
      });
      const [b, bPath] = await createInGroup({
        dataType: DataType.FLOAT64,
        virtual: true,
      });
      const writer = await client.openWriter({
        start: TimeStamp.now(),
        channels: [a.key, b.key],
      });
      const send = spyOnSend(client);
      try {
        await writer.write({ [aPath]: [1], [bPath]: [2] });
        expect(channelRetrieves(send, [aPath, bPath])).toHaveLength(1);
      } finally {
        send.mockRestore();
        await writer.close();
      }
    });
  });

  describe("NFC and NFD names", () => {
    it("should open a streamer on one channel named in NFC and in NFD", async () => {
      const nfc = `caf\u00e9_${id.create()}`;
      const ch = await client.channels.create({
        name: nfc,
        dataType: DataType.FLOAT64,
        virtual: true,
      });
      const nfd = nfc.normalize("NFD");
      expect(nfd).not.toEqual(nfc);
      const streamer = await client.openStreamer([nfc, nfd]);
      try {
        expect(streamer.keys).toEqual([ch.key]);
      } finally {
        streamer.close();
      }
    });
  });
});
