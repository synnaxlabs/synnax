// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { task } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { id } from "@synnaxlabs/x";
import { screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { NI } from "@/feature/ni";
import { createNIDevice, renderNITaskForm } from "@/feature/ni/task/testutil";
import { awaitCommand, clickDeploy, commitFieldInput } from "@/platform/task/testutil";
import { uniqueName } from "@/testutil";

const client = createTestClient();

interface CreateChannelOverrides extends Partial<NI.Task.DOChannel> {}

const createChannel = (
  port: number,
  line: number,
  overrides: CreateChannelOverrides = {},
): NI.Task.DOChannel => ({
  ...NI.Task.ZERO_DO_CHANNEL,
  key: id.create(),
  port,
  line,
  ...overrides,
});

// Draft creates mint their own key; the zero payload's empty key must not be sent.
const { key: _key, ...ZERO_DRAFT } = NI.Task.ZERO_DIGITAL_WRITE_PAYLOAD;

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

const deployAndAwaitStart = async (
  container: ParentNode,
  key: task.Key,
): Promise<void> => {
  const streamer = await client.openStreamer(task.COMMAND_CHANNEL_NAME);
  try {
    await clickDeploy(container);
    await awaitCommand(streamer, key);
  } finally {
    streamer.close();
  }
};

const createConfig = (
  channels: NI.Task.DOChannel[],
  device = "placeholder_device",
) => ({ ...NI.Task.ZERO_DIGITAL_WRITE_PAYLOAD.config, device, channels });

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

  describe("deploying against a live cluster", () => {
    it("should create per-line command and state channels keyed by port and line", async () => {
      const dev = await createNIDevice(client);
      const rendered = await renderDigitalWrite(
        createConfig([createChannel(0, 0), createChannel(0, 1)], dev.key),
      );
      await deployAndAwaitStart(rendered.container, rendered.draft.key);
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
      await deployAndAwaitStart(rendered.container, rendered.draft.key);
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

    it("should reuse existing channels when redeployed", async () => {
      const dev = await createNIDevice(client);
      const config = createConfig([createChannel(0, 0)], dev.key);
      const first = await renderDigitalWrite(config);
      await deployAndAwaitStart(first.container, first.draft.key);
      const firstTask = await client.tasks.retrieve({
        key: first.draft.key,
        schemas: NI.Task.DIGITAL_WRITE_SCHEMAS,
      });
      first.unmount();
      const second = await renderDigitalWrite(config);
      await deployAndAwaitStart(second.container, second.draft.key);
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
});
