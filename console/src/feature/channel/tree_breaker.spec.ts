// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type access,
  calculation,
  channel as channelClient,
  DataType,
  framer,
  type Synnax,
  user,
} from "@synnaxlabs/client";
import {
  createEmptyRole,
  createTestClient,
  createTestClientAs,
  type TestPolicy,
} from "@synnaxlabs/client/testutil";
import { id } from "@synnaxlabs/x";
import { assertDefined, uniqueName } from "@synnaxlabs/x/testutil";
import { act, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Channel } from "@/feature/channel";
import { createResource, renderTreeContextMenu } from "@/platform/tree/testutil";

const client = createTestClient();
const Item = Channel.TREE_ITEMS.channel;

const createVirtual = async () =>
  await client.channels.create({
    name: uniqueName("calc_ch"),
    dataType: DataType.FLOAT32,
    virtual: true,
  });

interface Account {
  username: string;
  grantee: access.Grantee;
  policies: access.policy.Policy[];
}

// Creates a user whose role holds the policies.
const createAccount = async (policies: TestPolicy[]): Promise<Account> => {
  const r = await createEmptyRole(client);
  const created = await client.access.policies.create(
    policies.map((p) => ({ ...p, role: r.key })),
  );
  const username = id.create();
  const u = await client.users.create({
    username,
    password: "test",
    name: "test",
    role: r.key,
  });
  return { username, grantee: { key: u.key, role: r.key }, policies: created };
};

// Connects as the user, as the Console does, so the cache streams changes.
const connect = async ({ username, grantee }: Account): Promise<Synnax> => {
  const c = createTestClientAs(username, "test");
  await c.connect();
  await c.access.policies.retrieve({ subjects: [user.resourceID(grantee.key)] });
  return c;
};

const CALCULATION_READER: TestPolicy = {
  name: "calculation_reader",
  objects: [calculation.TYPE_RESOURCE_ID],
  actions: ["retrieve"],
};

describe("Edit calculation in the channel tree", () => {
  it("should withhold Edit calculation for a calculation created while the menu is open that reads a channel the user cannot read", async () => {
    const secret = await createVirtual();
    const output = await createVirtual();
    const witness = await createVirtual();
    // The user may stream the change channels but not read secret.
    const internal = await client.channels.retrieve({ internal: true });
    const account = await createAccount([
      {
        name: "change_reader",
        objects: internal.flatMap(({ key }) => [
          channelClient.resourceID(key),
          framer.resourceID(String(key)),
        ]),
        actions: ["retrieve"],
      },
      {
        name: "channel_writer",
        objects: [output, witness].map(({ key }) => channelClient.resourceID(key)),
        actions: ["retrieve", "update"],
      },
      CALCULATION_READER,
    ]);
    const writer = await connect(account);
    assertDefined(Item.ContextMenu);
    await renderTreeContextMenu(Item.ContextMenu, {
      client: writer,
      resources: [createResource(channelClient.resourceID(output.key), output.name)],
    });
    expect(await screen.findByText("Rename")).toBeTruthy();
    await waitFor(() =>
      expect(writer.calculations.outputs.get(output.key)).toEqual({
        key: output.key,
        hidden: false,
        calculation: null,
      }),
    );
    await client.calculations.create({
      output: output.key,
      expression: `return ${secret.name} * 2`,
    });
    // A calculation the writer may read, created second on the same change channel,
    // proves the writer's stream has passed the first create.
    const shown = await client.calculations.create({
      output: witness.key,
      expression: "return 2",
    });
    await waitFor(
      () =>
        expect(writer.calculations.outputs.get(witness.key)).toEqual({
          key: witness.key,
          hidden: false,
          calculation: shown.key,
        }),
      { timeout: 5000 },
    );
    // The Core hides the calculation from the writer, as a fresh session shows.
    const fresh = await connect(account);
    expect(await fresh.calculations.outputs.retrieve([output.key])).toEqual([
      { key: output.key, hidden: true },
    ]);
    await act(async () => {});
    expect(screen.queryByText("Edit calculation")).toBeNull();
  });

  it("should withhold Edit calculation once the user loses access to the calculation", async () => {
    const output = await createVirtual();
    const calc = await client.calculations.create({
      output: output.key,
      expression: "return 2",
    });
    const account = await createAccount([
      {
        name: "channel_writer",
        objects: [channelClient.TYPE_RESOURCE_ID],
        actions: ["retrieve", "update"],
      },
      { name: "data_reader", objects: [framer.resourceID("")], actions: ["retrieve"] },
      CALCULATION_READER,
    ]);
    const writer = await connect(account);
    assertDefined(Item.ContextMenu);
    await renderTreeContextMenu(Item.ContextMenu, {
      client: writer,
      resources: [createResource(channelClient.resourceID(output.key), output.name)],
    });
    expect(await screen.findByText("Edit calculation")).toBeTruthy();
    const readable = writer.access.granted.watch(account.grantee, {
      objects: calculation.resourceID(calc.key),
      action: "retrieve",
    });
    expect(await readable.retrieve()).toBe(true);
    const reader = account.policies.find((p) => p.name === CALCULATION_READER.name);
    assertDefined(reader);
    await client.access.policies.delete(reader.key);
    // The writer's own client has seen the loss.
    await waitFor(() => expect(readable.getCached()).toBe(false), { timeout: 5000 });
    const fresh = await connect(account);
    expect(await fresh.calculations.outputs.retrieve([output.key])).toEqual([
      { key: output.key, hidden: true },
    ]);
    await act(async () => {});
    expect(screen.queryByText("Edit calculation")).toBeNull();
  });
});
