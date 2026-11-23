// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { compare, id, uuid } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { ni } from "@/ni";
import { createTestClient, waitForStreamLive } from "@/testutil";
import { vectors } from "@/testutil/vectors";

describe("Where", () => {
  vectors.itFiltersLive(
    vectors.read("niDevices", ni.device.deviceZ, ni.device.whereZ),
    () => {
      const client = createTestClient();
      let racks = new Map<number, number>();
      const keys = new Map<string, string>();
      return {
        client,
        target: "/ni/device/retrieve",
        prepare: async (records) => {
          racks = await vectors.createRacks(
            client,
            records.map(([, d]) => d.rack),
          );
          records.forEach(([, d]) => keys.set(d.key, uuid.create()));
          return {
            keys: new Map(
              records.map(([alias, d]) => [alias, keys.get(d.key) ?? d.key]),
            ),
            values: new Map<unknown, unknown>([...racks, ...keys]),
          };
        },
        write: async (records) => {
          await client.ni.devices.create(
            records.map(([, { status: _, ...d }]) => ({
              ...d,
              key: keys.get(d.key),
              rack: racks.get(d.rack) ?? d.rack,
            })),
          );
        },
        watch: (where) => client.ni.devices.watch({ where }),
      };
    },
  );
});

describe("live list order", () => {
  const orderOf = async (
    order: ni.device.Order | undefined,
    records: Array<{ key: string; name: string }>,
  ): Promise<string[]> => {
    const client = createTestClient();
    await client.connect();
    await waitForStreamLive(client.connection);
    const rack = await client.racks.create({ name: `where-order-${id.create()}` });
    const list = client.ni.devices.watch({
      where: { key: { in: records.map(({ key }) => key) } },
      order,
    });
    const close = list.onChange(() => {});
    try {
      await list.retrieve();
      await client.ni.devices.create(
        records.map(({ key, name }) => ({
          key,
          name,
          rack: rack.key,
          resourceName: "r",
          model: "m",
        })),
      );
      const held = () => list.getCached()?.records.map(({ key }) => key) ?? [];
      await expect.poll(() => held().length).toBe(records.length);
      return held();
    } finally {
      close();
    }
  };
  const keys = () => [uuid.create(), uuid.create(), uuid.create()].sort(compare.uuids);

  it("should order names by lower case, then by key", async () => {
    const [a, b, c] = keys();
    expect(
      await orderOf({ field: "name", descending: false }, [
        { key: c, name: "alpha" },
        { key: b, name: "Beta" },
        { key: a, name: "ALPHA" },
      ]),
    ).toEqual([a, c, b]);
  });

  it("should reverse the order and the key tie when descending", async () => {
    const [a, b, c] = keys();
    expect(
      await orderOf({ field: "name", descending: true }, [
        { key: a, name: "a" },
        { key: b, name: "a" },
        { key: c, name: "b" },
      ]),
    ).toEqual([c, b, a]);
  });

  it("should order by key alone with no order", async () => {
    const [a, b] = keys();
    expect(
      await orderOf(undefined, [
        { key: b, name: "a" },
        { key: a, name: "b" },
      ]),
    ).toEqual([a, b]);
  });
});
