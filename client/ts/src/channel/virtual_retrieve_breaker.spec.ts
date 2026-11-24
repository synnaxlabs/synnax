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
import type { Synnax } from "@/index";
import {
  createTestClient,
  createTestClientWithPolicy,
  spyOnSend,
  waitForStreamLive,
} from "@/testutil";

const admin = createTestClient();

const createVirtual = async (name: string): Promise<channel.Channel> =>
  await admin.channels.create({ name, dataType: DataType.FLOAT32, virtual: true });

// A reader that may retrieve and stream every channel but no calculation.
const createReader = async (): Promise<Synnax> => {
  const reader = await createTestClientWithPolicy(admin, {
    name: id.create(),
    objects: [channel.resourceID(0), framer.resourceID("")],
    actions: ["retrieve"],
  });
  await waitForStreamLive(reader.connection);
  return reader;
};

// watchBoth watches the virtual: true and virtual: false lists of the channels under
// prefix. It returns the keys that each list holds and a function that closes both.
const watchBoth = async (c: Synnax, prefix: string) => {
  const lists = [true, false].map((virtual) =>
    c.channels.watch({
      where: channel.whereZ.parse({
        virtual,
        isIndex: { eq: false },
        name: { startsWith: prefix },
      }),
    }),
  );
  const closers = lists.map((list) => list.onChange(() => {}));
  await Promise.all(lists.map(async (list) => await list.retrieve()));
  const keys = (virtual: boolean) =>
    lists[virtual ? 0 : 1].getCached()?.records.map(({ key }) => key) ?? [];
  return { keys, close: () => closers.forEach((close) => close()) };
};

// listFetches returns the retrieves that send carried for a list with a virtual line.
const listFetches = (send: ReturnType<typeof spyOnSend>) =>
  send.mock.calls.filter(
    ([target, req]) =>
      target === "/channel/retrieve" &&
      (req as { where?: { virtual?: unknown } }).where?.virtual != null,
  );

describe("virtual retrieve breaker", () => {
  it("should answer a repeated virtual: false retrieve after a calculation starts to write a channel", async () => {
    const prefix = `vr_${id.create()}`;
    const plain = await createVirtual(`${prefix}_a`);
    const input = await createVirtual(`in_${id.create()}`);
    const c = createTestClient();
    await c.connect();
    await waitForStreamLive(c.connection);
    const ask = async () =>
      (
        await c.channels.retrieve({
          where: channel.whereZ.parse({ virtual: false, name: { startsWith: prefix } }),
        })
      ).map(({ key }) => key);
    expect(await ask()).toEqual([]);
    await admin.calculations.create({
      output: plain.key,
      expression: `return ${input.name} * 2`,
    });
    expect(await ask()).toEqual([plain.key]);
  });

  it("should answer a repeated virtual: true retrieve after a calculation is deleted", async () => {
    const prefix = `vr_${id.create()}`;
    const input = await createVirtual(`in_${id.create()}`);
    const calc = await admin.calculations.create({
      name: `${prefix}_a`,
      expression: `return ${input.name} * 2`,
    });
    const c = createTestClient();
    await c.connect();
    await waitForStreamLive(c.connection);
    const ask = async () =>
      (
        await c.channels.retrieve({
          where: channel.whereZ.parse({
            virtual: true,
            isIndex: { eq: false },
            name: { startsWith: prefix },
          }),
        })
      ).map(({ key }) => key);
    expect(await ask()).toEqual([]);
    await admin.calculations.delete(calc.key);
    expect(await ask()).toEqual([calc.output]);
  });

  it("should move a channel from a reader's live virtual: true list to its virtual: false list when a calculation starts to write it, with no refetch", async () => {
    const prefix = `vr_${id.create()}`;
    const plain = await createVirtual(`${prefix}_a`);
    const input = await createVirtual(`in_${id.create()}`);
    const reader = await createReader();
    const { keys, close } = await watchBoth(reader, prefix);
    try {
      expect(keys(true)).toEqual([plain.key]);
      expect(keys(false)).toEqual([]);
      const send = spyOnSend(reader);
      await admin.calculations.create({
        output: plain.key,
        expression: `return ${input.name} * 2`,
      });
      await expect.poll(() => keys(false)).toEqual([plain.key]);
      await expect.poll(() => keys(true)).toEqual([]);
      // Past the debounce of a refetch that the change could start.
      await sleep.sleep(TimeSpan.milliseconds(500));
      expect(listFetches(send)).toHaveLength(0);
    } finally {
      close();
    }
  });

  it("should move an output from a reader's live virtual: false list to its virtual: true list when its calculation is deleted, with no refetch", async () => {
    const prefix = `vr_${id.create()}`;
    const input = await createVirtual(`in_${id.create()}`);
    const calc = await admin.calculations.create({
      name: `${prefix}_a`,
      expression: `return ${input.name} * 2`,
    });
    const reader = await createReader();
    const { keys, close } = await watchBoth(reader, prefix);
    try {
      expect(keys(false)).toEqual([calc.output]);
      expect(keys(true)).toEqual([]);
      const send = spyOnSend(reader);
      await admin.calculations.delete(calc.key);
      await expect.poll(() => keys(true)).toEqual([calc.output]);
      await expect.poll(() => keys(false)).toEqual([]);
      await sleep.sleep(TimeSpan.milliseconds(500));
      expect(listFetches(send)).toHaveLength(0);
    } finally {
      close();
    }
  });
});
