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
import { createLines } from "@/channel/where";
import { matches } from "@/channel/where.gen";
import { isCalculated, type Synnax } from "@/index";
import { createTestClient, createTestClientWithPolicy } from "@/testutil";

const admin = createTestClient();

// A calculated channel, and a reader that may retrieve every channel but no
// calculation.
const createHidden = async () => {
  const input = await admin.channels.create({
    name: id.create(),
    dataType: DataType.FLOAT32,
    virtual: true,
  });
  const output = await admin.channels.create({
    name: id.create(),
    dataType: DataType.FLOAT32,
    virtual: true,
    expression: `return ${input.name} * 2`,
  });
  const reader = await createTestClientWithPolicy(admin, {
    name: id.create(),
    objects: [channel.resourceID(0)],
    actions: ["retrieve"],
  });
  return { output, reader };
};

describe("calculated channel (check)", () => {
  it("should tell a reader that cannot retrieve the calculation that the channel is calculated", async () => {
    const { output, reader } = await createHidden();
    const seen = await reader.channels.retrieve(output.key);
    expect(seen.expression).toBe("");
    expect(seen.isCalculated).toBe(true);
    expect(isCalculated(seen.payload)).toBe(true);
  });

  it("should agree with the Core on whether a hidden calculation's output is virtual", async () => {
    const { output, reader } = await createHidden();
    const lines = createLines(() => new Set());
    const agreement = async (c: Synnax) => {
      const where = channel.whereZ.parse({ key: { in: [output.key] }, virtual: true });
      const fromCore = await c.channels.retrieve({ where });
      const seen = await c.channels.retrieve(output.key);
      return {
        core: fromCore.some((ch) => ch.key === output.key),
        client: matches(lines, seen.payload, where),
      };
    };
    const asAdmin = await agreement(admin);
    expect(asAdmin.client).toBe(asAdmin.core);
    const asReader = await agreement(reader);
    expect(asReader.client).toBe(asReader.core);
  });
});
