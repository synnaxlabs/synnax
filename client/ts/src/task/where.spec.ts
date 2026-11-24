// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { uuid } from "@synnaxlabs/x";
import { describe } from "vitest";

import { task } from "@/task";
import { createTestClient } from "@/testutil";
import { vectors } from "@/testutil/vectors";

describe("Where", () => {
  vectors.itFiltersLive(vectors.read("tasks", task.payloadZ(), task.whereZ), () => {
    const client = createTestClient();
    let racks = new Map<number, number>();
    const keys = new Map<string, string>();
    return {
      client,
      target: "/task/retrieve",
      prepare: async (records) => {
        racks = await vectors.createRacks(
          client,
          records.map(([, t]) => t.rack),
        );
        records.forEach(([, t]) => keys.set(t.key, uuid.create()));
        return {
          keys: new Map(records.map(([alias, t]) => [alias, keys.get(t.key)])),
          values: new Map<unknown, unknown>([...racks, ...keys]),
        };
      },
      write: async (records) => {
        await client.tasks.create(
          records.map(([, { status: _, ...t }]) => ({
            ...t,
            key: keys.get(t.key),
            rack: racks.get(t.rack) ?? t.rack,
          })),
        );
      },
      watch: (where) => client.tasks.watch({ where }),
    };
  });
});
