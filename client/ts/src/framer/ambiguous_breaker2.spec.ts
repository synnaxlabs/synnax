// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, id, Series, TimeStamp } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { type channel } from "@/channel";
import { NotFoundError, ValidationError } from "@/errors";
import {
  createProxiedTestClient,
  createTestClient,
  FAST_RETRY,
  newIndexedPair,
} from "@/testutil";

const client = createTestClient();

const virtual = (name: string): channel.New => ({
  name,
  dataType: DataType.FLOAT64,
  virtual: true,
});

const quiet = (): void => {};

describe("breaker2 record keys, NFC names, and the store", () => {
  describe("record columns of digits", () => {
    it("should refuse to set authority by a record key that names a key and a channel", async () => {
      const [index, data] = await newIndexedPair(client);
      await client.channels.create({
        name: String(data.key),
        dataType: DataType.FLOAT64,
        virtual: true,
      });
      const named = await client.channels.retrieve(String(data.key));
      const writer = await client.openWriter({
        start: TimeStamp.seconds(1),
        channels: [index.key, data.key, named.key],
      });
      try {
        await expect(writer.setAuthority({ [String(data.key)]: 100 })).rejects.toThrow(
          ValidationError,
        );
      } finally {
        await writer.close();
      }
    });

    it("should refuse a record key of digits past the uint32 limit as not found", async () => {
      const ch = await client.channels.create(virtual(id.create()));
      const writer = await client.openWriter({
        start: TimeStamp.now(),
        channels: [ch.key],
      });
      try {
        await expect(writer.write({ "99999999999": [1] })).rejects.toThrow(
          NotFoundError,
        );
      } finally {
        await writer.close();
      }
    });

    it("should write a record column of digits past the uint32 limit to the channel the writer holds by that name", async () => {
      const ch = await client.channels.create(virtual(`9${Date.now()}`));
      const writer = await client.openWriter({
        start: TimeStamp.now(),
        channels: [ch.key],
      });
      try {
        await writer.write({ [ch.name]: new Series([1]) });
      } finally {
        await writer.close();
      }
    });
  });

  describe("names in another normal form", () => {
    const create = async (): Promise<[channel.Channel, string]> => {
      const nfc = `caf${String.fromCharCode(0xe9)}_${id.create()}`;
      const ch = await client.channels.create(virtual(nfc));
      const nfd = nfc.normalize("NFD");
      expect(nfd).not.toEqual(nfc);
      return [ch, nfd];
    };

    it("should retrieve a channel by the NFD form of its name", async () => {
      const [ch, nfd] = await create();
      expect((await client.channels.retrieve(nfd)).key).toEqual(ch.key);
    });

    it("should open a writer on a channel by the NFD form of its name", async () => {
      const [ch, nfd] = await create();
      const writer = await client.openWriter({
        start: TimeStamp.now(),
        channels: [nfd],
      });
      try {
        await writer.write(nfd, [1]);
        expect(ch.name).not.toEqual(nfd);
      } finally {
        await writer.close();
      }
    });
  });

  describe("cached keys with the Core unreachable", () => {
    it("should retrieve a list of one cached key from the store", async () => {
      const { proxy, client: local } = await createProxiedTestClient({
        retry: FAST_RETRY,
        onRetry: quiet,
      });
      const ch = await local.channels.create(virtual(id.create()));
      await proxy.sever();
      const got = await local.channels.retrieve([ch.key]);
      expect(got.map((c) => c.key)).toEqual([ch.key]);
    }, 30000);

    it("should retrieve a list of two cached keys from the store", async () => {
      const { proxy, client: local } = await createProxiedTestClient({
        retry: FAST_RETRY,
        onRetry: quiet,
      });
      const a = await local.channels.create(virtual(id.create()));
      const b = await local.channels.create(virtual(id.create()));
      await proxy.sever();
      const got = await local.channels.retrieve([a.key, b.key]);
      expect(got.map((c) => c.key)).toEqual([a.key, b.key]);
    }, 30000);
  });
});
