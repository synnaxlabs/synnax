// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, test } from "vitest";

import { Channel } from "../channel";
import { newClient } from "../setupspecs";
import { DataType, Rate, TimeSpan, TimeStamp } from "../telem";
import { randomTypedArray } from "../util/telem";

const client = newClient();

const newChannel = async (): Promise<Channel> => {
  return await client.channel.create({
    name: "test",
    nodeId: 1,
    rate: Rate.hz(1),
    dataType: DataType.FLOAT64,
  });
};

describe("Writer", () => {
  describe("TypedWriter", () => {
    test("basic write", async () => {
      const ch = await newChannel();
      const writer = await client.data.newWriter(0, [ch.key]);
      try {
        await writer.write({ [ch.key]: randomTypedArray(10, ch.dataType) });
        await writer.commit();
      } finally {
        await writer.close();
      }
      expect(true).toBeTruthy();
    });
  });
  describe("Client", () => {
    test("Client - basic write", async () => {
      const ch = await newChannel();
      const data = randomTypedArray(10, ch.dataType);
      await client.data.write(ch.key, TimeStamp.seconds(1), data);
      await client.data.read(ch.key, TimeSpan.ZERO, TimeSpan.seconds(10000000));
      expect(data.length).toEqual(10);
    });
  });
});
