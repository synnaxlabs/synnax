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

import { createTestClient, waitForStreamLive } from "@/testutil";

const admin = createTestClient();

describe("live virtual lists", () => {
  it("should admit an output that the client never held when its calculation is deleted", async () => {
    const prefix = `virtual_${id.create()}`;
    const input = await admin.channels.create({
      name: id.create(),
      dataType: DataType.FLOAT32,
      virtual: true,
    });
    // The output keeps the data type of the input, so no data type set reaches it.
    const calc = await admin.calculations.create({
      name: `${prefix}_output`,
      expression: `return ${input.name} * 2`,
    });
    const client = createTestClient();
    await client.connect();
    await waitForStreamLive(client.connection);
    // Data channels only, so the index of the calculation stays out.
    const list = client.channels.watch({
      where: { virtual: true, isIndex: { eq: false }, name: { startsWith: prefix } },
    });
    const off = list.onChange(() => {});
    try {
      await list.retrieve();
      const keys = () => (list.getCached()?.records ?? []).map(({ key }) => key);
      expect(keys()).toEqual([]);
      await admin.calculations.delete(calc.key);
      await expect.poll(keys).toEqual([calc.output]);
    } finally {
      off();
    }
  });
});
