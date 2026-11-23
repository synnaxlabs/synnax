// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { ethercat } from "@/ethercat";
import { matches, serverComputed } from "@/ethercat/slave/where.gen";
import { vectors } from "@/testutil/vectors";

describe("Where", () => {
  const { records, cases, whereZ } = vectors.read(
    "ethercatSlaves",
    ethercat.slave.slaveZ,
    ethercat.slave.whereZ,
  );
  cases.forEach(({ name, where, expected, serverComputed: computed }) => {
    const parsed = whereZ.parse(where);
    it(`should match the case: ${name}`, () => {
      expect(serverComputed(parsed)).toBe(computed === true);
      if (computed === true) return;
      const matched = records
        .filter(([, r]) => matches(r, parsed))
        .map(([alias]) => alias);
      expect(matched.sort()).toEqual([...expected].sort());
    });
  });
});
