// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, Rate } from "@synnaxlabs/x";
import { describe, test, expect } from "vitest";

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
      NotFoundError,
    );
  });

  test("retrieve by name", async () => {
    const retrieved = await client.channels.retrieve(["test"]);
    expect(retrieved.length).toBeGreaterThan(0);
    retrieved.forEach((ch) => expect(ch.name).toEqual("test"));
  });
});
