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

import { createTestClient, secondsLinspace } from "@/testutil";

const client = createTestClient();

describe("Record frame check", () => {
  it("should write a record column to the writer's channel that holds its digit name", async () => {
    const index = await client.channels.create({
      name: id.create(),
      dataType: DataType.TIMESTAMP,
      isIndex: true,
    });
    const name = String(10_000_000_000 + Math.floor(Math.random() * 1_000_000_000));
    const data = await client.channels.create({
      name,
      dataType: DataType.FLOAT64,
      index: index.key,
    });
    const writer = await client.openWriter({ channels: [index.key, data.key] });
    try {
      await writer.write({
        [index.key]: secondsLinspace(1, 3),
        [name]: new Series({ data: [1, 2, 3], dataType: DataType.FLOAT64 }),
      });
      await writer.commit();
    } finally {
      await writer.close();
    }
    const fr = await client.read(TimeRange.MAX, [index.key, data.key]);
    expect(Array.from(fr.get(data.key).series[0])).toEqual([1, 2, 3]);
  });
});
