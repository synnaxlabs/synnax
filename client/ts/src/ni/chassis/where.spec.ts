// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { compare, uuid } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { ni } from "@/ni";
import {
  compareCursors,
  cursor,
  matches,
  serverComputed,
} from "@/ni/chassis/where.gen";
import { vectors } from "@/testutil/vectors";

describe("Where", () => {
  vectors.itMatches(
    vectors.read("niChassis", ni.chassis.chassisZ, ni.chassis.whereZ),
    matches,
    serverComputed,
  );
});

describe("compareCursors", () => {
  const [low, high] = [uuid.create(), uuid.create()].sort(compare.uuids);
  const record = (key: string, name: string): ni.chassis.Chassis =>
    ni.chassis.chassisZ.parse({ key, rack: 1, name, resourceName: "r", model: "m" });

  it("should order names by lower case, then by key", () => {
    const order: ni.chassis.Order = { field: "name", descending: false };
    const positions = [
      record(high, "alpha"),
      record(low, "Beta"),
      record(low, "ALPHA"),
    ].map((r) => cursor(r, order));
    positions.sort((a, b) => compareCursors(a, b, order));
    expect(positions).toEqual([
      { key: low, name: "ALPHA" },
      { key: high, name: "alpha" },
      { key: low, name: "Beta" },
    ]);
  });

  it("should reverse the order and the key tie when descending", () => {
    const order: ni.chassis.Order = { field: "name", descending: true };
    const early = cursor(record(low, "a"), order);
    const tie = cursor(record(high, "a"), order);
    const late = cursor(record(low, "b"), order);
    expect([early, late, tie].sort((a, b) => compareCursors(a, b, order))).toEqual([
      late,
      tie,
      early,
    ]);
  });

  it("should return 0 for the same position", () => {
    const order: ni.chassis.Order = { field: "name", descending: false };
    const a = cursor(record(low, "a"), order);
    expect(compareCursors(a, { ...a }, order)).toBe(0);
  });

  it("should order by key alone with no order", () => {
    const positions = [record(high, "a"), record(low, "b")].map((r) => cursor(r));
    expect(positions).toEqual([{ key: high }, { key: low }]);
    expect(positions.sort((a, b) => compareCursors(a, b))).toEqual([
      { key: low },
      { key: high },
    ]);
  });

  it("should throw for a position with no name", () => {
    const order: ni.chassis.Order = { field: "name", descending: false };
    expect(() => compareCursors({ key: low }, { key: high, name: "a" }, order)).toThrow(
      "a cursor has no name",
    );
  });
});
