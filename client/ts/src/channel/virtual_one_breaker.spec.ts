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

import { query } from "@/query";
import { createTestClient, spyOnSend, waitForStreamLive } from "@/testutil";

const admin = createTestClient();

const liveKey = (cached: unknown) =>
  query.isLive(cached as never) ? (cached as { key: number }).key : null;

describe("virtual one-record breaker", () => {
  it("should stop answering a virtual: true lookup by name once a calculation writes the channel", async () => {
    const name = `vone_${id.create()}`;
    const plain = await admin.channels.create({
      name,
      dataType: DataType.FLOAT32,
      virtual: true,
    });
    const c = createTestClient();
    await c.connect();
    await waitForStreamLive(c.connection);
    const live = c.channels.watch({ name, virtual: true });
    const off = live.onChange(() => {});
    try {
      expect((await live.retrieve()).key).toBe(plain.key);
      const send = spyOnSend(c);
      await admin.calculations.create({ output: plain.key, expression: "return 2" });
      await expect(
        createTestClient().channels.retrieve({ name, virtual: true }),
      ).rejects.toThrow();
      await sleep.sleep(TimeSpan.seconds(1));
      // The Core was asked again, and answered that nothing matches.
      const asked = send.mock.calls.filter(([t]) => t === "/channel/retrieve");
      expect(asked.length).toBeGreaterThan(0);
      expect(liveKey(live.getCached())).toBeNull();
    } finally {
      off();
    }
  });

  it("should stop answering a virtual: false lookup by name once its calculation is deleted", async () => {
    const name = `vone_${id.create()}`;
    const calc = await admin.calculations.create({ name, expression: "return 2" });
    const c = createTestClient();
    await c.connect();
    await waitForStreamLive(c.connection);
    const live = c.channels.watch({ name, virtual: false });
    const off = live.onChange(() => {});
    try {
      expect((await live.retrieve()).key).toBe(calc.output);
      await admin.calculations.delete(calc.key);
      await expect(
        createTestClient().channels.retrieve({ name, virtual: false }),
      ).rejects.toThrow();
      await sleep.sleep(TimeSpan.seconds(1));
      expect(liveKey(live.getCached())).toBeNull();
    } finally {
      off();
    }
  });
});
