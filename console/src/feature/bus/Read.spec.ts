// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type bus, can, type icd } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { uuid } from "@synnaxlabs/x";
import { fireEvent, screen, waitFor, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import {
  createBusDevice,
  createBusICD,
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
import { assertDefined, findDialogTriggerByText, uniqueName } from "@/testutil";

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

/**
 * Expects each field of a deployed message to write to the channel named prefix and the
 * field name, on the index named in indexes for the field. Fields with the same index
 * name must share one index channel.
 */
const expectBound = async (
  m: bus.ReadMessage,
  entry: icd.MessageEntry,
  prefix: string,
  indexes: Record<string, string>,
) => {
  const names = new Map(entry.payload.fields.map((f) => [f.key, f.name]));
  const bound = await Promise.all(
    m.fields.map(async (f) => {
      const [ch, index] = await Promise.all([
        client.channels.retrieve(f.channel),
        client.channels.retrieve(f.index),
      ]);
      const name = names.get(f.field) ?? "";
      expect(ch.name).toBe(`${prefix}_${name}`);
      expect(ch.index).toBe(index.key);
      expect(index.isIndex).toBe(true);
      return { name, index };
    }),
  );
  expect(Object.fromEntries(bound.map((b) => [b.name, b.index.name]))).toEqual(indexes);
  expect(new Set(bound.map((b) => b.index.key)).size).toBe(
    new Set(Object.values(indexes)).size,
  );
};

/**
 * Creates an ICD with two multiplexed CAN messages. In Mux, Mode selects A and B on
 * 0 and C and D on 1 or 2. In Ext, Mode selects Sub and F on 3, and Sub selects E on 5.
 * Mode and Speed are always present.
 */
const createMuxICD = async () => {
  const [muxMode, extMode, sub] = [uuid.create(), uuid.create(), uuid.create()];
  const ic = await client.icds.create({
    name: uniqueName("mux_icd"),
    entries: [
      {
        kind: "message",
        name: "Mux",
        payload: {
          format: "binary",
          identifier: { type: "can", id: 0x300, extended: false, fd: false },
          length: 8,
          fields: [
            { key: muxMode, name: "Mode", startBit: 0 },
            { name: "Speed", startBit: 8 },
            { name: "A", startBit: 16, multiplexor: muxMode, multiplexValues: [0] },
            { name: "B", startBit: 24, multiplexor: muxMode, multiplexValues: [0] },
            { name: "C", startBit: 16, multiplexor: muxMode, multiplexValues: [1, 2] },
            { name: "D", startBit: 24, multiplexor: muxMode, multiplexValues: [2, 1] },
          ],
        },
      },
      {
        kind: "message",
        name: "Ext",
        payload: {
          format: "binary",
          identifier: { type: "can", id: 0x301, extended: false, fd: false },
          length: 8,
          fields: [
            { key: extMode, name: "Mode", startBit: 0 },
            {
              key: sub,
              name: "Sub",
              startBit: 8,
              multiplexor: extMode,
              multiplexValues: [3],
            },
            { name: "F", startBit: 16, multiplexor: extMode, multiplexValues: [3] },
            { name: "E", startBit: 24, multiplexor: sub, multiplexValues: [5] },
          ],
        },
      },
    ],
  });
  const find = (name: string) => {
    const entry = ic.entries.find((e) => e.name === name);
    if (entry?.kind !== "message") throw new Error(`message ${name} not found`);
    return entry;
  };
  return { icd: ic, mux: find("Mux"), ext: find("Ext") };
};

describe("bus read task", () => {
  it("should put every field of a plain message on one index on deploy", async () => {
    const { icd: ic, engine } = await createBusICD(client);
    const dev = await createCANDevice();
    const draft = await createDraft({ icd: ic.key, device: dev.key });
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
    const index = await client.channels.retrieve(msg.fields[0].index);
    expect(index.name).toBe(`${dev.name}_Engine_time`);
    expect(index.isIndex).toBe(true);
    msg.fields.forEach((f) => expect(f.index).toBe(index.key));
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
    const { icd: ic, engine, brake } = await createBusICD(client);
    const dev = await createCANDevice();
    const draft = await createDraft({
      icd: ic.key,
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
    await expectBound(engineMsg, engine, `${dev.name}_Engine`, {
      Rpm: `${dev.name}_Engine_time`,
      Temp: `${dev.name}_Engine_time`,
    });
    await expectBound(brakeMsg, brake, `${dev.name}_Brake`, {
      Pressure: `${dev.name}_Brake_time`,
    });
  });

  it("should create one index per set of multiplexor values on deploy", async () => {
    const { icd: ic, mux } = await createMuxICD();
    const dev = await createCANDevice();
    const draft = await createDraft({
      icd: ic.key,
      device: dev.key,
      messages: [createMessage(mux)],
    });
    const { container } = await renderBusTask(CAN.Task.Read, client, draft.key);
    const prefix = `${dev.name}_Mux`;
    await screen.findByText(`Creates ${prefix}_mux1_2_time`);
    const deployed = await deployAndAwaitTask(
      client,
      container,
      draft.key,
      CAN.Task.READ_SCHEMAS,
    );
    await expectBound(deployed.config.messages[0], mux, prefix, {
      Mode: `${prefix}_time`,
      Speed: `${prefix}_time`,
      A: `${prefix}_mux0_time`,
      B: `${prefix}_mux0_time`,
      C: `${prefix}_mux1_2_time`,
      D: `${prefix}_mux1_2_time`,
    });
  });

  it("should create one index per link of an extended multiplexor chain", async () => {
    const { icd: ic, ext } = await createMuxICD();
    const dev = await createCANDevice();
    const draft = await createDraft({
      icd: ic.key,
      device: dev.key,
      messages: [createMessage(ext)],
    });
    const { container } = await renderBusTask(CAN.Task.Read, client, draft.key);
    await screen.findByRole("checkbox", { name: "Sub" });
    const deployed = await deployAndAwaitTask(
      client,
      container,
      draft.key,
      CAN.Task.READ_SCHEMAS,
    );
    const prefix = `${dev.name}_Ext`;
    await expectBound(deployed.config.messages[0], ext, prefix, {
      Mode: `${prefix}_time`,
      Sub: `${prefix}_Mode_3_time`,
      F: `${prefix}_Mode_3_time`,
      E: `${prefix}_Mode_3_Sub_5_time`,
    });
  });

  it("should only offer the ICD messages the integration can carry", async () => {
    const { icd: ic } = await createBusICD(client);
    const dev = await createCANDevice();
    const draft = await createDraft({ icd: ic.key, device: dev.key });
    await renderBusTask(CAN.Task.Read, client, draft.key);
    fireEvent.click(await findDialogTriggerByText("Add message"));
    const dialog = await findOpenDialog();
    await within(dialog).findByText("Engine");
    expect(within(dialog).queryByText("Status")).toBeNull();
  });

  it("should bind only the checked fields of a message", async () => {
    const { icd: ic, engine } = await createBusICD(client);
    const dev = await createCANDevice();
    const draft = await createDraft({ icd: ic.key, device: dev.key });
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
    const { icd: ic, engine } = await createBusICD(client);
    const dev = await createCANDevice();
    const first = await createDraft({
      icd: ic.key,
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
      icd: ic.key,
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
    expect(redeployed.config.messages[0].fields[0].index).toBe(
      deployed.config.messages[0].fields[0].index,
    );
    expect(redeployed.config.messages[0].fields[0].channel).toBe(
      deployed.config.messages[0].fields[0].channel,
    );
    expect(redeployed.config.raw).toBe(deployed.config.raw);
  });

  it("should remove the messages of the old ICD when the ICD changes", async () => {
    const first = await createBusICD(client);
    const second = await createBusICD(client);
    const dev = await createCANDevice();
    const draft = await createDraft({
      icd: first.icd.key,
      device: dev.key,
      messages: [{ message: first.engine.key }],
    });
    await renderBusTask(CAN.Task.Read, client, draft.key);
    await screen.findByText("0x100");
    fireEvent.click(await findDialogTriggerByText(first.icd.name));
    fireEvent.change(await screen.findByPlaceholderText("Search ICDs..."), {
      target: { value: second.icd.name },
    });
    fireEvent.click(await screen.findByText(second.icd.name));
    await waitFor(async () => {
      const saved = await retrieveRead(draft.key);
      expect(saved.config.icd).toBe(second.icd.key);
      expect(saved.config.messages).toEqual([]);
    });
  });

  it("should save the ICD chosen for a task with none", async () => {
    const { icd: ic } = await createBusICD(client);
    const draft = await createDraft({});
    await renderBusTask(CAN.Task.Read, client, draft.key);
    fireEvent.click(await findDialogTriggerByText("Select ICD"));
    fireEvent.change(await screen.findByPlaceholderText("Search ICDs..."), {
      target: { value: ic.name },
    });
    fireEvent.click(await screen.findByText(ic.name));
    await waitFor(async () =>
      expect((await retrieveRead(draft.key)).config.icd).toBe(ic.key),
    );
  });

  it("should remove a message from the task", async () => {
    const { icd: ic, engine } = await createBusICD(client);
    const dev = await createCANDevice();
    const draft = await createDraft({
      icd: ic.key,
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
    it("should show a missing ICD on the ICD field", async () => {
      const dev = await createCANDevice();
      const draft = await createDraft({ device: dev.key });
      const { container } = await renderBusTask(CAN.Task.Read, client, draft.key);
      await clickDeploy(container);
      const help = await screen.findByText("Select an ICD");
      const item = help.closest<HTMLElement>(".pluto-input__item");
      assertDefined(item);
      expect(within(item).getByText("ICD")).toBeDefined();
    });

    it("should show a missing device on the device field", async () => {
      const { icd: ic } = await createBusICD(client);
      const draft = await createDraft({ icd: ic.key });
      const { container } = await renderBusTask(CAN.Task.Read, client, draft.key);
      await clickDeploy(container);
      await screen.findByText("Device is required");
    });

    it("should show a task with no messages on the message picker", async () => {
      const { icd: ic } = await createBusICD(client);
      const dev = await createCANDevice();
      const draft = await createDraft({ icd: ic.key, device: dev.key });
      const { container } = await renderBusTask(CAN.Task.Read, client, draft.key);
      await clickDeploy(container);
      await screen.findByText("Add at least one enabled message");
    });

    it("should show a message with no fields on its field list", async () => {
      const { icd: ic, engine } = await createBusICD(client);
      const dev = await createCANDevice();
      const draft = await createDraft({
        icd: ic.key,
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
