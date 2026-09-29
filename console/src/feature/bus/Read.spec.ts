// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type bus, can } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { fireEvent, screen, waitFor, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import {
  createBusDevice,
  createBusLibrary,
  createBusTask,
  createMessage,
  findOpenDialog,
  renderBusTask,
} from "@/feature/bus/testutil";
import { CAN } from "@/feature/can";
import {
  clickDeploy,
  deployAndAwaitTask,
  selectFromDropdown,
} from "@/platform/task/testutil";
import { assertDefined, findDialogTriggerByText } from "@/testutil";

const client = createTestClient();

const createCANDevice = async () =>
  await createBusDevice(client, CAN.Device.MAKE, can.propertiesZ.parse({}));

const createDraft = async (config: Record<string, unknown>) =>
  await createBusTask(
    client,
    CAN.Task.READ_TYPE,
    CAN.Task.READ_SCHEMAS.config.parse(config),
  );

const retrieveRead = async (key: string) =>
  await client.tasks.retrieve({ key, schemas: CAN.Task.READ_SCHEMAS });

/** Expects the index and field channels of a deployed message to hold the names. */
const expectBound = async (m: bus.ReadMessage, index: string, fields: string[]) => {
  expect((await client.channels.retrieve(m.index)).name).toBe(index);
  const channels = await client.channels.retrieve(m.fields.map((f) => f.channel));
  expect(channels.map((c) => c.name).sort()).toEqual(fields);
  channels.forEach((c) => expect(c.index).toBe(m.index));
};

