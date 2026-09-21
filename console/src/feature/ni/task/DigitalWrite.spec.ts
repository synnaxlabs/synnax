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
import { Text } from "@synnaxlabs/pluto";
import { id } from "@synnaxlabs/x";
import { screen, waitFor } from "@testing-library/react";
import { act } from "react";
import { describe, expect, it } from "vitest";

import { NI } from "@/feature/ni";
import { createNIDevice, renderNITaskForm } from "@/feature/ni/task/testutil";
import { Task } from "@/platform/task";
import { commitFieldInput, deployAndAwaitTask } from "@/platform/task/testutil";
import { awaitTextEditing, commitTextEdit, uniqueName } from "@/testutil";

const client = createTestClient();

interface CreateChannelOverrides extends Partial<NI.Task.DOChannel> {}

const createChannel = (
  port: number,
  line: number,
  overrides: CreateChannelOverrides = {},
): NI.Task.DOChannel => ({
  ...NI.Task.createDOChannel(),
  key: id.create(),
  port,
  line,
  ...overrides,
});

// Drafts carry no key; the created row mints its own.
const ZERO_DRAFT: task.New<NI.Task.DigitalWriteSchemas> = {
  name: "NI digital write task",
  type: NI.Task.DIGITAL_WRITE_TYPE,
  config: NI.Task.DIGITAL_WRITE_SCHEMAS.config.parse({}),
};

const createDraft = async (
  config: task.Payload<NI.Task.DigitalWriteSchemas>["config"],
) =>
  await client.tasks.create({ ...ZERO_DRAFT, config }, NI.Task.DIGITAL_WRITE_SCHEMAS);

const renderDigitalWrite = async (
  config: task.Payload<NI.Task.DigitalWriteSchemas>["config"],
) => {
  const draft = await createDraft(config);
  const rendered = await renderNITaskForm(NI.Task.DigitalWrite, {
    client,
    taskKey: draft.key,
  });
  return { ...rendered, draft };
};

const createConfig = (
  channels: NI.Task.DOChannel[],
  device = "placeholder_device",
) => ({ ...NI.Task.DIGITAL_WRITE_SCHEMAS.config.parse({}), device, channels });

