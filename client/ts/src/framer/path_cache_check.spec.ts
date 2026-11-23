// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, id, TimeStamp } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { type channel } from "@/channel";
import { createTestClient } from "@/testutil";

const client = createTestClient();

const keysOf = async (
  streamer: Awaited<ReturnType<typeof client.openStreamer>>,
  want: channel.Key[],
): Promise<channel.Key[]> => {
  const fr = await streamer.read();
  return want.filter((k) => fr.get(k).length > 0);
};

describe("writer path cache check", () => {
  it("should write a record path to the channel that holds the path now", async () => {
    const g = await client.groups.create({ name: id.create(), scope: "channel" });
    const create = async () =>
      await client.channels.create({
        name: id.create(),
        dataType: DataType.FLOAT64,
        virtual: true,
        group: g.key,
      });
    const a = await create();
    const b = await create();
    const path = `${g.name}.${a.name}`;
    const streamer = await client.openStreamer([a.key, b.key]);
    const writer = await client.openWriter({
      start: TimeStamp.now(),
      channels: [a.key, b.key],
    });
    try {
      await writer.write({ [path]: [1] });
      expect(await keysOf(streamer, [a.key, b.key])).toEqual([a.key]);
      const old = a.name;
      await client.channels.rename(a.key, id.create());
      await client.channels.rename(b.key, old);
      expect((await client.channels.retrieve(path)).key).toEqual(b.key);
      await writer.write({ [path]: [2] });
      expect(await keysOf(streamer, [a.key, b.key])).toEqual([b.key]);
    } finally {
      await writer.close();
      streamer.close();
    }
  });
});
