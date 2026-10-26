// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { payloadZ, whereZ } from "@/rack/types.gen";
import { LINES } from "@/rack/where";
import { matches, serverComputed } from "@/rack/where.gen";
import { readVectors } from "@/testutil/vectors";

const { records, cases } = readVectors("racks", payloadZ, whereZ);

describe("Where", () => {
  cases.forEach(({ name, where, expected, serverComputed: computed }) => {
    if (computed === true)
      it(`should mark the case as server-computed: ${name}`, () => {
        expect(serverComputed(LINES, where)).toBe(true);
      });
    else
      it(`should match the case: ${name}`, () => {
        expect(serverComputed(LINES, where)).toBe(false);
        const matched = records
          .filter(([, r]) => matches(LINES, r, where))
          .map(([alias]) => alias);
        expect(matched.sort()).toEqual([...expected].sort());
      });
  });
});
