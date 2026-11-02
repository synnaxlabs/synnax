// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { framer } from "@/framer";
import { createTestClient, newIndexedPair, secondsLinspace } from "@/testutil";

const client = createTestClient();

describe("history changes", () => {
  it("should decode a published write into the past", async () => {
    const [index] = await newIndexedPair(client);
    const write = async (start: number): Promise<void> => {
      const writer = await client.openWriter({
        start: TimeStamp.seconds(start),
        channels: index.key,
      });
      try {
        await writer.write(index.key, secondsLinspace(start, 2));
      } finally {
        await writer.close();
      }
    };
    const streamer = await client.openStreamer("sy_history_change");
    try {
      await write(20);
      await write(10);
      let change;
      while (change == null) {
        const lists = (await streamer.read())
          .get("sy_history_change")
          .parseJSON(framer.historyChangeZ.array());
        change = lists.flat().find((c) => c.key === index.key);
      }
      expect(change.timeRange).toEqual(
        TimeStamp.seconds(10).range(TimeStamp.seconds(11).add(TimeSpan.nanoseconds(1))),
      );
    } finally {
      streamer.close();
    }
  });
});
