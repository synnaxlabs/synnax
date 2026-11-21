// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { device, NotFoundError } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import type { Status } from "@synnaxlabs/lyra/status";
import { uniqueName } from "@synnaxlabs/x/testutil";
import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { EtherCAT } from "@/feature/ethercat";
import { createReachedStatus, createSlaveDevice } from "@/feature/ethercat/testutil";
import { Notifications } from "@/platform/notifications";
import { CaptureStatuses, createConsoleWrapper, installPortalRoot } from "@/testutil";

const client = createTestClient();

/** Creates an old slave and a connected new slave of a model no other spec uses. */
const createPair = async () => {
  const rack = await client.racks.create({ name: uniqueName("ecat_rack") });
  const model = {
    network: "eth0",
    vendorId: Math.floor(Math.random() * 2 ** 31),
    productCode: Math.floor(Math.random() * 2 ** 31),
  };
  const old = await createSlaveDevice(client, rack.key, {
    ...model,
    name: uniqueName("old"),
    position: 3,
  });
  const slave = await createSlaveDevice(
    client,
    rack.key,
    { ...model, name: uniqueName("new"), position: 5, serial: 77 },
    false,
  );
  await createReachedStatus(client, slave);
  return { rack, model, old, slave };
};

/** Creates two old slaves, at positions 4 and 3, and a connected new slave. */
const createTwoOld = async () => {
  const rack = await client.racks.create({ name: uniqueName("ecat_rack") });
  const model = {
    network: "eth0",
    vendorId: Math.floor(Math.random() * 2 ** 31),
    productCode: Math.floor(Math.random() * 2 ** 31),
  };
  const fourth = await createSlaveDevice(client, rack.key, {
    ...model,
    name: uniqueName("old"),
    position: 4,
  });
  const third = await createSlaveDevice(client, rack.key, {
    ...model,
    name: uniqueName("old"),
    position: 3,
  });
  const slave = await createSlaveDevice(
    client,
    rack.key,
    { ...model, name: uniqueName("new"), position: 5, serial: 77 },
    false,
  );
  await createReachedStatus(client, slave);
  return { third, fourth, slave };
};

/** The answers that the prompt about the new slave with name offers, in order. */
const findAnswers = async (name: string): Promise<string[]> =>
  await waitFor(() => {
    const prompt = screen
      .getAllByRole("status")
      .find((el) => within(el).queryByText(name) != null);
    if (prompt == null) throw new Error(`No notification names ${name}`);
    return within(prompt)
      .getAllByRole("button")
      .map((b) => (b.textContent ?? "").trim())
      .filter((t) => t !== "No" && t !== "");
  });

const Listener = (): null => {
  EtherCAT.Device.useListenForReplacements();
  return null;
};

const renderPrompt = async (): Promise<{
  statuses: () => Status.NotificationSpec[];
}> => {
  const { wrapper } = await createConsoleWrapper({ client });
  let latest: Status.NotificationSpec[] = [];
  render(
    <>
      <Listener />
      <Notifications.Feed notifications={EtherCAT.Device.NOTIFICATIONS} />
      <CaptureStatuses onStatuses={(s) => (latest = s)} />
    </>,
    { wrapper },
  );
  return { statuses: () => latest };
};

/** The notification that asks the question, once it shows. */
const findPrompt = async (question: string): Promise<HTMLElement> => {
  await screen.findByText(question);
  const prompt = screen
    .getAllByRole("status")
    .find((el) => within(el).queryByText(question) != null);
  if (prompt == null) throw new Error(`No notification asks "${question}"`);
  return prompt;
};

