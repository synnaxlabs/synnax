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
import { matches, serverComputed } from "@/channel/where.gen";
import type Synnax from "@/client";
import { createTestClient, createTestClientWithPolicy } from "@/testutil";
import { itMatchesVectors, readVectors } from "@/testutil/vectors";

describe("Where", () => {
  const vectors = readVectors("channels", channel.payloadZ, channel.whereZ);
  // The internal records of the vectors are the built-in channels.
  const builtIns = new Set(
    vectors.records.filter(([, ch]) => ch.internal).map(([, ch]) => ch.key),
  );
  const lines = createLines(() => builtIns);
  itMatchesVectors(
    vectors,
    (c, where) => matches(lines, c, where),
    (where) => serverComputed(lines, where),
  );

  it("should leave the internal line to the Core before the built-in keys arrive", () => {
    const unknown = createLines(() => null);
    expect(serverComputed(unknown, { internal: { eq: false } })).toBe(true);
  });
});

describe("Where with a Core", () => {
  const admin = createTestClient();

  it("should leave a calculation's output out of virtual channels", async () => {
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
    // A reader that may retrieve every channel but no calculation.
    const reader = await createTestClientWithPolicy(admin, {
      name: id.create(),
      objects: [channel.resourceID(0)],
      actions: ["retrieve"],
    });
    const where = channel.whereZ.parse({ key: { in: [output.key] }, virtual: true });
    const virtualFor = async (c: Synnax) =>
      (await c.channels.retrieve({ where })).some(({ key }) => key === output.key);
    expect(await virtualFor(admin)).toBe(false);
    expect(await virtualFor(reader)).toBe(false);
  });
});
