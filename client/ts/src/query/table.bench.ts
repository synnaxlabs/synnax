// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { bench, describe } from "vitest";

import { query } from "@/query";

interface Row {
  n: number;
}

const makeTable = (keyedSubscribers: number): query.Table<string, Row> => {
  const table = new query.Table<string, Row>({ onError: () => {} });
  for (let i = 0; i < keyedSubscribers; i++) table.subscribe(() => {}, `key-${i}`);
  return table;
};

describe("Table.notify", () => {
  const small = makeTable(10);
  let a = 0;
  bench("one event, 10 keyed subscribers", () => {
    small.set("key-0", { n: a++ });
  });

  const large = makeTable(1000);
  let b = 0;
  bench("one event, 1,000 keyed subscribers", () => {
    large.set("key-0", { n: b++ });
  });
});
