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

import { NotFoundError } from "@/errors";
import { createTestClient } from "@/testutil";

const client = createTestClient();

const createInGroup = async () => {
  const g = await client.groups.create({ name: id.create(), scope: "channel" });
  const ch = await client.channels.create({
    name: id.create(),
    dataType: DataType.FLOAT32,
    virtual: true,
    group: g.key,
  });
  return { ch, path: `${g.name}.${ch.name}` };
};

describe("Path check", () => {
  it("should delete the channel given by its path", async () => {
    const { ch, path } = await createInGroup();
    await client.channels.delete(path);
    await expect(client.channels.retrieve(ch.key)).rejects.toThrow(NotFoundError);
  });

  it("should open a writer on the channel given by its path", async () => {
    const { ch, path } = await createInGroup();
    const writer = await client.openWriter({
      start: TimeStamp.now(),
      channels: [path],
    });
    try {
      expect(writer).toBeDefined();
    } finally {
      await writer.close();
    }
    expect((await client.channels.retrieve(path)).key).toEqual(ch.key);
  });
});
