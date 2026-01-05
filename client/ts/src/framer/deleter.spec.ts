// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeRange, TimeStamp } from "@synnaxlabs/x";
import { describe, expect, test } from "vitest";

import { NotFoundError, UnauthorizedError } from "@/errors";
import { newIndexedPair } from "@/testutil/channels";
import { createTestClient } from "@/testutil/client";
import { secondsLinspace } from "@/testutil/telem";
import { randomSeries } from "@/util/telem";

const client = createTestClient();

describe("Deleter", () => {
  test("Client - basic delete", async () => {
    const [indexCh, dataCh] = await newIndexedPair(client);
    const data = [1, 2, 3, 4, 5, 6, 7, 8, 9, 10];
    await client.write(TimeStamp.seconds(5), {
      [indexCh.key]: secondsLinspace(5, 10),
      [dataCh.key]: data,
    });
    const res = await client.read(TimeRange.MAX, dataCh.key);
    expect(res.data.length).toEqual(10);
    await client.delete(dataCh.key, TimeStamp.seconds(5).range(TimeStamp.seconds(7)));

    const deletedRes = await client.read(TimeRange.MAX, dataCh.key);
    expect(deletedRes.data.length).toEqual(8);
    expect(Array.from(deletedRes)).toEqual([3, 4, 5, 6, 7, 8, 9, 10]);
  });
  test("Client - basic delete by name", async () => {
    const [indexCh, dataCh] = await newIndexedPair(client);
    const data = [1, 2, 3, 4, 5, 6, 7, 8, 9, 10];
    await client.write(TimeStamp.seconds(5), {
      [indexCh.key]: secondsLinspace(5, 10),
      [dataCh.key]: data,
    });
    const res = await client.read(TimeRange.MAX, dataCh.key);
    expect(res.data.length).toEqual(10);
    await client.delete(dataCh.name, TimeStamp.seconds(5).range(TimeStamp.seconds(7)));

    const deletedRes = await client.read(TimeRange.MAX, dataCh.key);
    expect(deletedRes.data.length).toEqual(8);
    expect(Array.from(deletedRes)).toEqual([3, 4, 5, 6, 7, 8, 9, 10]);
  });
  test("Client - delete name not found", async () => {
    const [indexCh, dataCh] = await newIndexedPair(client);
    const data = randomSeries(10, dataCh.dataType);
    await client.write(TimeStamp.seconds(5), {
      [indexCh.key]: secondsLinspace(5, 10),
      [dataCh.key]: data,
    });

    await expect(
      client.delete(["nonexistent_channel_name", dataCh.name], TimeRange.MAX),
    ).rejects.toThrow(NotFoundError);

    const res = await client.read(TimeRange.MAX, dataCh.key);
    expect(res.data).toEqual(data);
  });
  test("Client - delete key not found", async () => {
    const [indexCh, dataCh] = await newIndexedPair(client);
    const data = randomSeries(10, dataCh.dataType);
    await client.write(TimeStamp.seconds(0), {
      [indexCh.key]: secondsLinspace(0, 10),
      [dataCh.key]: data,
    });

    await expect(client.delete([indexCh.key, 1232], TimeRange.MAX)).rejects.toThrow(
      NotFoundError,
    );

    const res = await client.read(TimeRange.MAX, dataCh.key);
    expect(res.data).toEqual(data);
  });

  test("Client - delete with writer", async () => {
    const [indexCh] = await newIndexedPair(client);

    const writer = await client.openWriter({
      start: TimeStamp.seconds(10),
      channels: [indexCh.key],
    });

    await expect(
      client.delete([indexCh.key], TimeStamp.seconds(12).range(TimeStamp.seconds(30))),
    ).rejects.toThrow(UnauthorizedError);

    await writer.close();
  });

  test("Client - delete index channel alone", async () => {
    const [indexCh, dataCh] = await newIndexedPair(client);
    const index = indexCh;
    const dat = dataCh;
    const data = randomSeries(10, dat.dataType);

    const time = BigInt64Array.from({ length: 10 }, (_, i) =>
      TimeStamp.milliseconds(i).valueOf(),
    );

    await index.write(0, time);
    await dat.write(0, data);

    await expect(
      client.delete(
        [index.key],
        TimeStamp.milliseconds(2).range(TimeStamp.milliseconds(5)),
      ),
    ).rejects.toThrow();
  });
});
