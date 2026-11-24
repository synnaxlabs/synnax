// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, id } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";
import { z } from "zod";

import { channel } from "@/channel";
import { createLines } from "@/channel/where";
import { matches, serverComputed } from "@/channel/where.gen";
import { createTestClient, waitForStreamLive } from "@/testutil";
import { vectors } from "@/testutil/vectors";

describe("Where", () => {
  // The internal records of the vectors are the built-in channels.
  const recordZ = channel.payloadZ.extend({ internal: z.boolean().default(false) });
  // The Core assigns channel keys and requires unique names, so the store creates the
  // channels under temporary names before the lists open and gives each its vector name
  // with a suffix after. A client cannot create a channel leased to another node or an
  // internal channel, and the Core leases a calculated channel to no node, so those
  // vector channels have no part.
  vectors.itFiltersLive(
    vectors.read("channels", recordZ, channel.whereZ),
    () => {
      const client = createTestClient();
      const run = id.create();
      const keys = new Map<number, number>();
      const nameOf = (name: string): string => `${name}_${run}`;
      const creatable = (ch: z.infer<typeof recordZ>): boolean =>
        ch.leaseholder === 1 && !ch.internal && !channel.isCalculated(ch);
      return {
        client,
        target: "/channel/retrieve",
        prepare: async (records) => {
          const create = async (batch: typeof records) => {
            const created = await client.channels.create(
              batch.map(([, ch]) => ({
                name: `where_${id.create()}`,
                dataType: ch.dataType,
                isIndex: ch.isIndex,
                virtual: ch.virtual,
                index: keys.get(ch.index) ?? 0,
              })),
            );
            batch.forEach(([, ch], i) => keys.set(ch.key, created[i].key));
          };
          const kept = records.filter(([, ch]) => creatable(ch));
          await create(kept.filter(([, ch]) => ch.isIndex));
          await create(kept.filter(([, ch]) => !ch.isIndex));
          return {
            keys: new Map(
              kept.map(([alias, ch]): [string, number] => [
                alias,
                keys.get(ch.key) ?? 0,
              ]),
            ),
            values: new Map<unknown, unknown>([
              ...keys,
              ...records.map(([, ch]): [string, string] => [ch.name, nameOf(ch.name)]),
            ]),
          };
        },
        write: async (records) => {
          const kept = records.filter(([, ch]) => keys.has(ch.key));
          await client.channels.rename(
            kept.map(([, ch]) => keys.get(ch.key) ?? 0),
            kept.map(([, ch]) => nameOf(ch.name)),
          );
        },
        watch: (where) => client.channels.watch({ where }),
      };
    },
  );

  it("should keep a calculated channel out of a live list of virtual channels", async () => {
    const client = createTestClient();
    await client.connect();
    await waitForStreamLive(client.connection);
    const source = await client.channels.create({
      name: `where_source_${id.create()}`,
      dataType: DataType.FLOAT32,
      virtual: true,
    });
    const name = { startsWith: `where_virtual_${source.key}` };
    const all = client.channels.watch({ where: { name } });
    const virtual = client.channels.watch({ where: { virtual: true, name } });
    const closers = [all.onChange(() => {}), virtual.onChange(() => {})];
    try {
      await Promise.all([all.retrieve(), virtual.retrieve()]);
      const [plain, calculated] = await client.channels.create([
        { name: `${name.startsWith}_plain`, dataType: DataType.FLOAT32, virtual: true },
        {
          name: `${name.startsWith}_calculated`,
          dataType: DataType.FLOAT32,
          virtual: true,
          expression: `return ${source.name} * 2`,
        },
      ]);
      const keys = (list: typeof all) =>
        list
          .getCached()
          ?.records.map(({ key }) => key)
          .sort();
      await expect
        .poll(() => keys(all))
        .toEqual(expect.arrayContaining([plain.key, calculated.key]));
      expect(keys(virtual)).toContain(plain.key);
      expect(keys(virtual)).not.toContain(calculated.key);
    } finally {
      closers.forEach((close) => close());
    }
  });

  it("should match the internal line against the built-in keys", () => {
    const table = vectors.read("channels", recordZ, channel.whereZ);
    const builtIn = table.records.find(([, ch]) => ch.internal)?.[1];
    const user = table.records.find(([, ch]) => !ch.internal)?.[1];
    if (builtIn == null || user == null)
      throw new Error("the vectors hold no internal and user channel");
    const lines = createLines(() => new Set([builtIn.key]));
    expect(serverComputed(lines, { internal: { eq: true } })).toBe(false);
    expect(matches(lines, builtIn, { internal: { eq: true } })).toBe(true);
    expect(matches(lines, user, { internal: { eq: true } })).toBe(false);
    expect(matches(lines, user, { internal: { eq: false } })).toBe(true);
  });

  it("should leave the internal line to the Core before the built-in keys arrive", () => {
    const unknown = createLines(() => null);
    expect(serverComputed(unknown, { internal: { eq: false } })).toBe(true);
    expect(serverComputed(unknown, { virtual: true })).toBe(false);
  });
});
