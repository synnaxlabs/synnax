// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { uuid } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { ranger } from "@/ranger";
import { createLines } from "@/ranger/where";
import { matches, retrieveInputZ, serverComputed } from "@/ranger/where.gen";
import { itMatchesVectors, readVectors } from "@/testutil/vectors";

const lines = createLines(
  (r, keys) => r.labels?.some((l) => keys.includes(l.key)) ?? false,
);

describe("Where", () => {
  itMatchesVectors(
    readVectors("ranges", ranger.payloadZ, ranger.whereZ),
    (r, where) => matches(lines, r, where),
    (where) => serverComputed(lines, where),
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
    expect(
      inputZ.parse({ name: "a", limit: 2, offset: 1, includeLabels: true }),
    ).toEqual({
      where: { name: { eq: "a" } },
      limit: 2,
      offset: 1,
      includeLabels: true,
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
