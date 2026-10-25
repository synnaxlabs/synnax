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

import { payloadZ, whereZ } from "@/rack/types.gen";
import { type Lines, matches, serverComputed } from "@/rack/where.gen";

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

const { racks } = caseconv.snakeToCamel(
  JSON.parse(readFileSync(VECTORS_URL, "utf-8")),
) as { racks: { records: Record<string, unknown>; cases: Case[] } };

const records = Object.entries(racks.records).map(
  ([alias, r]) => [alias, payloadZ.parse(r)] as const,
);
const lines: Lines = { hostIsNode: null };

describe("Where", () => {
  racks.cases.forEach((c) => {
    const where = whereZ.parse(c.where);
    if (c.serverComputed === true)
      it(`should mark the case as server-computed: ${c.name}`, () => {
        expect(serverComputed(lines, where)).toBe(true);
      });
    else
      it(`should match the case: ${c.name}`, () => {
        expect(serverComputed(lines, where)).toBe(false);
        const matched = records
          .filter(([, r]) => matches(lines, r, where))
          .map(([alias]) => alias);
        expect(matched.sort()).toEqual([...c.expected].sort());
      });
  });
});
