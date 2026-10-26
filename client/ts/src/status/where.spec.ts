// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { statusZ, whereZ } from "@/status/types.gen";
import { type Lines, matches, serverComputed } from "@/status/where.gen";
import { readVectors } from "@/testutil/vectors";

const { records, cases } = readVectors("statuses", statusZ(), whereZ);
const lines: Lines = {
  labels: (s, keys) => s.labels?.some((l) => keys.includes(l.key)) ?? false,
};

describe("Where", () => {
  cases.forEach(({ name, where, expected }) => {
    it(`should match the case: ${name}`, () => {
      expect(serverComputed(lines, where)).toBe(false);
      const matched = records
        .filter(([, s]) => matches(lines, s, where))
        .map(([alias]) => alias);
      expect(matched.sort()).toEqual([...expected].sort());
    });
  });
});
