// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, Rate, TimeStamp } from "@synnaxlabs/x";
import { describe, test, expect, it } from "vitest";

import { Channel } from "@/channel/client";
import { QueryError } from "@/errors";
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

      const sensors = await client.channels.create([sensorOne, sensorTwo, sensorThree]);
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
    await expect(async () => await client.channels.retrieve("1-1000")).rejects.toThrow(
      QueryError,
    );
  });
  test("retrieve by name", async () => {
    const retrieved = await client.channels.retrieve(["test"]);
    expect(retrieved.length).toBeGreaterThan(0);
    retrieved.forEach((ch) => expect(ch.name).toEqual("test"));
  });
  test("retrieve by key - not found", async () => {
    await expect(async () => await client.channels.retrieve("1-1000")).rejects.toThrow(
      QueryError,
    );
  });
  test("retrieve by name", async () => {
    const retrieved = await client.channels.retrieve(["test"]);
    expect(retrieved.length).toBeGreaterThan(0);
    retrieved.forEach((ch) => expect(ch.name).toEqual("test"));
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
});
