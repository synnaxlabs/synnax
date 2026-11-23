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

import { channel } from "@/channel";
import { createTestClient, waitForStreamLive } from "@/testutil";

const admin = createTestClient();

const createVirtual = async (name: string): Promise<channel.Channel> =>
  await admin.channels.create({ name, dataType: DataType.FLOAT32, virtual: true });

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
});
