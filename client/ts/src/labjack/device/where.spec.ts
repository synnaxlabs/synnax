// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { id } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { labjack } from "@/labjack";
import { createTestClient, waitForStreamLive } from "@/testutil";
import { vectors } from "@/testutil/vectors";

describe("whereZ", () => {
  const { cases } = vectors.read(
    "labjackDevices",
    labjack.device.deviceZ,
    labjack.device.whereZ,
  );
  it.each(cases)("should parse the where of $name", ({ where }) => {
    expect(labjack.device.whereZ.safeParse(where).success).toBe(true);
  });

  it("should reject a where on a field the filter does not have", () => {
    expect(labjack.device.whereZ.safeParse({ connection: { eq: "x" } }).success).toBe(
      false,
    );
  });
});

describe("live list order", () => {
  const namesInOrder = async (
    order: labjack.device.Order | undefined,
  ): Promise<string[]> => {
    const client = createTestClient();
    await client.connect();
    await waitForStreamLive(client.connection);
    const rack = await client.racks.create({ name: `where-order-${id.create()}` });
    const prefix = id.create();
    const names = ["beta", "Alpha", "gamma"].map((n) => `${prefix}${n}`);
    const list = client.labjack.devices.watch({
      where: { name: { in: names } },
      order,
    });
    const close = list.onChange(() => {});
    try {
      await list.retrieve();
      await client.labjack.devices.create(
        names.map((name) => ({
          name,
          rack: rack.key,
          serial: Math.floor(Math.random() * 0xfffffffe) + 1,
          connection: "LJM_ctUSB" as const,
          model: "t7" as const,
          ports: { model: "t7" as const },
        })),
      );
      const held = () =>
        list.getCached()?.records.map(({ name }) => name.slice(prefix.length)) ?? [];
      await expect.poll(() => held().length).toBe(names.length);
      return held();
    } finally {
      close();
    }
  };

  it("should order names by lower case", async () => {
    expect(await namesInOrder({ field: "name", descending: false })).toEqual([
      "Alpha",
      "beta",
      "gamma",
    ]);
  });

  it("should reverse the order when descending", async () => {
    expect(await namesInOrder({ field: "name", descending: true })).toEqual([
      "gamma",
      "beta",
      "Alpha",
    ]);
  });
});
