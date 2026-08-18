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

import { type control } from "@/control";
import { query } from "@/query";
import { createTestClient } from "@/testutil";

const client = createTestClient();

const createVirtual = async () =>
  await client.channels.create({
    name: `control_state_${id.create()}`,
    dataType: DataType.FLOAT64,
    virtual: true,
  });

describe("state", () => {
  it("should mirror an acquire and a release from the cluster", async () => {
    const ch = await createVirtual();
    let latest: query.Cached<control.KeyedState[]> | undefined;
    const stop = client.control.onChange([ch.key], (cached) => {
      latest = cached;
    });
    try {
      expect(await client.control.retrieve([ch.key])).toHaveLength(0);
      const w = await client.openWriter({
        start: TimeStamp.now(),
        channels: [ch.key],
        controlSubject: { key: "seattle", name: "seattle" },
      });
      try {
        await expect
          .poll(() =>
            query.isLive(latest) ? latest.map((s) => s.subject.name) : undefined,
          )
          .toEqual(["seattle"]);
      } finally {
        await w.close();
      }
      await expect
        .poll(() => (query.isLive(latest) ? latest.length : undefined))
        .toEqual(0);
    } finally {
      stop();
    }
  });
});
