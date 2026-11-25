// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, id } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { channel } from "@/channel";
import type { Synnax } from "@/index";
import { createTestClient, createTestClientWithPolicy } from "@/testutil";

const admin = createTestClient();

// The Where of each Console query that 8f-4b moves to the Core.
const QUERIES: Record<string, channel.Where> = {
  SelectAxis: {
    not: { dataType: { in: [DataType.STRING, DataType.JSON, DataType.UUID] } },
    virtual: false,
  },
  "the CSV download": { virtual: false },
  "the Sift pickers": {
    virtual: false,
    isIndex: { eq: false },
    internal: { eq: false },
  },
};

describe("virtual line check", () => {
  it("should keep and drop the same channels as the old client line, for admin and for a reader without the calculation", async () => {
    const index = await admin.channels.create({
      name: id.create(),
      dataType: DataType.TIMESTAMP,
      isIndex: true,
    });
    const data = await admin.channels.create({
      name: id.create(),
      dataType: DataType.FLOAT32,
      index: index.key,
    });
    const plain = await admin.channels.create({
      name: id.create(),
      dataType: DataType.FLOAT32,
      virtual: true,
    });
    const text = await admin.channels.create({
      name: id.create(),
      dataType: DataType.STRING,
      virtual: true,
    });
    const output = await admin.channels.retrieve(
      (
        await admin.calculations.create({
          name: id.create(),
          expression: `return ${plain.name} * 2`,
        })
      ).output,
    );
    // A reader that may retrieve every channel but no calculation.
    const reader = await createTestClientWithPolicy(admin, {
      name: id.create(),
      objects: [channel.resourceID(0)],
      actions: ["retrieve"],
    });
    const all = [index, data, plain, text, output].map((ch) => ch.key);
    const kept = async (c: Synnax, where: channel.Where): Promise<channel.Key[]> =>
      (
        await c.channels.retrieve({
          where: channel.whereZ.parse({ and: [{ key: { in: all } }, where] }),
        })
      )
        .map((ch) => ch.key)
        .sort((a, b) => a - b);
    const sorted = (keys: channel.Key[]) => [...keys].sort((a, b) => a - b);
    const expected: Record<string, channel.Key[]> = {
      SelectAxis: sorted([index.key, data.key, output.key]),
      "the CSV download": sorted([index.key, data.key, output.key]),
      "the Sift pickers": sorted([data.key, output.key]),
    };
    for (const [name, where] of Object.entries(QUERIES)) {
      expect({ name, keys: await kept(admin, where) }).toEqual({
        name,
        keys: expected[name],
      });
      expect({ name, keys: await kept(reader, where) }).toEqual({
        name,
        keys: expected[name],
      });
    }
  });
});
