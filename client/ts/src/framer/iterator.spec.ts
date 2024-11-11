// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, Rate, TimeRange, TimeSpan, TimeStamp } from "@synnaxlabs/x/telem";
import { describe, expect, test } from "vitest";

import { type channel } from "@/channel";
import { AUTO_SPAN } from "@/framer/iterator";
import { newClient } from "@/setupspecs";
import { randomSeries } from "@/util/telem";

const client = newClient();

const newChannel = async (): Promise<channel.Channel> => await client.channels.create({
    name: "test",
    leaseholder: 1,
    rate: Rate.hz(25),
    dataType: DataType.FLOAT64,
  });

describe("Iterator", () => {
  test("happy path", async () => {
    const ch = await newChannel();
    const writer = await client.openWriter({
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

    const iter = await client.openIterator(
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
  test("chunk size", async () => {
    const ch = await newChannel();
    const data = Float64Array.of(0, 1, 2, 3, 4, 5, 6, 7, 8, 9);
    await ch.write(0, data);

    const iter = await client.openIterator(TimeRange.MAX, [ch.key], { chunkSize: 4 });

    try {
      expect(await iter.seekFirst()).toBeTruthy();

      expect(await iter.next(AUTO_SPAN)).toBeTruthy();
      expect(iter.value.get(ch.key).data).toEqual(Float64Array.of(0, 1, 2, 3));

      expect(await iter.next(AUTO_SPAN)).toBeTruthy();
      expect(iter.value.get(ch.key).data).toEqual(Float64Array.of(4, 5, 6, 7));

      expect(await iter.next(AUTO_SPAN)).toBeTruthy();
      expect(iter.value.get(ch.key).data).toEqual(Float64Array.of(8, 9));

      expect(await iter.next(AUTO_SPAN)).toBeFalsy();
    } finally {
      await iter.close();
    }
  });
});
