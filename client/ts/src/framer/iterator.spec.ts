// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, Rate, TimeRange, TimeSpan, TimeStamp } from "@synnaxlabs/x/telem";
import { describe, test, expect } from "vitest";

import { type channel } from "@/channel";
import { newClient } from "@/setupspecs";
import { randomSeries } from "@/util/telem";

const client = newClient();

const newChannel = async (): Promise<channel.Channel> => {
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
    const writer = await client.telem.openWriter({
      start: TimeStamp.SECOND,
      channels: ch.key,
    });
    const data = randomSeries(25, ch.dataType);
    try {
      await writer.write(ch.key, data);
      await writer.write(ch.key, data);
      await writer.write(ch.key, data);
      await writer.commit();
    } finally {
      await writer.close();
    }

    const iter = await client.telem.openIterator(
      new TimeRange(TimeSpan.ZERO, TimeSpan.seconds(4)),
      [ch.key],
    );

    try {
      expect(await iter.seekFirst()).toBeTruthy();
      let c = 0;
      while (await iter.next(TimeSpan.seconds(1))) {
        c++;
        expect(iter.value.get(ch.key)).toHaveLength(25);
      }
      expect(c).toEqual(3);
    } finally {
      await iter.close();
    }
  });
});
