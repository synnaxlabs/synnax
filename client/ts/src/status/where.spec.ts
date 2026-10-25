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

import { statusZ, whereZ } from "@/status/types.gen";
import { createLines } from "@/status/where";
import { matches, serverComputed } from "@/status/where.gen";

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
}

const { statuses } = caseconv.snakeToCamel(
  JSON.parse(readFileSync(VECTORS_URL, "utf-8")),
) as { statuses: { records: Record<string, unknown>; cases: Case[] } };

const records = Object.entries(statuses.records).map(
  ([alias, s]) => [alias, statusZ().parse(s)] as const,
);
const lines = createLines(
  (key, labelKey) =>
    records
      .find(([, s]) => s.key === key)?.[1]
      .labels?.some((l) => l.key === labelKey) ?? false,
);

describe("Where", () => {
  statuses.cases.forEach((c) => {
    it(`should match the case: ${c.name}`, () => {
      const where = whereZ.parse(c.where);
      expect(serverComputed(lines, where)).toBe(false);
      const matched = records
        .filter(([, s]) => matches(lines, s, where))
        .map(([alias]) => alias);
      expect(matched.sort()).toEqual([...c.expected].sort());
    });
  });
});
