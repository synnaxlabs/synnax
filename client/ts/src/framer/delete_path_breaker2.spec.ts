// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, id, TimeRange, TimeStamp } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { createTestClient, secondsLinspace } from "@/testutil";

const client = createTestClient();

const step = async <T>(label: string, f: () => Promise<T>): Promise<T> => {
  try {
    return await f();
  } catch (e) {
    throw new Error(`${label}: ${String(e)}`, { cause: e });
  }
};

describe("breaker2 delete by path", () => {
  it("should delete the channel at a path", async () => {
    const g = await client.groups.create({ name: id.create(), scope: "channel" });
    const ch = await client.channels.create({
      name: id.create(),
      dataType: DataType.FLOAT64,
      virtual: true,
      group: g.key,
    });
    const path = `${g.name}.${ch.name}`;
    expect((await client.channels.retrieve(path)).key).toEqual(ch.key);
    await step("delete", async () => await client.channels.delete(path));
    const fresh = createTestClient();
    const after = await step(
      "retrieve",
      async () => await fresh.channels.retrieve({ where: { key: { eq: ch.key } } }),
    );
    expect(after.map((c) => c.key)).toEqual([]);
  });

  it("should delete the data of the channel at a path", async () => {
    const g = await client.groups.create({ name: id.create(), scope: "channel" });
    const index = await client.channels.create({
      name: id.create(),
      dataType: DataType.TIMESTAMP,
      isIndex: true,
      group: g.key,
    });
    await client.write(TimeStamp.seconds(1), index.key, secondsLinspace(1, 3));
    expect((await client.read(TimeRange.MAX, index.key)).length).toEqual(3);
    await step(
      "delete",
      async () => await client.delete(`${g.name}.${index.name}`, TimeRange.MAX),
    );
    expect((await client.read(TimeRange.MAX, index.key)).length).toEqual(0);
  });
});
