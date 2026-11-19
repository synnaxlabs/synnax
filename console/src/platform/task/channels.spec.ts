// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client/testutil";
import { uniqueName } from "@synnaxlabs/x/testutil";
import { describe, expect, it } from "vitest";

import { Task } from "@/platform/task";

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
        stored: { read, key: "src" },
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

    it("should number a new index whose name a field takes", async () => {
      const prefix = uniqueName("src");
      const read: Record<string, Task.ReadChannelProps> = {};
      const fields = [createField("f1", "/time")];
      await Task.configureReadChannels({
        client,
        stored: { read, key: "src" },
        fields,
        namePrefix: prefix,
        indexKey: "",
      });
      expect((await client.channels.retrieve(fields[0].channel)).name).toBe(
        `${prefix}_time`,
      );
      expect((await client.channels.retrieve(read.src.index)).name).toBe(
        `${prefix}_time_2`,
      );
    });

    it("should number a new channel whose name the Core holds", async () => {
      const prefix = uniqueName("src");
      await createIndex(`${prefix}_temperature`);
      await createIndex(`${prefix}_temperature_2`);
      const read: Record<string, Task.ReadChannelProps> = {};
      const fields = [createField("f1", "/temperature")];
      await Task.configureReadChannels({
        client,
        stored: { read, key: "src" },
        fields,
        namePrefix: prefix,
        indexKey: "",
      });
      expect((await client.channels.retrieve(fields[0].channel)).name).toBe(
        `${prefix}_temperature_3`,
      );
      expect((await client.channels.retrieve(read.src.index)).name).toBe(
        `${prefix}_time`,
      );
    });

    describe("replacement", () => {
      /** Configures a field whose new channel replaces old. */
      const replace = async (prefix: string, old: number) => {
        const fields = [{ ...createField("f1", "/mode", "string"), replaces: old }];
        await Task.configureReadChannels({
          client,
          fields,
          namePrefix: prefix,
          indexKey: "",
        });
        return await client.channels.retrieve(fields[0].channel);
      };

      it("should move a replaced channel to a numbered name and keep the plain name", async () => {
        const prefix = uniqueName("src");
        const old = await createIndex(`${prefix}_mode`);
        expect((await replace(prefix, old.key)).name).toBe(`${prefix}_mode`);
        expect((await client.channels.retrieve(old.key)).name).toBe(`${prefix}_mode_2`);
      });

      it("should leave a replaced channel that the user renamed", async () => {
        const prefix = uniqueName("src");
        const old = await createIndex(uniqueName("custom"));
        expect((await replace(prefix, old.key)).name).toBe(`${prefix}_mode`);
        expect((await client.channels.retrieve(old.key)).name).toBe(old.name);
      });

      it("should number the new channel when another channel holds its name", async () => {
        const prefix = uniqueName("src");
        const other = await createIndex(`${prefix}_mode`);
        const old = await createIndex(uniqueName("custom"));
        expect((await replace(prefix, old.key)).name).toBe(`${prefix}_mode_2`);
        expect((await client.channels.retrieve(other.key)).name).toBe(`${prefix}_mode`);
        expect((await client.channels.retrieve(old.key)).name).toBe(old.name);
      });
    });

    it("should create the index and the channels of a source that the device does not store", async () => {
      const prefix = uniqueName("src");
      const fields = [
        createField("f1", "/temperature"),
        createField("ts", "/ts", "timestamp"),
      ];
      await Task.configureReadChannels({
        client,
        fields,
        namePrefix: prefix,
        indexKey: "ts",
      });
      const index = await client.channels.retrieve(fields[1].channel);
      expect(index).toMatchObject({ name: `${prefix}_time`, isIndex: true });
      const data = await client.channels.retrieve(fields[0].channel);
      expect(data).toMatchObject({ name: `${prefix}_temperature`, index: index.key });
    });

    it("should create no channel or index for a disabled field", async () => {
      const read: Record<string, Task.ReadChannelProps> = {};
      const fields = [{ ...createField("f1", "/temperature"), disabled: true }];
      await Task.configureReadChannels({
        client,
        stored: { read, key: "src" },
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
        stored: { read, key: "src" },
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
        stored: { read, key: "src" },
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
        stored: { read, key: "src" },
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
      const ownIndex = await createIndex(uniqueName("own_time"));
      const own = await client.channels.create({
        name: uniqueName("own"),
        dataType: "float64",
        index: ownIndex.key,
      });
      const read: Record<string, Task.ReadChannelProps> = {};
      const fields = [{ ...createField("f1", "/own"), channel: own.key }];
      await Task.configureReadChannels({
        client,
        stored: { read, key: "src" },
        fields,
        namePrefix: uniqueName("src"),
        indexKey: "",
      });
      expect(read).toEqual({});
    });

    it("should skip the index when no field counts as indexed", async () => {
      const ownIndex = await createIndex(uniqueName("own_time"));
      const own = await client.channels.create({
        name: uniqueName("own"),
        dataType: "timestamp",
        index: ownIndex.key,
      });
      const read: Record<string, Task.ReadChannelProps> = {};
      const fields = [{ ...createField("f1", "/ts", "timestamp"), channel: own.key }];
      await Task.configureReadChannels({
        client,
        stored: { read, key: "src" },
        fields,
        namePrefix: uniqueName("src"),
        indexKey: "",
        indexed: () => false,
      });
      expect(read).toEqual({});
    });
  });

  describe("planReadChannels", () => {
    it("should plan the channels of a source without writing to the Core", async () => {
      const prefix = uniqueName("src");
      await createIndex(`${prefix}_temperature`);
      const props: Task.ReadChannelProps = { index: 0, channels: {} };
      const fields = [
        createField("f1", "/temperature"),
        createField("ts", "/ts", "timestamp"),
      ];
      const plan = await Task.planReadChannels({
        client,
        props,
        fields,
        namePrefix: prefix,
        indexKey: "ts",
      });
      expect(plan).toEqual({
        props: { index: 0, channels: {} },
        newIndex: `${prefix}_time`,
        indexField: "ts",
        channels: {},
        creates: [{ key: "f1", name: `${prefix}_temperature_2`, indexed: true }],
        renames: [],
      });
      expect(fields.map(({ channel }) => channel)).toEqual([0, 0]);
      expect(props).toEqual({ index: 0, channels: {} });
      expect(
        await client.channels.retrieve([`${prefix}_temperature_2`, `${prefix}_time`]),
      ).toEqual([]);
    });

    it("should plan to move a replaced channel off the name of its replacement", async () => {
      const prefix = uniqueName("src");
      const old = await createIndex(`${prefix}_mode`);
      const fields = [{ ...createField("f1", "/mode", "string"), replaces: old.key }];
      const plan = await Task.planReadChannels({
        client,
        fields,
        namePrefix: prefix,
        indexKey: "",
      });
      expect(plan.renames).toEqual([{ key: old.key, name: `${prefix}_mode_2` }]);
      expect(plan.creates).toEqual([
        { key: "f1", name: `${prefix}_mode`, indexed: false },
      ]);
      expect((await client.channels.retrieve(old.key)).name).toBe(`${prefix}_mode`);
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
        channel: 0,
        name,
        dataType: "uint8",
      });
      expect(write["/cmd"]).toBe(key);
      const ch = await client.channels.retrieve(key);
      expect(ch.name).toBe(name);
      expect((await client.channels.retrieve(ch.index)).name).toBe(`${name}_time`);
    });

    it("should create a virtual channel for a variable-density type", async () => {
      const write = {};
      const key = await Task.configureCommandChannel(client, write, {
        propertiesKey: "/cmd",
        channel: 0,
        name: uniqueName("cmd"),
        dataType: "string",
      });
      const ch = await client.channels.retrieve(key);
      expect(ch.virtual).toBe(true);
      expect(ch.index).toBe(0);
    });
  });
});
