// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { id } from "@synnaxlabs/x";
import { describe } from "vitest";

import { status } from "@/status";
import { createTestClient } from "@/testutil";
import { vectors } from "@/testutil/vectors";

describe("Where", () => {
  vectors.itFiltersLive(
    vectors.read("statuses", status.statusZ(), status.whereZ),
    () => {
      const client = createTestClient();
      const run = id.create();
      let labels = new Map<string, string>();
      const keyOf = (key: string): string => `${key}-${run}`;
      return {
        client,
        target: "/status/retrieve",
        prepare: async (records) => {
          labels = await vectors.createLabels(
            client,
            records.flatMap(([, s]) => s.labels ?? []),
          );
          return {
            keys: new Map(records.map(([alias, s]) => [alias, keyOf(s.key)])),
            values: new Map<unknown, unknown>([
              ...labels,
              ...records.map(([, s]): [string, string] => [s.key, keyOf(s.key)]),
            ]),
          };
        },
        write: async (records) => {
          await client.statuses.set(
            records.map(([, s]) => ({
              ...s,
              key: keyOf(s.key),
              labels: s.labels?.map((l) => labels.get(l) ?? l),
            })),
          );
        },
        watch: (where) => client.statuses.watch({ where }),
      };
    },
  );
});
