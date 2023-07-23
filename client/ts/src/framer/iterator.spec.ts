// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, Rate, TimeRange, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { describe, test, expect } from "vitest";

import { Channel } from "../channel";
import { newClient } from "../setupspecs";
import { randomTypedArray } from "../util/telem";

const client = newClient();

const newChannel = async (): Promise<Channel> => {
  return await client.channels.create({
    name: "test",
    leaseholder: 1,
    rate: Rate.hz(25),
    dataType: DataType.FLOAT64,
  });
};

describe("Iterator", () => {
  test("happy path", async () => {
    const ch = await newChannel();
    const writer = await client.telem.newWriter(TimeStamp.SECOND, ch.key);
    const data = randomTypedArray(25, ch.dataType);
    try {
      await writer.write(ch.key, data);
      await writer.write(ch.key, data);
      await writer.write(ch.key, data);
      await writer.commit();
    } finally {
      await writer.close();
    }

    const iter = await client.telem.newIterator(
      new TimeRange(TimeSpan.ZERO, TimeSpan.seconds(4)),
      [ch.key]
    );

    try {
      expect(await iter.seekFirst()).toBeTruthy();
      let c = 0;
      while (await iter.next(TimeSpan.seconds(1))) {
        c++;
        expect(iter.value.get(ch.key)[0]).toHaveLength(25);
      }
      expect(c).toEqual(3);
    } finally {
      await iter.close();
    }
  });
});
