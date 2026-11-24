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

import { rack } from "@/rack";
import { createTestClient } from "@/testutil";
import { vectors } from "@/testutil/vectors";

describe("Where", () => {
  // The Core assigns rack keys, so the store creates the racks under temporary names
  // before the lists open and renames them after. Only the Core makes an embedded rack,
  // so the vectors' embedded rack has no part.
  vectors.itFiltersLive(vectors.read("racks", rack.payloadZ, rack.whereZ), () => {
    const client = createTestClient();
    const keys = new Map<number, number>();
    return {
      client,
      target: "/rack/retrieve",
      prepare: async (records) => {
        const creatable = records.filter(([, r]) => !r.embedded);
        const created = await client.racks.create(
          creatable.map(([, r]) => ({
            name: `where-${id.create()}`,
            integrations: r.integrations,
          })),
        );
        creatable.forEach(([, r], i) => keys.set(r.key, created[i].key));
        return {
          keys: new Map(creatable.map(([alias, r]) => [alias, keys.get(r.key) ?? 0])),
          values: new Map<unknown, unknown>(keys),
        };
      },
      write: async (records) => {
        await Promise.all(
          records
            .filter(([, r]) => keys.has(r.key))
            .map(
              async ([, r]) => await client.racks.rename(keys.get(r.key) ?? 0, r.name),
            ),
        );
      },
      watch: (where) => client.racks.watch({ where }),
    };
  });
});
