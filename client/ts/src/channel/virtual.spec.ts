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

import { calculation } from "@/calculation";
import { channel } from "@/channel";
import { framer } from "@/framer";
import {
  createTestClient,
  createTestClientWithPolicy,
  spyOnSend,
  waitForStreamLive,
} from "@/testutil";

const admin = createTestClient();

// An input whose doubles keep its data type, so no data type set reaches an output.
const createInput = async (): Promise<channel.Channel> =>
  await admin.channels.create({
    name: id.create(),
    dataType: DataType.FLOAT32,
    virtual: true,
  });

// Data channels only, so the index of a calculation stays out.
const whereOf = (prefix: string) => ({
  virtual: true,
  isIndex: { eq: false },
  name: { startsWith: prefix },
});

describe("live virtual lists", () => {
  it("should drop and admit an output for a reader that cannot read its calculation", async () => {
    const prefix = `virtual_${id.create()}`;
    const input = await createInput();
    const output = await admin.channels.create({
      name: `${prefix}_output`,
      dataType: DataType.FLOAT32,
      virtual: true,
    });
    const reader = await createTestClientWithPolicy(admin, {
      name: id.create(),
      objects: [channel.resourceID(0), framer.TYPE_RESOURCE_ID],
      actions: ["retrieve"],
    });
    await reader.connect();
    await waitForStreamLive(reader.connection);
    const list = reader.channels.watch({ where: whereOf(prefix) });
    const off = list.onChange(() => {});
    try {
      await list.retrieve();
      const keys = () => (list.getCached()?.records ?? []).map(({ key }) => key);
      expect(keys()).toEqual([output.key]);
      const calc = await admin.calculations.create({
        output: output.key,
        expression: `return ${input.name} * 2`,
      });
      await expect.poll(keys).toEqual([]);
      await admin.calculations.delete(calc.key);
      await expect.poll(keys).toEqual([output.key]);
    } finally {
      off();
    }
  });

  it("should admit an output that the client never held when its calculation is deleted", async () => {
    const prefix = `virtual_${id.create()}`;
    const input = await createInput();
    const output = await admin.channels.create({
      name: `${prefix}_output`,
      dataType: DataType.FLOAT32,
      virtual: true,
    });
    // A calculation on a stored output creates and deletes no other channel, so only
    // the announce of its delete can reach the list.
    const calc = await admin.calculations.create({
      output: output.key,
      expression: `return ${input.name} * 2`,
    });
    const client = createTestClient();
    await client.connect();
    await waitForStreamLive(client.connection);
    const list = client.channels.watch({ where: whereOf(prefix) });
    const off = list.onChange(() => {});
    try {
      await list.retrieve();
      const keys = () => (list.getCached()?.records ?? []).map(({ key }) => key);
      expect(keys()).toEqual([]);
      await admin.calculations.delete(calc.key);
      await expect.poll(keys).toEqual([output.key]);
    } finally {
      off();
    }
  });

  describe("first lookups", () => {
    // Counts the channel retrieves that a first lookup of key starts while the client
    // watches a live virtual list. The client cannot stream changes, so the changes of
    // other specs on the Core start no retrieves.
    const retrievesOfLookup = async (key: channel.Key): Promise<number> => {
      const client = await createTestClientWithPolicy(admin, {
        name: id.create(),
        objects: [channel.TYPE_RESOURCE_ID, calculation.TYPE_RESOURCE_ID],
        actions: ["retrieve"],
      });
      expect((await client.connect()).details.streamDenied).toBe(true);
      const list = client.channels.watch({
        where: { virtual: true, name: { startsWith: id.create() } },
      });
      const off = list.onChange(() => {});
      try {
        await list.retrieve();
        const send = spyOnSend(client);
        await client.calculations.outputs.retrieve([key]);
        await sleep.sleep(TimeSpan.milliseconds(500));
        return send.mock.calls.filter(([target]) => target === "/channel/retrieve")
          .length;
      } finally {
        off();
      }
    };

    it("should not fetch a virtual list again for a channel no calculation writes", async () => {
      const plain = await createInput();
      expect(await retrievesOfLookup(plain.key)).toBe(0);
    });

    it("should not fetch a virtual list again for a calculation's output", async () => {
      const input = await createInput();
      const output = await createInput();
      const calc = await admin.calculations.create({
        output: output.key,
        expression: `return ${input.name} * 2`,
      });
      expect(await retrievesOfLookup(calc.output)).toBe(0);
    });
  });
});
