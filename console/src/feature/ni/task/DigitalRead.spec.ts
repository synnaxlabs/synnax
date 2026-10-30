// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type task } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { id } from "@synnaxlabs/x";
import { fireEvent, screen, waitFor, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { NI } from "@/feature/ni";
import {
  createHardware,
  createNIDevice,
  renderNITaskForm,
} from "@/feature/ni/task/testutil";
import {
  commitFieldInput,
  createWithHook,
  deployAndAwaitTask,
} from "@/platform/task/testutil";
import { getIconButton, uniqueName } from "@/testutil";

const client = createTestClient();

interface CreateChannelOverrides extends Partial<NI.Task.DIChannel> {}

const createChannel = (
  port: number,
  line: number,
  overrides: CreateChannelOverrides = {},
): NI.Task.DIChannel => ({
  ...NI.Task.createDIChannel(),
  key: id.create(),
  port,
  line,
  ...overrides,
});

// Drafts carry no key; the created row mints its own.
const ZERO_DRAFT: task.New<NI.Task.DigitalReadSchemas> = {
  name: "NI digital read task",
  type: NI.Task.DIGITAL_READ_TYPE,
  config: NI.Task.DIGITAL_READ_SCHEMAS.config.parse({}),
};

const createDraft = async (
  config: task.Payload<NI.Task.DigitalReadSchemas>["config"],
) => await client.tasks.create({ ...ZERO_DRAFT, config }, NI.Task.DIGITAL_READ_SCHEMAS);

const renderDigitalRead = async (
  config: task.Payload<NI.Task.DigitalReadSchemas>["config"],
) => {
  const draft = await createDraft(config);
  const rendered = await renderNITaskForm(NI.Task.DigitalRead, {
    client,
    taskKey: draft.key,
  });
  return { ...rendered, draft };
};

const createConfig = (
  channels: NI.Task.DIChannel[],
  device = "placeholder_device",
) => ({ ...NI.Task.DIGITAL_READ_SCHEMAS.config.parse({}), device, channels });

describe("DigitalRead", () => {
  it("should write edits to a channel's line number back into the form", async () => {
    await renderDigitalRead(
      createConfig([
        createChannel(0, 0, { name: "di_chan_a" }),
        createChannel(0, 1, { name: "di_chan_b" }),
      ]),
    );
    await waitFor(() => expect(screen.getByText("di_chan_b")).toBeTruthy());
    commitFieldInput(screen.getByDisplayValue("1"), "7");
    await waitFor(() => expect(screen.getByDisplayValue("7")).toBeTruthy());
    expect(screen.queryByDisplayValue("1")).toBeNull();
  });

  it("should create per-line channels keyed by port and line and update the device", async () => {
    const dev = await createNIDevice(client);
    const namedChannel = uniqueName("di_named");
    const rendered = await renderDigitalRead(
      createConfig(
        [createChannel(0, 0), createChannel(0, 1, { name: namedChannel })],
        dev.key,
      ),
    );
    await deployAndAwaitTask(client, rendered.container, rendered.draft.key);
    const created = await client.tasks.retrieve({
      key: rendered.draft.key,
      schemas: NI.Task.DIGITAL_READ_SCHEMAS,
    });
    expect(created.type).toBe(NI.Task.DIGITAL_READ_TYPE);
    expect(created.rack).toBe(dev.rack);
    const [c0, c1] = created.config.channels;
    expect(c0.channel).not.toBe(0);
    expect(c1.channel).not.toBe(0);

    const identifier = dev.properties.identifier;
    const defaultNamed = await client.channels.retrieve(c0.channel);
    expect(defaultNamed.name).toBe(`${identifier}_di_0_0`);
    const named = await client.channels.retrieve(c1.channel);
    expect(named.name).toBe(namedChannel);

    const updated = await client.devices.retrieve({
      key: dev.key,
      schemas: NI.Device.SCHEMAS,
    });
    expect(updated.properties.digitalInput.channels["0l0"]).toBe(c0.channel);
    expect(updated.properties.digitalInput.channels["0l1"]).toBe(c1.channel);
    const index = await client.channels.retrieve(updated.properties.digitalInput.index);
    expect(index.name).toBe(`${identifier}_di_time`);
    expect(index.isIndex).toBe(true);
  });

  it("should bind a new entry to the channel the device already maps", async () => {
    const dev = await createNIDevice(client);
    const config = createConfig([createChannel(0, 0)], dev.key);
    const first = await renderDigitalRead(config);
    await deployAndAwaitTask(client, first.container, first.draft.key);
    const firstTask = await client.tasks.retrieve({
      key: first.draft.key,
      schemas: NI.Task.DIGITAL_READ_SCHEMAS,
    });
    first.unmount();
    const existing = await client.channels.retrieve(
      firstTask.config.channels[0].channel,
    );
    const second = await renderDigitalRead(config);
    await screen.findByText(existing.name);
    await waitFor(async () => {
      const saved = await client.tasks.retrieve({
        key: second.draft.key,
        schemas: NI.Task.DIGITAL_READ_SCHEMAS,
      });
      expect(saved.config.channels[0].channel).toBe(existing.key);
    });
  });

  it("should reuse existing channels when redeployed", async () => {
    const dev = await createNIDevice(client);
    const config = createConfig([createChannel(0, 0)], dev.key);
    const first = await renderDigitalRead(config);
    await deployAndAwaitTask(client, first.container, first.draft.key);
    const firstTask = await client.tasks.retrieve({
      key: first.draft.key,
      schemas: NI.Task.DIGITAL_READ_SCHEMAS,
    });
    first.unmount();
    const second = await renderDigitalRead(config);
    await deployAndAwaitTask(client, second.container, second.draft.key);
    await waitFor(async () => {
      const again = await client.tasks.retrieve({
        key: second.draft.key,
        schemas: NI.Task.DIGITAL_READ_SCHEMAS,
      });
      expect(again.config.channels[0].channel).toBe(
        firstTask.config.channels[0].channel,
      );
    });
  });

  it("should create the task with one channel", async () => {
    const tsk = await createWithHook(
      NI.Task.useCreateDigitalRead,
      NI.Task.DIGITAL_READ_SCHEMAS,
    );
    expect(tsk.config.channels).toHaveLength(1);
  });
});

describe("DigitalRead ports", () => {
  it("should add a channel on a free line from the ports view", async () => {
    const dev = await createNIDevice(client, {
      properties: { hardware: createHardware() },
    });
    const { draft } = await renderDigitalRead(
      createConfig([createChannel(0, 0)], dev.key),
    );
    fireEvent.click(await screen.findByText("Ports"));
    const grid = await screen.findByRole("group", { name: "port1 lines" });
    fireEvent.click(within(grid).getByRole("button", { name: "port1/line2" }));
    await waitFor(async () => {
      const saved = await client.tasks.retrieve({
        key: draft.key,
        schemas: NI.Task.DIGITAL_READ_SCHEMAS,
      });
      expect(saved.config.channels.map(({ port, line }) => [port, line])).toEqual([
        [0, 0],
        [1, 2],
      ]);
    });
  });

  it("should add a channel on the next port once each line of a port holds one", async () => {
    const dev = await createNIDevice(client, {
      properties: { hardware: createHardware() },
    });
    const { container, draft } = await renderDigitalRead(
      createConfig(
        Array.from({ length: 8 }, (_, line) => createChannel(0, line)),
        dev.key,
      ),
    );
    await screen.findByText("Ports");
    fireEvent.click(getIconButton(container, "add"));
    await waitFor(async () => {
      const saved = await client.tasks.retrieve({
        key: draft.key,
        schemas: NI.Task.DIGITAL_READ_SCHEMAS,
      });
      expect(
        saved.config.channels.map(({ port, line }) => [port, line]).at(-1),
      ).toEqual([1, 0]);
    });
  });

  it("should mark a line that two channels share", async () => {
    const dev = await createNIDevice(client, {
      properties: { hardware: createHardware() },
    });
    await renderDigitalRead(
      createConfig([createChannel(0, 2), createChannel(0, 2)], dev.key),
    );
    fireEvent.click(await screen.findByText("Ports"));
    const grid = await screen.findByRole("group", { name: "port0 lines" });
    const line = within(grid).getByRole("button", { name: "port0/line2" });
    expect(line.className).toContain("console-port-grid__cell--conflict");
  });
});

describe("DigitalRead disabled channels", () => {
  it("should show no conflict for a line a disabled channel shares, and redeploy", async () => {
    const dev = await createNIDevice(client, {
      properties: { hardware: createHardware() },
    });
    const first = await renderDigitalRead(
      createConfig(
        [createChannel(0, 2, { disabled: true }), createChannel(0, 2)],
        dev.key,
      ),
    );
    fireEvent.click(await screen.findByText("Ports"));
    const grid = await screen.findByRole("group", { name: "port0 lines" });
    const line = within(grid).getByRole("button", { name: "port0/line2" });
    expect(line.className).toContain("console-port-grid__cell--used");
    expect(line.className).not.toContain("console-port-grid__cell--conflict");
    const { config } = await deployAndAwaitTask(
      client,
      first.container,
      first.draft.key,
      NI.Task.DIGITAL_READ_SCHEMAS,
    );
    expect(config.channels[1].channel).toBe(config.channels[0].channel);
    first.unmount();
    const second = await renderDigitalRead(config);
    await deployAndAwaitTask(client, second.container, second.draft.key);
  });

  it("should show a line a disabled channel holds as disabled", async () => {
    const dev = await createNIDevice(client, {
      properties: { hardware: createHardware() },
    });
    await renderDigitalRead(
      createConfig(
        [createChannel(0, 0), createChannel(1, 4, { disabled: true })],
        dev.key,
      ),
    );
    fireEvent.click(await screen.findByText("Ports"));
    const grid = await screen.findByRole("group", { name: "port1 lines" });
    const line = within(grid).getByRole("button", { name: "port1/line4" });
    expect(line.className).toContain("console-port-grid__cell--disabled");
  });
});
