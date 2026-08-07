// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, id, Series, TimeRange, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { afterAll, describe, expect, it } from "vitest";

import { UnexpectedError } from "@/errors";
import { type Transform } from "@/framer/cache/transform";
import { createTestClient } from "@/testutil";

const client = createTestClient();
const feed = client.openFeed();
afterAll(async () => await feed.close());

const createChannels = async () => {
  const time = await client.channels.create({
    name: id.create(),
    dataType: "timestamp",
    isIndex: true,
  });
  const data = await client.channels.create({
    name: id.create(),
    dataType: "float32",
    index: time.key,
  });
  return { time, data };
};

describe("feed", () => {
  it("should read written samples through the feed cache", async () => {
    const { time, data } = await createChannels();
    const start = TimeStamp.now();
    await client.write(start, {
      [time.key]: [start, start.add(TimeSpan.milliseconds(1))],
      [data.key]: [1, 2],
    });
    const tr = new TimeRange(start, start.add(TimeSpan.seconds(1)));
    const res = await feed.read(tr, data.key);
    expect(Array.from(res)).toEqual([1, 2]);
  });

  it("should apply the configured transform to cached series", async () => {
    const transform: Transform = {
      resolveDataType: () => DataType.FLOAT64,
      convert: (series) =>
        new Series({
          data: new Float64Array((Array.from(series) as number[]).map((v) => v * 2)),
          timeRange: series.timeRange,
          alignment: series.alignment,
        }),
    };
    const transformed = client.openFeed({ transform });
    const { time, data } = await createChannels();
    const start = TimeStamp.now();
    await client.write(start, { [time.key]: [start], [data.key]: [3] });
    const tr = new TimeRange(start, start.add(TimeSpan.seconds(1)));
    const res = await transformed.read(tr, data.key);
    await transformed.close();
    expect(res.dataType.equals(DataType.FLOAT64)).toBe(true);
    expect(Array.from(res)).toEqual([6]);
  });

  it("should deliver written frames to a subscribed stream handler", async () => {
    const { time, data } = await createChannels();
    const received: number[] = [];
    const sub = feed.stream(
      (res) => {
        const series = res.get(data.key);
        if (series != null) received.push(...(Array.from(series) as number[]));
      },
      [data.key],
    );
    // The writer must stream as well as persist: the plain write convenience is
    // persist-only and the relay never broadcasts it.
    const writer = await client.openWriter({
      start: TimeStamp.now(),
      channels: [time.key, data.key],
    });
    try {
      // The stream converges to the registered demand in the background, so writes
      // repeat until one lands on the open stream.
      await expect
        .poll(
          async () => {
            const now = TimeStamp.now();
            await writer.write({ [time.key]: [now], [data.key]: [42] });
            return received.length > 0;
          },
          { timeout: 10000, interval: 250 },
        )
        .toBe(true);
    } finally {
      await writer.close();
    }
    expect(received).toContain(42);
    sub.close();
  });

  it("should reject reads after the feed closes", async () => {
    const closable = client.openFeed();
    await closable.close();
    const tr = new TimeRange(TimeStamp.now(), TimeStamp.now().add(TimeSpan.seconds(1)));
    await expect(closable.read(tr, 123)).rejects.toThrow(UnexpectedError);
  });
});
