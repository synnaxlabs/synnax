// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type record } from "@synnaxlabs/x";
import { bench, describe } from "vitest";
import z from "zod";

import { query } from "@/query";

interface Thing extends record.Keyed<string> {
  key: string;
  name: string;
  size: number;
}

const requestZ = z.object({
  keys: z.string().array().optional(),
  minSize: z.number().optional(),
  searchTerm: z.string().optional(),
});

class Client extends query.Retriever<typeof requestZ, string, Thing> {
  constructor(cache: query.Cache) {
    const store = cache.createTable<string, Thing>({
      name: "things",
      fetch: async (keys) => keys.map((key) => ({ key, name: key, size: 1 })),
    });
    super(cache, {
      name: "thing",
      table: store,
      request: { schema: requestZ, fetch: async () => [] },
    });
  }
}

const client = new Client(new query.Cache({ openStreamer: null }));
const stableParams = { minSize: 3 };

describe("Retriever.getCached", () => {
  bench("repeat params object (memoized route)", () => {
    client.getCached(stableParams);
  });

  bench("fresh params object per call", () => {
    client.getCached({ minSize: 3 });
  });
});

const stableQuery = { keys: ["a", "b", "c"], rangeKey: "r1", minSize: 3 };

describe("hash", () => {
  bench("repeat object (memoized)", () => {
    query.hash(stableQuery);
  });

  bench("fresh object per call", () => {
    query.hash({ keys: ["a", "b", "c"], rangeKey: "r1", minSize: 3 });
  });
});
