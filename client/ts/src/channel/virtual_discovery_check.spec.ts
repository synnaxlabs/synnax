// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, id, sleep, TimeSpan } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { createTestClient, spyOnSend } from "@/testutil";

describe("virtual list discovery check", () => {
  it("should not refetch a virtual list when a lookup finds an outputs entry", async () => {
    const client = createTestClient();
    const prefix = `vdisc_${id.create()}`;
    await client.channels.create({
      name: `${prefix}_v`,
      dataType: DataType.FLOAT32,
      virtual: true,
    });
    const other = await client.channels.create({
      name: `${prefix}_other`,
      dataType: DataType.FLOAT32,
      virtual: true,
    });
    const list = client.channels.watch({
      where: { virtual: true, name: { startsWith: prefix } },
    });
    const off = list.onChange(() => {});
    try {
      await list.retrieve();
      const send = spyOnSend(client);
      // A first lookup only finds what the Core holds, so no list can change.
      await client.calculations.outputs.retrieve([other.key]);
      await sleep.sleep(TimeSpan.milliseconds(500));
      const retrieves = send.mock.calls.filter(
        ([target]) => target === "/channel/retrieve",
      );
      expect(retrieves).toHaveLength(0);
    } finally {
      off();
    }
  });
});
