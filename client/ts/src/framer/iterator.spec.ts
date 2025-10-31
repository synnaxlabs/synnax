// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeRange, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { describe, expect, test } from "vitest";

import { AUTO_SPAN } from "@/framer/iterator";
import { newIndexedPair } from "@/testutil/channels";
import { createTestClient } from "@/testutil/client";
import { secondsLinspace } from "@/testutil/telem";
import { randomSeries } from "@/util/telem";

const client = createTestClient();

describe("Iterator", () => {
  test("happy path", async () => {
    const channels = await newIndexedPair(client);
    const writer = await client.openWriter({
      start: TimeStamp.SECOND,
      channels,
    });
    const [idx_ch, data_ch] = channels;
    try {
      await writer.write({
        [idx_ch.key]: secondsLinspace(1, 10),
        [data_ch.key]: randomSeries(10, data_ch.dataType),
      });
      await writer.write({
        [idx_ch.key]: secondsLinspace(11, 10),
        [data_ch.key]: randomSeries(10, data_ch.dataType),
      });
      await writer.write({
        [idx_ch.key]: secondsLinspace(21, 10),
        [data_ch.key]: randomSeries(10, data_ch.dataType),
      });
      await writer.commit();
    } finally {
      await writer.close();
    }

    const iter = await client.openIterator(
      new TimeRange(TimeStamp.SECOND, TimeSpan.seconds(30)),
      channels,
    );

    try {
      expect(await iter.seekFirst()).toBe(true);
      let c = 0;
      while (await iter.next(TimeSpan.seconds(1))) {
        c++;
        expect(iter.value.get(idx_ch.key)).toHaveLength(1);
        expect(iter.value.get(data_ch.key)).toHaveLength(1);
      }
      expect(c).toEqual(29);
    } finally {
      await iter.close();
    }
  });
  test("chunk size", async () => {
    const channels = await newIndexedPair(client);
    const [idx_ch, data_ch] = channels;
    const writer = await client.openWriter({
      start: TimeStamp.SECOND,
      channels,
    });
    await writer.write({
      [idx_ch.key]: secondsLinspace(1, 10),
      [data_ch.key]: randomSeries(10, data_ch.dataType),
    });
    await writer.close();
    const iter = await client.openIterator(TimeRange.MAX, channels, { chunkSize: 4 });

    try {
      expect(await iter.seekFirst()).toBe(true);

      expect(await iter.next(AUTO_SPAN)).toBe(true);
      expect(iter.value.get(idx_ch.key).data).toEqual(
        new BigInt64Array(secondsLinspace(1, 4).map((v) => v.valueOf())),
      );

      expect(await iter.next(AUTO_SPAN)).toBe(true);
      expect(iter.value.get(idx_ch.key).data).toEqual(
        new BigInt64Array(secondsLinspace(5, 4).map((v) => v.valueOf())),
      );

      expect(await iter.next(AUTO_SPAN)).toBe(true);
      expect(iter.value.get(idx_ch.key).data).toEqual(
        new BigInt64Array(secondsLinspace(9, 2).map((v) => v.valueOf())),
      );

      expect(await iter.next(AUTO_SPAN)).toBe(false);
    } finally {
      await iter.close();
    }
  });
});
