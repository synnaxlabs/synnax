// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, Rate, TimeRange, TimeStamp } from "@synnaxlabs/x/telem";
import { describe, expect, test } from "vitest";
import { NotFoundError, UnauthorizedError } from "@/errors"

import { type channel } from "@/channel";
import { newClient } from "@/setupspecs";
import { randomSeries } from "@/util/telem";

const client = newClient();

const newChannel = async (): Promise<channel.Channel> =>
  await client.channels.create({
    name: "test",
    leaseholder: 1,
    rate: Rate.hz(1),
    dataType: DataType.FLOAT64,
  });

const newIndexDataChannelPair = async (): Promise<channel.Channel[]> => {
  const ind = await client.channels.create({
    name: "index",
    leaseholder: 1,
    isIndex: true,
    dataType: DataType.TIMESTAMP,
  })
  const data = await client.channels.create({
    name: "data",
    leaseholder: 1,
    index: ind.key,
    dataType: DataType.INT64,
  })

  return [ind, data]
}

describe("Deleter", () => {
  test("Client - basic delete", async () => {
    const ch = await newChannel();
    const data = randomSeries(10, ch.dataType);
    await client.write(TimeStamp.seconds(0), ch.key, data);

    await client.delete(ch.key, TimeStamp.seconds(2).range(TimeStamp.seconds(5)))

    const res = await client.read(TimeRange.MAX, ch.key);
    expect(res.length).toEqual(data.length - 3);
    expect(res.data.slice(0, 2)).toEqual(data.slice(0, 2))
    expect(res.data.slice(2)).toEqual(data.slice(5))
  });
  test("Client - basic delete by name", async () => {
    const ch = await newChannel()
    const data = randomSeries(10, ch.dataType);
    await client.write(TimeStamp.seconds(0), ch.key, data);

    await client.delete(ch.name, TimeStamp.seconds(2).range(TimeStamp.seconds(5)))

    const res = await client.read(TimeRange.MAX, ch.key);
    expect(res.length).toEqual(data.length - 3);
    expect(res.data.slice(0, 2)).toEqual(data.slice(0, 2))
    expect(res.data.slice(2)).toEqual(data.slice(5))
  })
  test("Client - delete name not found", async () => {
    const ch = await newChannel();
    const data = randomSeries(10, ch.dataType);
    await client.write(TimeStamp.seconds(0), ch.key, data);

    await expect(
      client.delete(["billy bob", ch.name], TimeRange.MAX)
    ).rejects.toThrow(NotFoundError)

    const res = await client.read(TimeRange.MAX, ch.key);
    expect(res.data).toEqual(data);
  })
  test("Client - delete key not found", async () => {
    const ch = await newChannel();
    const data = randomSeries(10, ch.dataType);
    await client.write(TimeStamp.seconds(0), ch.key, data);

    await expect(
      client.delete([ch.key, 1232], TimeRange.MAX)
    ).rejects.toThrow(NotFoundError)

    const res = await client.read(TimeRange.MAX, ch.key);
    expect(res.data).toEqual(data);
  })

  test("Client - delete with writer", async () => {
    const ch = await newChannel();

    const writer = await client.openWriter({
      start: TimeStamp.seconds(10),
      channels: [ch.key],
    });

    await expect(
      client.delete(
      [ch.key], TimeStamp.seconds(12).range(TimeStamp.seconds(30)))
    ).rejects.toThrow(UnauthorizedError)

    await writer.close()
  })

  test("Client - delete index channel alone", async () => {
    const chs = await newIndexDataChannelPair()
    const index = chs[0]
    const dat = chs[1]
    const data = randomSeries(10, dat.dataType)

    const time = BigInt64Array.from({ length: 10 },
      (_, i) => (TimeStamp.milliseconds(i)).valueOf());

    await index.write(0, time)
    await dat.write(0, data)

    await expect(
      client.delete(
        [index.key], TimeStamp.milliseconds(2).range(TimeStamp.milliseconds(5))
      )
    ).rejects.toThrow()
  })
});
