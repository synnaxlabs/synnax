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
import { framer } from "@/framer";
import {
  createTestClient,
  createTestClientWithPolicy,
  waitForStreamLive,
} from "@/testutil";

const admin = createTestClient();

describe("live virtual lists", () => {
  it("should drop and admit an output for a reader that cannot read its calculation", async () => {
    const prefix = `virtual_${id.create()}`;
    const output = await admin.channels.create({
      name: `${prefix}_output`,
      dataType: DataType.FLOAT32,
      virtual: true,
    });
    const reader = await createTestClientWithPolicy(admin, {
      name: id.create(),
      objects: [channel.resourceID(0), framer.TYPE_RESOURCE_ID],
      actions: ["retrieve"],
    });
    await reader.connect();
    await waitForStreamLive(reader.connection);
    const list = reader.channels.watch({
      where: { virtual: true, name: { startsWith: prefix } },
    });
    const off = list.onChange(() => {});
    try {
      await list.retrieve();
      const keys = () => (list.getCached()?.records ?? []).map(({ key }) => key);
      expect(keys()).toEqual([output.key]);
      const calc = await admin.calculations.create({
        output: output.key,
        expression: "return 2",
      });
      await expect.poll(keys).toEqual([]);
      await admin.calculations.delete(calc.key);
      await expect.poll(keys).toEqual([output.key]);
    } finally {
      off();
    }
  });
});
