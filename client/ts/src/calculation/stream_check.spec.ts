// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, id } from "@synnaxlabs/x";
import { uniqueName } from "@synnaxlabs/x/testutil";
import { describe, expect, it } from "vitest";

import { calculation } from "@/calculation";
import { channel } from "@/channel";
import type Synnax from "@/client";
import { framer } from "@/framer";
import {
  createEmptyRole,
  createTestClient,
  createTestClientAs,
  type TestPolicy,
} from "@/testutil";
import { user } from "@/user";

const client = createTestClient();

const createVirtual = async () =>
  await client.channels.create({
    name: uniqueName("calc_ch"),
    dataType: DataType.FLOAT32,
    virtual: true,
  });

// Connects a user whose role holds the change readers, the calculation reader, and a
// channel reader for the readable channels.
const connectReader = async (readable: channel.Key[]): Promise<Synnax> => {
  const internal = await client.channels.retrieve({ internal: true });
  const policies: TestPolicy[] = [
    {
      name: "change_reader",
      objects: internal.flatMap(({ key }) => [
        channel.resourceID(key),
        framer.resourceID(String(key)),
      ]),
      actions: ["retrieve"],
    },
    {
      name: "channel_reader",
      objects: readable.map((key) => channel.resourceID(key)),
      actions: ["retrieve"],
    },
    {
      name: "calculation_reader",
      objects: [calculation.TYPE_RESOURCE_ID],
      actions: ["retrieve"],
    },
  ];
  const r = await createEmptyRole(client);
  await client.access.policies.create(policies.map((p) => ({ ...p, role: r.key })));
  const username = id.create();
  const u = await client.users.create({
    username,
    password: "test",
    name: "test",
    role: r.key,
  });
  const c = createTestClientAs(username, "test");
  await c.connect();
  await c.access.policies.retrieve({ subjects: [user.resourceID(u.key)] });
  return c;
};

// Creates a calculation the reader may retrieve, after the calculations before it, and
// waits until the reader's outputs table shows it, so the reader's stream has passed
// every earlier create.
const passStream = async (reader: Synnax, witness: channel.Key) => {
  const shown = await client.calculations.create({
    output: witness,
    expression: "return 2",
  });
  await expect
    .poll(() => reader.calculations.outputs.get(witness), { timeout: 5000 })
    .toEqual({ key: witness, hidden: false, calculation: shown.key });
};

describe("calculation stream check", () => {
  it("should stream as hidden a calculation that reads a channel the subject cannot read", async () => {
    const secret = await createVirtual();
    const output = await createVirtual();
    const witness = await createVirtual();
    const reader = await connectReader([output.key, witness.key]);
    await reader.calculations.outputs.retrieve([output.key, witness.key]);
    const hidden = await client.calculations.create({
      output: output.key,
      expression: `return ${secret.name} * 2`,
    });
    await passStream(reader, witness.key);
    expect(reader.calculations.outputs.get(output.key)).toEqual({
      key: output.key,
      hidden: true,
    });
    expect(reader.calculations.watch(hidden.key).getCached()).toBeUndefined();
  });

  it("should stream nothing of a calculation whose output the subject cannot read", async () => {
    const output = await createVirtual();
    const witness = await createVirtual();
    const reader = await connectReader([witness.key]);
    await reader.calculations.outputs.retrieve([witness.key]);
    const unseen = await client.calculations.create({
      output: output.key,
      expression: "return 1",
    });
    await passStream(reader, witness.key);
    expect(reader.calculations.outputs.get(output.key)).toBeUndefined();
    expect(reader.calculations.watch(unseen.key).getCached()).toBeUndefined();
  });

  it("should stream the full record to a subject who may read every channel", async () => {
    const input = await createVirtual();
    const output = await createVirtual();
    const witness = await createVirtual();
    const reader = await connectReader([input.key, output.key, witness.key]);
    await reader.calculations.outputs.retrieve([output.key, witness.key]);
    const shown = await client.calculations.create({
      output: output.key,
      expression: `return ${input.name} * 2`,
    });
    await passStream(reader, witness.key);
    expect(reader.calculations.outputs.get(output.key)).toEqual({
      key: output.key,
      hidden: false,
      calculation: shown.key,
    });
    expect(reader.calculations.watch(shown.key).getCached()?.expression).toBe(
      `return ${input.name} * 2`,
    );
  });
});
