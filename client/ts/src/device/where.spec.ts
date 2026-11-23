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

import { device } from "@/device";
import { createTestClient } from "@/testutil";
import { vectors } from "@/testutil/vectors";

describe("Where", () => {
  vectors.itFiltersLive(
    vectors.read("devices", device.deviceZ(), device.whereZ),
    () => {
      const client = createTestClient();
      const run = id.create();
      let racks = new Map<number, number>();
      const keyOf = (key: string): string => `${key}-${run}`;
      return {
        client,
        target: "/device/retrieve",
        prepare: async (records) => {
          racks = await vectors.createRacks(
            client,
            records.map(([, d]) => d.rack),
          );
          return {
            keys: new Map(records.map(([alias, d]) => [alias, keyOf(d.key)])),
            values: new Map<unknown, unknown>([
              ...racks,
              ...records.map(([, d]): [string, string] => [d.key, keyOf(d.key)]),
            ]),
          };
        },
        write: async (records) => {
          await client.devices.create(
            records.map(([, { status: _, ...d }]) => ({
              ...d,
              key: keyOf(d.key),
              rack: racks.get(d.rack) ?? d.rack,
            })),
          );
        },
        watch: (where) => client.devices.watch({ where }),
      };
    },
  );
});
