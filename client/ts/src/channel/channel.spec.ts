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

import { Channel } from "@/channel/client";
import { NotFoundError, QueryError } from "@/errors";
import { newClient } from "@/setupspecs";

const client = newClient();

describe("Channel", () => {
  describe("create", () => {
    test("create one", async () => {
      const channel = await client.channels.create({
        name: "test",
        leaseholder: 1,
        rate: Rate.hz(1),
        dataType: DataType.FLOAT32,
      });
      expect(channel.name, "test").toEqual("test");
      expect(channel.leaseholder).toEqual(1);
      expect(channel.rate).toEqual(Rate.hz(1));
      expect(channel.dataType).toEqual(DataType.FLOAT32);
    });

    test("create calculated", async () => {
      const chOne = new Channel({
        name: "test",
        virtual: true,
        dataType: DataType.FLOAT32,
      });
      client.channels.create(chOne);
      let calculatedCH = new Channel({
        name: "test2",
        virtual: true,
        dataType: DataType.FLOAT32,
        expression: "test * 2",
        requires: [chOne.key],
      });
      calculatedCH = await client.channels.create(calculatedCH);
      expect(calculatedCH.key).not.toEqual(0);
      expect(calculatedCH.virtual).toEqual(true);
      expect(calculatedCH.expression).toEqual("test * 2");
    });

    test("create index and indexed pair", async () => {
      const one = await client.channels.create({
        name: "Time",
        isIndex: true,
        dataType: DataType.TIMESTAMP,
      });
      expect(one.key).not.toEqual(0);
      const two = await client.channels.create({
        name: "test",
        index: one.key,
        dataType: DataType.FLOAT32,
      });
      expect(two.key).not.toEqual(0);
    });

    test("create many", async () => {
      const channels = await client.channels.create([
        {
          name: "test1",
          leaseholder: 1,
          rate: Rate.hz(1),
          dataType: DataType.FLOAT32,
        },
        {
          name: "test2",
          leaseholder: 1,
          rate: Rate.hz(1),
          dataType: DataType.FLOAT32,
        },
      ]);
      expect(channels.length).toEqual(2);
      expect(channels[0].name).toEqual("test1");
      expect(channels[1].name).toEqual("test2");
    });

    test("create instances of channels", async () => {
      const timeIndexChannel = await client.channels.create({
        name: "time",
        dataType: DataType.TIMESTAMP,
        isIndex: true,
      });

      const sensorOne = new Channel({
        name: "sensor_one",
        dataType: DataType.FLOAT32,
        index: timeIndexChannel.key,
      });

      const sensorTwo = new Channel({
        name: "sensor_two",
        dataType: DataType.FLOAT32,
        index: timeIndexChannel.key,
      });

      const sensorThree = new Channel({
        name: "sensor_three",
        dataType: DataType.FLOAT32,
        index: timeIndexChannel.key,
      });
      await client.channels.create([sensorOne, sensorTwo, sensorThree]);
    });

    describe("virtual", () => {
      it("should create a virtual channel", async () => {
        const channel = await client.channels.create({
          name: "test",
          dataType: DataType.JSON,
          virtual: true,
        });
        expect(channel.virtual).toEqual(true);
        const retrieved = await client.channels.retrieve(channel.key);
        expect(retrieved.virtual).toBeTruthy();
      });
    });

    describe("retrieveIfNameExists", () => {
      it("should retrieve the existing channel when it exists", async () => {
        const name = `test-${Math.random()}-${TimeStamp.now().valueOf()}`;
        const channel = await client.channels.create({
          name,
          leaseholder: 1,
          rate: Rate.hz(1),
          dataType: DataType.FLOAT32,
        });
        const channelTwo = await client.channels.create(
          {
            name,
            leaseholder: 1,
            rate: Rate.hz(1),
            dataType: DataType.FLOAT32,
          },
          { retrieveIfNameExists: true },
        );
        expect(channelTwo.key).toEqual(channel.key);
      });
      it("should create a new channel when it does not exist", async () => {
        const name = `test-${Math.random()}-${TimeStamp.now().valueOf()}`;
        const channel = await client.channels.create({
          name,
          leaseholder: 1,
          rate: Rate.hz(1),
          dataType: DataType.FLOAT32,
        });
        const channelTwo = await client.channels.create(
          {
            name: `${name}-2`,
            leaseholder: 1,
            rate: Rate.hz(1),
            dataType: DataType.FLOAT32,
          },
          { retrieveIfNameExists: true },
        );
        expect(channelTwo.key).not.toEqual(channel.key);
      });
      it("should retrieve and create the correct channels when creating many", async () => {
        const name = `test-${Math.random()}-${TimeStamp.now().valueOf()}`;
        const channel = await client.channels.create({
          name,
          leaseholder: 1,
          rate: Rate.hz(1),
          dataType: DataType.FLOAT32,
        });
        const channelTwo = await client.channels.create(
          [
            {
              name,
              leaseholder: 1,
              rate: Rate.hz(1),
              dataType: DataType.FLOAT32,
            },
            {
              name: `${name}-2`,
              leaseholder: 1,
              rate: Rate.hz(1),
              dataType: DataType.FLOAT32,
            },
          ],
          { retrieveIfNameExists: true },
        );
        expect(channelTwo.length).toEqual(2);
        expect(channelTwo[0].key).toEqual(channel.key);
        expect(channelTwo[1].key).not.toEqual(channel.key);
      });
    });
  });

  describe("retrieve", () => {
    test("retrieve by key", async () => {
      const channel = await client.channels.create({
        name: "test",
        leaseholder: 1,
        rate: Rate.hz(1),
        dataType: DataType.FLOAT32,
      });
      const retrieved = await client.channels.retrieve(channel.key);
      expect(retrieved.name).toEqual("test");
      expect(retrieved.leaseholder).toEqual(1);
      expect(retrieved.rate).toEqual(Rate.hz(1));
      expect(retrieved.dataType).toEqual(DataType.FLOAT32);
    });
    test("retrieve by key - not found", async () => {
      await expect(
        async () => await client.channels.retrieve("1-1000"),
      ).rejects.toThrow(QueryError);
    });
    test("retrieve by name", async () => {
      const retrieved = await client.channels.retrieve(["test"]);
      expect(retrieved.length).toBeGreaterThan(0);
      retrieved.forEach((ch) => expect(ch.name).toEqual("test"));
    });
    test("retrieve by key - not found", async () => {
      await expect(
        async () => await client.channels.retrieve("1-1000"),
      ).rejects.toThrow(NotFoundError);
    });
    test("retrieve by name", async () => {
      const retrieved = await client.channels.retrieve(["test"]);
      expect(retrieved.length).toBeGreaterThan(0);
      retrieved.forEach((ch) => expect(ch.name).toEqual("test"));
    });
  });

  describe("delete", async () => {
    test("delete by key", async () => {
      const channel = await client.channels.create({
        name: "test",
        leaseholder: 1,
        rate: Rate.hz(1),
        dataType: DataType.FLOAT32,
      });
      await client.channels.delete(channel.key);
      await expect(
        async () => await client.channels.retrieve(channel.key),
      ).rejects.toThrow(QueryError);
    });
    test("delete by name", async () => {
      const channel = await client.channels.create({
        name: "test",
        leaseholder: 1,
        rate: Rate.hz(1),
        dataType: DataType.FLOAT32,
      });
      await client.channels.delete(["test"]);
      await expect(
        async () => await client.channels.retrieve(channel.key),
      ).rejects.toThrow(QueryError);
    });
  });
  describe("rename", async () => {
    test("single rename", async () => {
      const channel = await client.channels.create({
        name: "test",
        leaseholder: 1,
        rate: Rate.hz(1),
        dataType: DataType.FLOAT32,
      });
      await client.channels.rename(channel.key, "test2");
      const renamed = await client.channels.retrieve(channel.key);
      expect(renamed.name).toEqual("test2");
    });
    test("multiple rename", async () => {
      const channels = await client.channels.create([
        {
          name: "test1",
          leaseholder: 1,
          rate: Rate.hz(1),
          dataType: DataType.FLOAT32,
        },
        {
          name: "test2",
          leaseholder: 1,
          rate: Rate.hz(1),
          dataType: DataType.FLOAT32,
        },
      ]);
      // Retrieve channels here to ensure we check for cache invalidation
      const initial = await client.channels.retrieve(channels.map((c) => c.key));
      expect(initial[0].name).toEqual("test1");
      expect(initial[1].name).toEqual("test2");
      await client.channels.rename(
        channels.map((c) => c.key),
        ["test3", "test4"],
      );
      const renamed = await client.channels.retrieve(channels.map((c) => c.key));
      expect(renamed[0].name).toEqual("test3");
      expect(renamed[1].name).toEqual("test4");
    });
  });

  describe("update", () => {
    test("update virtual channel expression", async () => {
      const channel = await client.channels.create({
        name: "virtual-calc",
        dataType: DataType.FLOAT32,
        virtual: true,
        expression: "return np.array([])",
      });

      const updated = await client.channels.create({
        key: channel.key,
        name: channel.name,
        dataType: channel.dataType,
        virtual: true,
        expression: "return np.array([1, 2, 3])",
      });

      const channelsWithName = await client.channels.retrieve(["virtual-calc"]);
      expect(channelsWithName.length).toEqual(1);

      expect(updated.expression).toEqual("return np.array([1, 2, 3])");

      const retrieved = await client.channels.retrieve(channel.key);
      expect(retrieved.expression).toEqual("return np.array([1, 2, 3])");
    });

    test("update virtual channel name", async () => {
      const channel = await client.channels.create({
        name: "virtual-calc",
        dataType: DataType.FLOAT32,
        virtual: true,
        expression: "return np.array([])",
      });

      const updated = await client.channels.create({
        key: channel.key,
        name: "new-name",
        dataType: channel.dataType,
        virtual: true,
        expression: channel.expression,
      });
      expect(updated.name).toEqual("new-name");

      const retrieved = await client.channels.retrieve(channel.key);
      expect(retrieved.name).toEqual("new-name");
    });

    test("should not allow updates to non-virtual channels", async () => {
      const channel = await client.channels.create({
        name: "regular-channel",
        leaseholder: 1,
        rate: Rate.hz(1),
        dataType: DataType.FLOAT32,
      });

      const _updated = await client.channels.create({
        key: channel.key,
        name: "new-name",
        leaseholder: channel.leaseholder,
        rate: channel.rate,
        dataType: channel.dataType,
      });

      const retrieved = await client.channels.retrieve(channel.key);
      expect(retrieved.name).toEqual("regular-channel");
    });
  });
});
