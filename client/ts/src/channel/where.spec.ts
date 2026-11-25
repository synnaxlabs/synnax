// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, id, sleep, TimeSpan } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { channel } from "@/channel";
import { framer } from "@/framer";
import {
  createTestClient,
  createTestClientWithPolicy,
  spyOnSend,
  waitForStreamLive,
} from "@/testutil";
import { vectors } from "@/testutil/vectors";

describe("Where", () => {
  // The Core assigns channel keys and requires unique names, so the store creates the
  // channels under temporary names before the lists open and gives each its vector name
  // with a suffix after. A client cannot create a channel leased to another node or an
  // internal channel, and the Core leases a calculated channel to no node, so those
  // vector channels have no part.
  vectors.itFiltersLive(
    vectors.read("channels", channel.payloadZ, channel.whereZ),
    () => {
      const client = createTestClient();
      const run = id.create();
      const keys = new Map<number, number>();
      const nameOf = (name: string): string => `${name}_${run}`;
      const creatable = (ch: channel.Payload): boolean =>
        ch.leaseholder === 1 && !ch.internal && !ch.calculated;
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
});

describe("Where with a Core", () => {
  const admin = createTestClient();

  it("should keep a hidden calculation's output out of a live virtual list with no refetch", async () => {
    const input = await admin.channels.create({
      name: id.create(),
      dataType: DataType.FLOAT32,
      virtual: true,
    });
    // A reader that may retrieve and stream every channel but no calculation.
    const reader = await createTestClientWithPolicy(admin, {
      name: id.create(),
      objects: [channel.resourceID(0), framer.resourceID("")],
      actions: ["retrieve"],
    });
    await waitForStreamLive(reader.connection);
    const where = channel.whereZ.parse({
      virtual: true,
      name: { startsWith: `where_hidden_${input.key}` },
    });
    const list = reader.channels.watch({ where });
    const close = list.onChange(() => {});
    try {
      await list.retrieve();
      const send = spyOnSend(reader);
      const [plain, output] = await admin.channels.create([
        {
          name: `where_hidden_${input.key}_plain`,
          dataType: DataType.FLOAT32,
          virtual: true,
        },
        {
          name: `where_hidden_${input.key}_output`,
          dataType: DataType.FLOAT32,
          virtual: true,
          expression: `return ${input.name} * 2`,
        },
      ]);
      const keys = () => list.getCached()?.records.map(({ key }) => key);
      await expect.poll(keys).toContain(plain.key);
      expect(keys()).not.toContain(output.key);
      // Past the debounce of a refetch that the create could start.
      await sleep.sleep(TimeSpan.milliseconds(500));
      expect(keys()).not.toContain(output.key);
      const listFetches = send.mock.calls.filter(
        ([target, req]) =>
          target === "/channel/retrieve" &&
          (req as { where?: { virtual?: unknown } }).where?.virtual != null,
      );
      expect(listFetches).toHaveLength(0);
      const fromCore = (await reader.channels.retrieve({ where })).map(
        ({ key }) => key,
      );
      expect(fromCore).toContain(plain.key);
      expect(fromCore).not.toContain(output.key);
    } finally {
      close();
    }
  });
});
