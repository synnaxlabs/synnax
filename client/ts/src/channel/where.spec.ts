// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { readFileSync } from "node:fs";

import { caseconv } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { payloadZ, whereZ } from "@/channel/types.gen";
import { LINES } from "@/channel/where";
import { matches, serverComputed } from "@/channel/where.gen";

// The case table is shared with the Go interpreter, so that the Core and the client
// cache agree on every filter.
const VECTORS_URL = new URL(
  "../../../../x/testdata/filter/vectors.json",
  import.meta.url,
);

interface Case {
  name: string;
  where: unknown;
  expected: string[];
  serverComputed?: boolean;
}

const { channels } = caseconv.snakeToCamel(
  JSON.parse(readFileSync(VECTORS_URL, "utf-8")),
) as { channels: { records: Record<string, unknown>; cases: Case[] } };

const records = Object.entries(channels.records).map(
  ([alias, r]) => [alias, payloadZ.parse(r)] as const,
);

describe("Where", () => {
  channels.cases.forEach((c) => {
    const where = whereZ.parse(c.where);
    if (c.serverComputed === true)
      it(`should mark the case as server-computed: ${c.name}`, () => {
        expect(serverComputed(LINES, where)).toBe(true);
      });
    else
      it(`should match the case: ${c.name}`, () => {
        expect(serverComputed(LINES, where)).toBe(false);
        const matched = records
          .filter(([, r]) => matches(LINES, r, where))
          .map(([alias]) => alias);
        expect(matched.sort()).toEqual([...c.expected].sort());
      });
  });
});
