import { test, expect } from "vitest";

import { newClient } from "../../setupspecs";
import { QueryError } from "../errors";
import { DataType, Rate } from "../telem";

const client = newClient();

test("Channel - create", async () => {
  const channel = await client.channel.create({
    name: "test",
    nodeId: 1,
    rate: Rate.Hz(1),
    dataType: DataType.Float32,
  });
  expect(channel.name, "test").toEqual("test");
  expect(channel.nodeId).toEqual(1);
  expect(channel.rate).toEqual(Rate.Hz(1));
  expect(channel.dataType).toEqual(DataType.Float32);
});

test("Channel - retrieve by key", async () => {
  const channel = await client.channel.create({
    name: "test",
    nodeId: 1,
    rate: Rate.Hz(1),
    dataType: DataType.Float32,
  });
  const retrieved = await client.channel.retrieve({ key: channel.key });
  expect(retrieved.name).toEqual("test");
  expect(retrieved.nodeId).toEqual(1);
  expect(retrieved.rate).toEqual(Rate.Hz(1));
  expect(retrieved.dataType).toEqual(DataType.Float32);
});

test("Channel - retrieve by key - not found", async () => {
  await expect(async () => {
    await client.channel.retrieve({ key: "1-1000" });
  }).rejects.toThrow(QueryError);
});

test("Channel - retrieve by node id", async () => {
  const retrieved = await client.channel.filter({ nodeId: 1 });
  expect(retrieved.length).toBeGreaterThan(0);
  retrieved.forEach((ch) => expect(ch.nodeId).toEqual(1));
});

test("Channel - retrieve by name", async () => {
  const retrieved = await client.channel.filter({ names: ["test"] });
  expect(retrieved.length).toBeGreaterThan(0);
  retrieved.forEach((ch) => expect(ch.name).toEqual("test"));
});
