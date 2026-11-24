// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, id } from "@synnaxlabs/x";
import { beforeAll, describe, expect, it } from "vitest";

import type { channel } from "@/channel";
import { createFileTestClient, createTestClient } from "@/testutil";

const admin = createFileTestClient();

// Past the most records that a search returns.
const COUNT = 1010;
const PAGE = 50;

// Pages through a list as the Pluto select does: a first page, then more on scroll.
const pageAll = async (where: channel.Where): Promise<number> => {
  const c = createTestClient();
  const list = c.channels.watch({ where, limit: PAGE });
  const off = list.onChange(() => {});
  try {
    let window = await list.retrieve();
    // Bounded: COUNT / PAGE pages reach every match.
    for (let i = 0; i < 2 * (COUNT / PAGE) && window.end != null; i++)
      window = await list.more(PAGE);
    return window.records.length;
  } finally {
    off();
  }
};

describe("virtual list paging breaker", () => {
  const prefix = `vpage_${id.create()}`;
  beforeAll(async () => {
    const index = await admin.channels.create({
      name: `${prefix}_time`,
      dataType: DataType.TIMESTAMP,
      isIndex: true,
    });
    await admin.channels.create(
      Array.from({ length: COUNT }, (_, i) => ({
        name: `${prefix}_${String(i).padStart(5, "0")}`,
        dataType: DataType.FLOAT32,
        index: index.key,
      })),
    );
  }, 120_000);

  it("should page a list with no virtual line past a thousand channels", async () => {
    expect(
      await pageAll({ isIndex: { eq: false }, name: { startsWith: prefix } }),
    ).toBe(COUNT);
  }, 60_000);

  it("should page a virtual: false list, as the Sift pickers send, past a thousand channels", async () => {
    expect(
      await pageAll({
        virtual: false,
        isIndex: { eq: false },
        internal: { eq: false },
        name: { startsWith: prefix },
      }),
    ).toBe(COUNT);
  }, 60_000);
});
