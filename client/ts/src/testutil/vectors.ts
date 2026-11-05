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
import { expect, it } from "vitest";
import type z from "zod";

// The case table is shared with the Go interpreter, so that the Core and the client
// cache agree on every filter.
const VECTORS_URL = new URL(
  "../../../../core/testdata/filter/vectors.json",
  import.meta.url,
);

/** A filter case: the records that where matches, by alias. */
export interface Case<W> {
  name: string;
  where: W;
  expected: string[];
  serverComputed?: boolean;
}

/** The parsed records, keyed by alias, and the cases of one entity. */
export interface Vectors<R, W> {
  records: [alias: string, record: R][];
  cases: Case<W>[];
}

interface RawVectors {
  records: Record<string, unknown>;
  cases: Case<unknown>[];
}

/**
 * Reads the filter vectors of the entity, such as "ranges", and parses its records and
 * wheres with the given schemas.
 */
export const readVectors = <R, W>(
  entity: string,
  recordZ: z.ZodType<R>,
  whereZ: z.ZodType<W>,
): Vectors<R, W> => {
  const all = caseconv.snakeToCamel(
    JSON.parse(readFileSync(VECTORS_URL, "utf-8")),
  ) as Record<string, RawVectors>;
  const { records, cases } = all[entity];
  return {
    records: Object.entries(records).map(([alias, r]) => [alias, recordZ.parse(r)]),
    cases: cases.map((c) => ({ ...c, where: whereZ.parse(c.where) })),
  };
};

/**
 * Adds a spec for each case: the client matches the case, or a case the vectors mark as
 * server-computed is one the client leaves to the Core.
 */
export const itMatchesVectors = <R, W>(
  { records, cases }: Vectors<R, W>,
  matches: (record: R, where: W) => boolean,
  serverComputed: (where: W) => boolean,
): void =>
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
