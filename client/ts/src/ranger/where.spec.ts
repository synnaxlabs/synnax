// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { compare, TimeStamp, uuid } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { ranger } from "@/ranger";
import { LINES } from "@/ranger/where";
import {
  compareCursors,
  cursor,
  matches,
  retrieveInputZ,
  serverComputed,
} from "@/ranger/where.gen";
import { itMatchesVectors, readVectors } from "@/testutil/vectors";

describe("Where", () => {
  itMatchesVectors(
    readVectors("ranges", ranger.payloadZ, ranger.whereZ),
    (r, where) => matches(LINES, r, where),
    (where) => serverComputed(LINES, where),
  );
});

describe("retrieveInputZ", () => {
  const inputZ = retrieveInputZ(ranger.retrieveRequestZ.omit({ where: true }).shape);
  const key = uuid.create();
  const labelKey = uuid.create();

  it("should read a value as eq", () => {
    expect(inputZ.parse({ key })).toEqual({ where: { key: { eq: key } } });
  });

  it("should read a list as in", () => {
    expect(inputZ.parse({ name: ["a", "b"] })).toEqual({
      where: { name: { in: ["a", "b"] } },
    });
  });

  it("should keep an operator", () => {
    const name = { startsWith: "Hot", caseInsensitive: true };
    expect(inputZ.parse({ name })).toEqual({ where: { name } });
  });

  it("should read a value on a list line as a list of one", () => {
    expect(inputZ.parse({ labels: labelKey })).toEqual({
      where: { labels: [labelKey] },
    });
  });

  it("should read the children of or", () => {
    expect(
      inputZ.parse({
        or: [{ name: { startsWith: "Hot" } }, { labels: labelKey }],
      }),
    ).toEqual({
      where: { or: [{ name: { startsWith: "Hot" } }, { labels: [labelKey] }] },
    });
  });

  it("should split the options from the fields", () => {
    expect(inputZ.parse({ name: "a", limit: 2, searchTerm: "b" })).toEqual({
      where: { name: { eq: "a" } },
      limit: 2,
      searchTerm: "b",
    });
  });

  it("should leave where unset when only options are set", () => {
    expect(inputZ.parse({ limit: 5 }).where).toBeUndefined();
  });

  it("should reject a field set to undefined", () => {
    const { error } = inputZ.safeParse({ name: undefined });
    expect(error?.issues).toMatchObject([{ code: "invalid_union", path: ["name"] }]);
  });

  it("should reject a field the filter does not have", () => {
    const { error } = inputZ.safeParse({ color: "red" });
    expect(error?.issues).toMatchObject([
      { code: "unrecognized_keys", keys: ["color"] },
    ]);
  });
});

describe("compareCursors", () => {
  const [low, high] = [uuid.create(), uuid.create()].sort(compare.uuids);
  const record = (key: string, name: string, start: number): ranger.Payload =>
    ranger.payloadZ.parse({
      key,
      name,
      timeRange: { start: TimeStamp.seconds(start), end: TimeStamp.seconds(start + 1) },
    });

  it("should order names by lower case, then by key", () => {
    const order: ranger.Order = { field: "name", descending: false };
    const positions = [
      record(high, "alpha", 1),
      record(low, "Beta", 1),
      record(low, "ALPHA", 1),
    ].map((r) => cursor(r, order));
    positions.sort((a, b) => compareCursors(a, b, order));
    expect(positions).toEqual([
      { key: low, name: "ALPHA" },
      { key: high, name: "alpha" },
      { key: low, name: "Beta" },
    ]);
  });

  it("should reverse the order and the key tie when descending", () => {
    const order: ranger.Order = { field: "time_range_start", descending: true };
    const early = cursor(record(low, "a", 1), order);
    const tie = cursor(record(high, "a", 1), order);
    const late = cursor(record(low, "a", 2), order);
    expect([early, late, tie].sort((a, b) => compareCursors(a, b, order))).toEqual([
      late,
      tie,
      early,
    ]);
  });

  it("should order by end time, then by key", () => {
    const order: ranger.Order = { field: "time_range_end", descending: false };
    const at = (key: string, start: number, end: number): ranger.Payload =>
      ranger.payloadZ.parse({
        key,
        name: "a",
        timeRange: { start: TimeStamp.seconds(start), end: TimeStamp.seconds(end) },
      });
    const late = cursor(at(low, 1, 9), order);
    const tie = cursor(at(high, 2, 5), order);
    const early = cursor(at(low, 3, 5), order);
    expect(late).toEqual({ key: low, timeRangeEnd: TimeStamp.seconds(9) });
    expect([late, tie, early].sort((a, b) => compareCursors(a, b, order))).toEqual([
      early,
      tie,
      late,
    ]);
    expect(() => compareCursors({ key: low }, late, order)).toThrow(
      "a cursor has no time_range_end",
    );
  });

  it("should return 0 for the same position", () => {
    const order: ranger.Order = { field: "name", descending: false };
    const a = cursor(record(low, "a", 1), order);
    expect(compareCursors(a, { ...a }, order)).toBe(0);
  });

  it("should order by key alone with no order", () => {
    const positions = [record(high, "a", 1), record(low, "b", 2)].map((r) => cursor(r));
    expect(positions).toEqual([{ key: high }, { key: low }]);
    expect(positions.sort((a, b) => compareCursors(a, b))).toEqual([
      { key: low },
      { key: high },
    ]);
  });

  it("should throw for a position with no value for the order's field", () => {
    const order: ranger.Order = { field: "name", descending: false };
    expect(() => compareCursors({ key: low }, { key: high, name: "a" }, order)).toThrow(
      "a cursor has no name",
    );
  });
});