describe("useListenForReplacements", () => {
  installPortalRoot();

  it("should ask once whether a new slave is the old slave of its model", async () => {
    const { statuses } = await renderPrompt();
    const { rack, model, old, slave } = await createPair();
    const question = `Is this your old ${old.name}?`;
    await findPrompt(question);
    await createSlaveDevice(client, rack.key, { ...model, position: 4 });
    const other = await createPair();
    await findPrompt(`Is this your old ${other.old.name}?`);
    const asked = statuses().filter((s) => s.name === slave.name);
    expect(asked.map((s) => [s.message, s.count])).toEqual([[question, 1]]);
  });

  it("should move the module into the old slave when the user says yes", async () => {
    await renderPrompt();
    const { old, slave } = await createPair();
    const prompt = await findPrompt(`Is this your old ${old.name}?`);
    fireEvent.click(await within(prompt).findByRole("button", { name: "Yes" }));
    await waitFor(async () =>
      expect(
        (
          await client.devices.retrieve({
            key: old.key,
            schemas: EtherCAT.Device.SLAVE_SCHEMAS,
          })
        ).properties,
      ).toMatchObject({ serial: 77, position: 5 }),
    );
    await waitFor(async () => {
      await expect(client.devices.retrieve({ key: slave.key })).rejects.toThrow(
        NotFoundError,
      );
    });
  });

  it("should keep both slaves when the user says no", async () => {
    await renderPrompt();
    const { old, slave } = await createPair();
    const question = `Is this your old ${old.name}?`;
    const prompt = await findPrompt(question);
    fireEvent.click(await within(prompt).findByRole("button", { name: "No" }));
    await waitFor(() => expect(screen.queryByText(question)).toBeNull());
    expect((await client.devices.retrieve({ key: slave.key })).key).toEqual(slave.key);
    const kept = await client.devices.retrieve({
      key: old.key,
      schemas: EtherCAT.Device.SLAVE_SCHEMAS,
    });
    expect(kept.properties.position).toEqual(3);
  });

  it("should name each old slave by position when several match", async () => {
    await renderPrompt();
    const { third, fourth, slave } = await createTwoOld();
    expect(await findAnswers(slave.name)).toEqual([
      `${third.name} at position 3`,
      `${fourth.name} at position 4`,
    ]);
    fireEvent.click(
      screen.getByRole("button", { name: `${fourth.name} at position 4` }),
    );
    await waitFor(async () =>
      expect(
        (
          await client.devices.retrieve({
            key: fourth.key,
            schemas: EtherCAT.Device.SLAVE_SCHEMAS,
          })
        ).properties,
      ).toMatchObject({ serial: 77, position: 5 }),
    );
    const kept = await client.devices.retrieve({
      key: third.key,
      schemas: EtherCAT.Device.SLAVE_SCHEMAS,
    });
    expect(kept.properties.position).toEqual(3);
  });

  it("should keep naming the old slave that is left when another is back on the bus", async () => {
    await renderPrompt();
    const { third, fourth, slave } = await createTwoOld();
    expect(await findAnswers(slave.name)).toHaveLength(2);
    await createReachedStatus(client, third);
    await waitFor(async () =>
      expect(await findAnswers(slave.name)).toEqual([`${fourth.name} at position 4`]),
    );
  });

  it("should offer no answer once the old module is back on the bus", async () => {
    await renderPrompt();
    const { old } = await createPair();
    const prompt = await findPrompt(`Is this your old ${old.name}?`);
    await within(prompt).findByRole("button", { name: "Yes" });
    await createReachedStatus(client, old);
    await waitFor(() =>
      expect(within(prompt).queryByRole("button", { name: "Yes" })).toBeNull(),
    );
  });

  it("should offer no answer once the user configures the new slave", async () => {
    await renderPrompt();
    const { old, slave } = await createPair();
    const prompt = await findPrompt(`Is this your old ${old.name}?`);
    await within(prompt).findByRole("button", { name: "Yes" });
    await client.devices.dispatch(
      slave.key,
      device.set({ path: ["configured"], value: true }),
    );
    await waitFor(() =>
      expect(within(prompt).queryByRole("button", { name: "Yes" })).toBeNull(),
    );
  });
});
