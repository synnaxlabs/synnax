// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client/testutil";
import { describe, expect, it } from "vitest";

import { Task } from "@/platform/task";
import { uniqueName } from "@/testutil";

const client = createTestClient();
const MISSING_KEY = 4294967295;

const createIndex = async (name: string) =>
  await client.channels.create({ name, dataType: "timestamp", isIndex: true });

const createField = (
  key: string,
  pointer: string,
  dataType = "float64",
): Task.ReadChannelField => ({ key, pointer, channel: 0, name: "", dataType });

describe("channels", () => {
  describe("retrieveChannel", () => {
    it("should return null for key 0 and for a missing channel", async () => {
      expect(await Task.retrieveChannel(client, 0)).toBeNull();
      expect(await Task.retrieveChannel(client, MISSING_KEY)).toBeNull();
      expect(await Task.channelExists(client, MISSING_KEY)).toBe(false);
    });

    it("should return an existing channel", async () => {
      const ch = await client.channels.create({
        name: uniqueName("ch"),
        dataType: "string",
        virtual: true,
      });
      expect((await Task.retrieveChannel(client, ch.key))?.key).toBe(ch.key);
      expect(await Task.channelExists(client, ch.key)).toBe(true);
    });
  });

  describe("createChannelWithIndex", () => {
    it("should give a fixed-density channel its own index", async () => {
      const name = uniqueName("fixed");
      const ch = await Task.createChannelWithIndex(client, name, "float32");
      expect(ch.name).toBe(name);
      expect((await client.channels.retrieve(ch.index)).name).toBe(`${name}_time`);
    });

    it("should make a variable-density channel virtual", async () => {
      const ch = await Task.createChannelWithIndex(
        client,
        uniqueName("variable"),
        "string",
      );
      expect(ch.virtual).toBe(true);
      expect(ch.index).toBe(0);
    });
  });

  describe("configureReadChannels", () => {
    it("should create the index and a channel for each field", async () => {
      const prefix = uniqueName("src");
      const read: Record<string, Task.ReadChannelProps> = {};
      const fields = [
        createField("f1", "/temperature"),
        createField("f2", "/label", "string"),
        createField("ts", "/ts", "timestamp"),
      ];
      await Task.configureReadChannels({
        client,
        read,
        propertiesKey: "src",
        fields,
        namePrefix: prefix,
        indexKey: "ts",
      });
      const { index } = read.src;
      expect((await client.channels.retrieve(index)).name).toBe(`${prefix}_time`);
      const data = await client.channels.retrieve(fields[0].channel);
      expect(data.name).toBe(`${prefix}_temperature`);
      expect(data.index).toBe(index);
      expect((await client.channels.retrieve(fields[1].channel)).virtual).toBe(true);
      expect(fields[2].channel).toBe(index);
      expect(read.src.channels).toEqual({
        "/temperature": fields[0].channel,
        "/label": fields[1].channel,
      });
    });

    it("should create no channel or index for a disabled field", async () => {
      const read: Record<string, Task.ReadChannelProps> = {};
      const fields = [{ ...createField("f1", "/temperature"), disabled: true }];
      await Task.configureReadChannels({
        client,
        read,
        propertiesKey: "src",
        fields,
        namePrefix: uniqueName("src"),
        indexKey: "",
      });
      expect(read).toEqual({});
      expect(fields[0].channel).toBe(0);
    });

    it("should keep a field channel that exists and reuse a stored one", async () => {
      const index = await createIndex(uniqueName("idx"));
      const own = await client.channels.create({
        name: uniqueName("own"),
        dataType: "float64",
        index: index.key,
      });
      const stored = await client.channels.create({
        name: uniqueName("stored"),
        dataType: "float64",
        index: index.key,
      });
      const read: Record<string, Task.ReadChannelProps> = {
        src: { index: index.key, channels: { "/stored": stored.key } },
      };
      const fields = [
        { ...createField("f1", "/own"), channel: own.key },
        createField("f2", "/stored"),
      ];
      await Task.configureReadChannels({
        client,
        read,
        propertiesKey: "src",
        fields,
        namePrefix: uniqueName("src"),
        indexKey: "",
      });
      expect(read).toEqual({
        src: { index: index.key, channels: { "/stored": stored.key } },
      });
      expect(fields[0].channel).toBe(own.key);
      expect(fields[1].channel).toBe(stored.key);
    });

    it("should recover the index from a stored channel when the index is gone", async () => {
      const index = await createIndex(uniqueName("idx"));
      const stored = await client.channels.create({
        name: uniqueName("stored"),
        dataType: "float64",
        index: index.key,
      });
      const read: Record<string, Task.ReadChannelProps> = {
        src: { index: MISSING_KEY, channels: { "/stored": stored.key } },
      };
      const fields = [createField("f1", "/new")];
      await Task.configureReadChannels({
        client,
        read,
        propertiesKey: "src",
        fields,
        namePrefix: uniqueName("src"),
        indexKey: "",
      });
      expect(read.src.index).toBe(index.key);
      expect((await client.channels.retrieve(fields[0].channel)).index).toBe(index.key);
    });

    it("should take the index of a live field channel when the index is gone", async () => {
      const index = await createIndex(uniqueName("idx"));
      const own = await client.channels.create({
        name: uniqueName("own"),
        dataType: "float64",
        index: index.key,
      });
      const read: Record<string, Task.ReadChannelProps> = {
        src: { index: MISSING_KEY, channels: {} },
      };
      const fields = [
        { ...createField("f1", "/own"), channel: own.key },
        createField("f2", "/new"),
      ];
      await Task.configureReadChannels({
        client,
        read,
        propertiesKey: "src",
        fields,
        namePrefix: uniqueName("src"),
        indexKey: "",
      });
      expect(read).toEqual({
        src: { index: index.key, channels: { "/new": fields[1].channel } },
      });
      expect((await client.channels.retrieve(fields[1].channel)).index).toBe(index.key);
    });

    it("should leave the index alone when no field takes it or needs a new channel", async () => {
      const own = await Task.createChannelWithIndex(
        client,
        uniqueName("own"),
        "float64",
      );
      const read: Record<string, Task.ReadChannelProps> = {};
      const fields = [{ ...createField("f1", "/own"), channel: own.key }];
      await Task.configureReadChannels({
        client,
        read,
        propertiesKey: "src",
        fields,
        namePrefix: uniqueName("src"),
        indexKey: "",
      });
      expect(read).toEqual({});
    });

    it("should skip the index when no field counts as indexed", async () => {
      const own = await Task.createChannelWithIndex(
        client,
        uniqueName("own"),
        "timestamp",
      );
      const read: Record<string, Task.ReadChannelProps> = {};
      const fields = [{ ...createField("f1", "/ts", "timestamp"), channel: own.key }];
      await Task.configureReadChannels({
        client,
        read,
        propertiesKey: "src",
        fields,
        namePrefix: uniqueName("src"),
        indexKey: "",
        indexed: () => false,
      });
      expect(read).toEqual({});
    });
  });

  describe("configureCommandChannel", () => {
    it("should adopt the channel of the target and store it", async () => {
      const ch = await client.channels.create({
        name: uniqueName("cmd"),
        dataType: "string",
        virtual: true,
      });
      const write = {};
      const key = await Task.configureCommandChannel(client, write, {
        propertiesKey: "/cmd",
        channel: ch.key,
        name: "unused",
        dataType: "string",
      });
      expect(key).toBe(ch.key);
      expect(write).toEqual({ "/cmd": ch.key });
    });

    it("should reuse the stored channel of a target with none", async () => {
      const ch = await client.channels.create({
        name: uniqueName("cmd"),
        dataType: "string",
        virtual: true,
      });
      const write = { "/cmd": ch.key };
      expect(
        await Task.configureCommandChannel(client, write, {
          propertiesKey: "/cmd",
          channel: 0,
          name: "unused",
          dataType: "string",
        }),
      ).toBe(ch.key);
      expect(write).toEqual({ "/cmd": ch.key });
    });

    it("should create a channel when neither exists", async () => {
      const name = uniqueName("cmd");
      const write = { "/cmd": MISSING_KEY };
      const key = await Task.configureCommandChannel(client, write, {
        propertiesKey: "/cmd",
        channel: MISSING_KEY,
        name,
        dataType: "uint8",
      });
      expect(write["/cmd"]).toBe(key);
      const ch = await client.channels.retrieve(key);
      expect(ch.name).toBe(name);
      expect((await client.channels.retrieve(ch.index)).name).toBe(`${name}_time`);
    });
  });
});
