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

import { createTestClient, spyOnSend } from "@/testutil";

const client = createTestClient();

describe("Lookup store check", () => {
  it("should serve a key from the store after a writer opened its channel by name", async () => {
    const ch = await client.channels.create({
      name: id.create(),
      dataType: DataType.FLOAT32,
      virtual: true,
    });
    const fresh = createTestClient();
    const byName = await fresh.openWriter({
      start: TimeStamp.now(),
      channels: [ch.name],
    });
    await byName.close();
    const send = spyOnSend(fresh);
    try {
      const byKey = await fresh.openWriter({
        start: TimeStamp.now(),
        channels: [ch.key],
      });
      await byKey.close();
      const mention = new RegExp(`\\b${ch.key}\\b`);
      const retrieves = send.mock.calls.filter(
        ([target, req]) =>
          target === "/channel/retrieve" && mention.test(JSON.stringify(req)),
      );
      expect(retrieves).toHaveLength(0);
    } finally {
      send.mockRestore();
    }
  });
});