describe("DigitalWrite", () => {
  it("should write edits to a channel's line number back into the form", async () => {
    await renderDigitalWrite(
      createConfig([
        createChannel(0, 0, { cmdChannelName: "cmd_a", stateChannelName: "state_a" }),
        createChannel(0, 1, { cmdChannelName: "cmd_b", stateChannelName: "state_b" }),
      ]),
    );
    await waitFor(() => expect(screen.getByText("cmd_b")).toBeTruthy());
    commitFieldInput(screen.getByDisplayValue("1"), "5");
    await waitFor(() => expect(screen.getByDisplayValue("5")).toBeTruthy());
    expect(screen.queryByDisplayValue("1")).toBeNull();
  });

  it("should create per-line command and state channels keyed by port and line", async () => {
    const dev = await createNIDevice(client);
    const rendered = await renderDigitalWrite(
      createConfig([createChannel(0, 0), createChannel(0, 1)], dev.key),
    );
    await deployAndAwaitTask(client, rendered.container, rendered.draft.key);
    const created = await client.tasks.retrieve({
      key: rendered.draft.key,
      schemas: NI.Task.DIGITAL_WRITE_SCHEMAS,
    });
    expect(created.type).toBe(NI.Task.DIGITAL_WRITE_TYPE);
    expect(created.rack).toBe(dev.rack);
    const [c0, c1] = created.config.channels;
    expect(c0.cmdChannel).not.toBe(0);
    expect(c1.stateChannel).not.toBe(0);

    const identifier = dev.properties.identifier;
    const cmd = await client.channels.retrieve(c0.cmdChannel);
    expect(cmd.name).toBe(`${identifier}_do_0_0_cmd`);
    const state = await client.channels.retrieve(c1.stateChannel);
    expect(state.name).toBe(`${identifier}_do_0_1_state`);

    const updated = await client.devices.retrieve({
      key: dev.key,
      schemas: NI.Device.SCHEMAS,
    });
    expect(updated.properties.digitalOutput.channels["0l0"]).toEqual({
      command: c0.cmdChannel,
      state: c0.stateChannel,
    });
    expect(updated.properties.digitalOutput.channels["0l1"]).toEqual({
      command: c1.cmdChannel,
      state: c1.stateChannel,
    });
    const stateIndex = await client.channels.retrieve(
      updated.properties.digitalOutput.stateIndex,
    );
    expect(stateIndex.name).toBe(`${identifier}_do_state_time`);
    expect(stateIndex.isIndex).toBe(true);
  });

  // A row still showing the channel its old line mapped would rename that channel
  // instead of the one the row now names.
  it("should unbind a row moved to a line the device maps nothing for", async () => {
    const dev = await createNIDevice(client);
    const rendered = await renderDigitalWrite(
      createConfig([createChannel(0, 0)], dev.key),
    );
    const deployed = await deployAndAwaitTask(
      client,
      rendered.container,
      rendered.draft.key,
      NI.Task.DIGITAL_WRITE_SCHEMAS,
    );
    const cmd = await client.channels.retrieve(deployed.config.channels[0].cmdChannel);
    await screen.findByText(cmd.name);
    const inputs = rendered.container.querySelectorAll("input");
    commitFieldInput(inputs[inputs.length - 1], "7");
    await waitFor(async () => {
      const saved = await client.tasks.retrieve({
        key: rendered.draft.key,
        schemas: NI.Task.DIGITAL_WRITE_SCHEMAS,
      });
      expect(saved.config.channels[0].line).toBe(7);
      expect(saved.config.channels[0].cmdChannel).toBe(0);
      expect(saved.config.channels[0].stateChannel).toBe(0);
    });
    expect(screen.queryByText(cmd.name)).toBeNull();
  });

  it("should use custom command and state channel names when provided", async () => {
    const dev = await createNIDevice(client);
    const cmdName = uniqueName("do_cmd");
    const stateName = uniqueName("do_state");
    const rendered = await renderDigitalWrite(
      createConfig(
        [
          createChannel(0, 0, {
            cmdChannelName: cmdName,
            stateChannelName: stateName,
          }),
        ],
        dev.key,
      ),
    );
    await deployAndAwaitTask(client, rendered.container, rendered.draft.key);
    const created = await client.tasks.retrieve({
      key: rendered.draft.key,
      schemas: NI.Task.DIGITAL_WRITE_SCHEMAS,
    });
    const [c0] = created.config.channels;
    const cmd = await client.channels.retrieve(c0.cmdChannel);
    expect(cmd.name).toBe(cmdName);
    const state = await client.channels.retrieve(c0.stateChannel);
    expect(state.name).toBe(stateName);
    const cmdIndex = await client.channels.retrieve(cmd.index);
    expect(cmdIndex.name).toBe(`${cmdName}_time`);
  });

  it("should bind a new entry to the channel the device already maps, so a rename reaches the Core", async () => {
    const dev = await createNIDevice(client);
    const first = await renderDigitalWrite(
      createConfig([createChannel(0, 0)], dev.key),
    );
    await deployAndAwaitTask(client, first.container, first.draft.key);
    const firstTask = await client.tasks.retrieve({
      key: first.draft.key,
      schemas: NI.Task.DIGITAL_WRITE_SCHEMAS,
    });
    first.unmount();
    const [bound] = firstTask.config.channels;
    const cmd = await client.channels.retrieve(bound.cmdChannel);
    const state = await client.channels.retrieve(bound.stateChannel);
    const ch = createChannel(0, 0);
    const second = await renderDigitalWrite(createConfig([ch], dev.key));
    await screen.findByText(cmd.name);
    await screen.findByText(state.name);
    await waitFor(async () => {
      const saved = await client.tasks.retrieve({
        key: second.draft.key,
        schemas: NI.Task.DIGITAL_WRITE_SCHEMAS,
      });
      expect(saved.config.channels[0].cmdChannel).toBe(bound.cmdChannel);
      expect(saved.config.channels[0].stateChannel).toBe(bound.stateChannel);
    });
    const renamed = uniqueName("valve_cmd");
    const editID = Task.getChannelNameID(ch.key, "cmd");
    Text.edit(editID);
    const el = await awaitTextEditing(editID);
    act(() => commitTextEdit(el, renamed));
    await waitFor(async () => {
      const after = await client.channels.retrieve(bound.cmdChannel);
      expect(after.name).toBe(renamed);
    });
  });

  it("should reuse existing channels when redeployed", async () => {
    const dev = await createNIDevice(client);
    const config = createConfig([createChannel(0, 0)], dev.key);
    const first = await renderDigitalWrite(config);
    await deployAndAwaitTask(client, first.container, first.draft.key);
    const firstTask = await client.tasks.retrieve({
      key: first.draft.key,
      schemas: NI.Task.DIGITAL_WRITE_SCHEMAS,
    });
    first.unmount();
    const second = await renderDigitalWrite(config);
    await deployAndAwaitTask(client, second.container, second.draft.key);
    await waitFor(async () => {
      const again = await client.tasks.retrieve({
        key: second.draft.key,
        schemas: NI.Task.DIGITAL_WRITE_SCHEMAS,
      });
      expect(again.config.channels[0].cmdChannel).toBe(
        firstTask.config.channels[0].cmdChannel,
      );
    });
  });
});
