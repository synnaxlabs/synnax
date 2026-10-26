// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { payloadZ, whereZ } from "@/ranger/types.gen";
import { createLines } from "@/ranger/where";
import { matches, serverComputed } from "@/ranger/where.gen";
import { readVectors } from "@/testutil/vectors";

const { records, cases } = readVectors("ranges", payloadZ, whereZ);
const lines = createLines(
  (r, keys) => r.labels?.some((l) => keys.includes(l.key)) ?? false,
);

describe("Where", () => {
  cases.forEach(({ name, where, expected, serverComputed: computed }) => {
    if (computed === true)
      it(`should mark the case as server-computed: ${name}`, () => {
        expect(serverComputed(lines, where)).toBe(true);
      });
    else
      it(`should match the case: ${name}`, () => {
        expect(serverComputed(lines, where)).toBe(false);
        const matched = records
          .filter(([, r]) => matches(lines, r, where))
          .map(([alias]) => alias);
        expect(matched.sort()).toEqual([...expected].sort());
      });
  });
});