describe("bus read task", () => {
  it("should create one index per message and one channel per field on deploy", async () => {
    const { library, engine } = await createBusLibrary(client);
    const dev = await createCANDevice();
    const draft = await createDraft({ library: library.key, device: dev.key });
    const { container } = await renderBusTask(CAN.Task.Read, client, draft.key);
    await selectFromDropdown("Add message", "Engine");
    await screen.findByRole("checkbox", { name: "Rpm" });

    const deployed = await deployAndAwaitTask(
      client,
      container,
      draft.key,
      CAN.Task.READ_SCHEMAS,
    );
    const [msg] = deployed.config.messages;
    expect(msg.message).toBe(engine.key);
    expect(msg.fields.map((f) => f.field)).toEqual(
      engine.payload.fields.map((f) => f.key),
    );
    const index = await client.channels.retrieve(msg.index);
    expect(index.name).toBe(`${dev.name}_Engine_time`);
    expect(index.isIndex).toBe(true);
    const fields = await client.channels.retrieve(msg.fields.map((f) => f.channel));
    expect(fields.map((c) => c.name).sort()).toEqual([
      `${dev.name}_Engine_Rpm`,
      `${dev.name}_Engine_Temp`,
    ]);
    fields.forEach((c) => expect(c.index).toBe(index.key));
    const raw = await client.channels.retrieve(deployed.config.raw);
    expect(raw.name).toBe(`${dev.name}_raw`);
    expect(raw.virtual).toBe(true);
    expect(raw.dataType.toString()).toBe("bytes");
    expect(deployed.rack).toBe(dev.rack);
  });

  it("should bind the fields of each message to the index of that message", async () => {
    const { library, engine, brake } = await createBusLibrary(client);
    const dev = await createCANDevice();
    const draft = await createDraft({
      library: library.key,
      device: dev.key,
      messages: [createMessage(engine), createMessage(brake)],
    });
    const { container } = await renderBusTask(CAN.Task.Read, client, draft.key);
    await screen.findByRole("checkbox", { name: "Rpm" });
    const deployed = await deployAndAwaitTask(
      client,
      container,
      draft.key,
      CAN.Task.READ_SCHEMAS,
    );
    const [engineMsg, brakeMsg] = deployed.config.messages;
    await expectBound(engineMsg, `${dev.name}_Engine_time`, [
      `${dev.name}_Engine_Rpm`,
      `${dev.name}_Engine_Temp`,
    ]);
    await expectBound(brakeMsg, `${dev.name}_Brake_time`, [
      `${dev.name}_Brake_Pressure`,
    ]);
  });

  it("should only offer the library messages the integration can carry", async () => {
    const { library } = await createBusLibrary(client);
    const dev = await createCANDevice();
    const draft = await createDraft({ library: library.key, device: dev.key });
    await renderBusTask(CAN.Task.Read, client, draft.key);
    fireEvent.click(await findDialogTriggerByText("Add message"));
    const dialog = await findOpenDialog();
    await within(dialog).findByText("Engine");
    expect(within(dialog).queryByText("Status")).toBeNull();
  });

  it("should bind only the checked fields of a message", async () => {
    const { library, engine } = await createBusLibrary(client);
    const dev = await createCANDevice();
    const draft = await createDraft({ library: library.key, device: dev.key });
    const { container } = await renderBusTask(CAN.Task.Read, client, draft.key);
    await selectFromDropdown("Add message", "Engine");
    fireEvent.click(await screen.findByRole("checkbox", { name: "Temp" }));

    const deployed = await deployAndAwaitTask(
      client,
      container,
      draft.key,
      CAN.Task.READ_SCHEMAS,
    );
    const [msg] = deployed.config.messages;
    expect(msg.fields.map((f) => f.field)).toEqual([engine.payload.fields[0].key]);
    expect(await client.channels.retrieve([`${dev.name}_Engine_Temp`])).toHaveLength(0);
  });

  it("should reuse the channels of an earlier deploy", async () => {
    const { library, engine } = await createBusLibrary(client);
    const dev = await createCANDevice();
    const first = await createDraft({
      library: library.key,
      device: dev.key,
      messages: [
        { message: engine.key, fields: [{ field: engine.payload.fields[0].key }] },
      ],
    });
    const firstForm = await renderBusTask(CAN.Task.Read, client, first.key);
    const deployed = await deployAndAwaitTask(
      client,
      firstForm.container,
      first.key,
      CAN.Task.READ_SCHEMAS,
    );
    firstForm.unmount();
    const second = await createDraft({
      library: library.key,
      device: dev.key,
      messages: [
        { message: engine.key, fields: [{ field: engine.payload.fields[0].key }] },
      ],
    });
    const secondForm = await renderBusTask(CAN.Task.Read, client, second.key);
    const redeployed = await deployAndAwaitTask(
      client,
      secondForm.container,
      second.key,
      CAN.Task.READ_SCHEMAS,
    );
    expect(redeployed.config.messages[0].index).toBe(deployed.config.messages[0].index);
    expect(redeployed.config.messages[0].fields[0].channel).toBe(
      deployed.config.messages[0].fields[0].channel,
    );
    expect(redeployed.config.raw).toBe(deployed.config.raw);
  });

  it("should remove the messages of the old library when the library changes", async () => {
    const first = await createBusLibrary(client);
    const second = await createBusLibrary(client);
    const dev = await createCANDevice();
    const draft = await createDraft({
      library: first.library.key,
      device: dev.key,
      messages: [{ message: first.engine.key }],
    });
    await renderBusTask(CAN.Task.Read, client, draft.key);
    await screen.findByText("0x100");
    fireEvent.click(await findDialogTriggerByText(first.library.name));
    fireEvent.change(await screen.findByPlaceholderText("Search libraries..."), {
      target: { value: second.library.name },
    });
    fireEvent.click(await screen.findByText(second.library.name));
    await waitFor(async () => {
      const saved = await retrieveRead(draft.key);
      expect(saved.config.library).toBe(second.library.key);
      expect(saved.config.messages).toEqual([]);
    });
  });

  it("should save the library chosen for a task with none", async () => {
    const { library } = await createBusLibrary(client);
    const draft = await createDraft({});
    await renderBusTask(CAN.Task.Read, client, draft.key);
    fireEvent.click(await findDialogTriggerByText("Select library"));
    fireEvent.change(await screen.findByPlaceholderText("Search libraries..."), {
      target: { value: library.name },
    });
    fireEvent.click(await screen.findByText(library.name));
    await waitFor(async () =>
      expect((await retrieveRead(draft.key)).config.library).toBe(library.key),
    );
  });

  it("should remove a message from the task", async () => {
    const { library, engine } = await createBusLibrary(client);
    const dev = await createCANDevice();
    const draft = await createDraft({
      library: library.key,
      device: dev.key,
      messages: [{ message: engine.key }],
    });
    await renderBusTask(CAN.Task.Read, client, draft.key);
    fireEvent.click(await screen.findByRole("button", { name: "Remove message" }));
    await waitFor(async () =>
      expect((await retrieveRead(draft.key)).config.messages).toEqual([]),
    );
  });

  describe("validation", () => {
    it("should show a missing library on the library field", async () => {
      const dev = await createCANDevice();
      const draft = await createDraft({ device: dev.key });
      const { container } = await renderBusTask(CAN.Task.Read, client, draft.key);
      await clickDeploy(container);
      const help = await screen.findByText("Select a library");
      const item = help.closest<HTMLElement>(".pluto-input__item");
      assertDefined(item);
      expect(within(item).getByText("Library")).toBeDefined();
    });

    it("should show a missing device on the device field", async () => {
      const { library } = await createBusLibrary(client);
      const draft = await createDraft({ library: library.key });
      const { container } = await renderBusTask(CAN.Task.Read, client, draft.key);
      await clickDeploy(container);
      await screen.findByText("Device is required");
    });

    it("should show a task with no messages on the message picker", async () => {
      const { library } = await createBusLibrary(client);
      const dev = await createCANDevice();
      const draft = await createDraft({ library: library.key, device: dev.key });
      const { container } = await renderBusTask(CAN.Task.Read, client, draft.key);
      await clickDeploy(container);
      await screen.findByText("Add at least one enabled message");
    });

    it("should show a message with no fields on its field list", async () => {
      const { library, engine } = await createBusLibrary(client);
      const dev = await createCANDevice();
      const draft = await createDraft({
        library: library.key,
        device: dev.key,
        messages: [{ message: engine.key }],
      });
      const { container } = await renderBusTask(CAN.Task.Read, client, draft.key);
      await screen.findByRole("checkbox", { name: "Rpm" });
      await clickDeploy(container);
      await screen.findByText("Select at least one field");
    });
  });
});
