// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, id, Series, TimeRange } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { NotFoundError } from "@/errors";
import { createTestClient, newIndexedPair, secondsLinspace } from "@/testutil";

const client = createTestClient();

describe("Lookup check", () => {
  it("should delete the channel whose name starts with digits", async () => {
    const ch = await client.channels.create({
      name: `${10_000_000_000 + Math.floor(Math.random() * 1_000_000_000)}_${id.create()}`,
      dataType: DataType.FLOAT32,
      virtual: true,
    });
    await client.channels.delete(ch.name);
    await expect(client.channels.retrieve(ch.key)).rejects.toThrow(NotFoundError);
  });

  it("should read a CSV of channels given as payloads", async () => {
    const [index, data] = await newIndexedPair(client);
    const writer = await client.openWriter({ channels: [index.key, data.key] });
    try {
      await writer.write({
        [index.key]: secondsLinspace(1, 3),
        [data.key]: new Series({ data: [1, 2, 3], dataType: data.dataType }),
      });
      await writer.commit();
    } finally {
      await writer.close();
    }
    const stream = await client.readCSV(TimeRange.MAX, [index.payload, data.payload]);
    expect(await new Response(stream).text()).toContain("3");
  });
});
