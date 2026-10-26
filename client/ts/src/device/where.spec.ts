// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { deviceZ, whereZ } from "@/device/types.gen";
import { matches, serverComputed } from "@/device/where.gen";
import { readVectors } from "@/testutil/vectors";

const { records, cases } = readVectors("devices", deviceZ(), whereZ);

describe("Where", () => {
  cases.forEach(({ name, where, expected, serverComputed: computed }) => {
    if (computed === true)
      it(`should mark the case as server-computed: ${name}`, () => {
        expect(serverComputed(where)).toBe(true);
      });
    else
      it(`should match the case: ${name}`, () => {
        expect(serverComputed(where)).toBe(false);
        const matched = records
          .filter(([, r]) => matches(r, where))
          .map(([alias]) => alias);
        expect(matched.sort()).toEqual([...expected].sort());
      });
  });
});
