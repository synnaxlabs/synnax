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
import { createTestClient } from "@/testutil";

const admin = createTestClient();

// Past the most records that a search returns.
const COUNT = 1010;
const OUTPUTS = 120;
const PAGE = 50;

// Pages through a list as the Pluto select does: a first page, then more on scroll.
const pageAll = async (where: channel.Where, count: number): Promise<channel.Key[]> => {
  const c = createTestClient();
  const list = c.channels.watch({ where, limit: PAGE });
  const off = list.onChange(() => {});
  try {
    let window = await list.retrieve();
    for (let i = 0; i < 2 * Math.ceil(count / PAGE) && window.end != null; i++)
      window = await list.more(PAGE);
    return window.records.map(({ key }) => key);
  } finally {
    off();
  }
};

describe("calculated list paging check", () => {
  const prefix = `cpage_${id.create()}`;
  let outputs: channel.Key[] = [];
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
    const calcs = await Promise.all(
      Array.from(
        { length: OUTPUTS },
        async (_, i) =>
          await admin.calculations.create({
            name: `${prefix}_out_${String(i).padStart(5, "0")}`,
            expression: "return 2",
          }),
      ),
    );
    outputs = calcs.map(({ output }) => output);
  }, 300_000);

  it("should page the Sift query past a thousand channels", async () => {
    const keys = await pageAll(
      {
        virtual: false,
        calculated: false,
        isIndex: { eq: false },
        internal: { eq: false },
        name: { startsWith: prefix },
      },
      COUNT,
    );
    expect(keys).toHaveLength(COUNT);
    expect(keys.filter((k) => outputs.includes(k))).toEqual([]);
  }, 120_000);

  it("should page the CSV query past a thousand channels and keep each output", async () => {
    const keys = await pageAll(
      {
        or: [{ virtual: false }, { calculated: true }],
        isIndex: { eq: false },
        name: { startsWith: prefix },
      },
      COUNT + OUTPUTS,
    );
    expect(keys).toHaveLength(COUNT + OUTPUTS);
    expect(new Set(keys).size).toBe(COUNT + OUTPUTS);
  }, 120_000);

  it("should page a calculated list through every output once", async () => {
    const keys = await pageAll(
      { calculated: true, name: { startsWith: prefix } },
      OUTPUTS,
    );
    expect(new Set(keys).size).toBe(OUTPUTS);
    expect([...keys].sort()).toEqual([...outputs].sort());
  }, 120_000);
});
