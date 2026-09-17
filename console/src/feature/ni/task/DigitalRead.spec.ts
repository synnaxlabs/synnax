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
import { screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { NI } from "@/feature/ni";
import { createNIDevice, renderNITaskForm } from "@/feature/ni/task/testutil";
import {
  commitFieldInput,
  createTaskStatus,
  createTestChannel,
  deployAndAwaitTask,
  isRedeployHidden,
} from "@/platform/task/testutil";
import { uniqueName } from "@/testutil";

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

  describe("deploying against a live Core", () => {
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
      const index = await client.channels.retrieve(
        updated.properties.digitalInput.index,
      );
      expect(index.name).toBe(`${identifier}_di_time`);
      expect(index.isIndex).toBe(true);
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
  });
});

describe("DigitalRead device map binding", () => {
  const createMappedDevice = async (line: string, key: number) =>
    await createNIDevice(client, {
      properties: {
        digitalInput: {
          portCount: 0,
          lineCounts: [],
          index: 0,
          channels: { [line]: key },
        },
      },
    });

  it("should show the channel the device map binds to a row that holds none", async () => {
    const bound = await createTestChannel(client, "di");
    const dev = await createMappedDevice("0l1", bound.key);
    await renderDigitalRead(createConfig([createChannel(0, 1)], dev.key));
    await screen.findByText(bound.name);
  });

  it("should save the bound channel into the task", async () => {
    const bound = await createTestChannel(client, "di");
    const dev = await createMappedDevice("0l1", bound.key);
    const { draft } = await renderDigitalRead(
      createConfig([createChannel(0, 1)], dev.key),
    );
    await screen.findByText(bound.name);
    await waitFor(async () => {
      const saved = await client.tasks.retrieve({
        key: draft.key,
        schemas: NI.Task.DIGITAL_READ_SCHEMAS,
      });
      expect(saved.config.channels[0].channel).toBe(bound.key);
    });
  });

  it("should leave a stale channel in the task when the device map has no entry", async () => {
    const stale = await createTestChannel(client, "di");
    const dev = await createNIDevice(client);
    const { draft } = await renderDigitalRead(
      createConfig([createChannel(0, 1, { channel: stale.key })], dev.key),
    );
    await screen.findByText("No channel");
    const saved = await client.tasks.retrieve({
      key: draft.key,
      schemas: NI.Task.DIGITAL_READ_SCHEMAS,
    });
    expect(saved.config.channels[0].channel).toBe(stale.key);
  });

  // A running task whose row was deleted and re-added: the row holds a fresh key and
  // no channel, exactly as the form autosaves it before this binding lands.
  const createRunningWithReAddedRow = async (bound: number, dev: string) => {
    const deployed = await createDraft(
      createConfig([createChannel(0, 1, { channel: bound })], dev),
    );
    await client.tasks.create(
      {
        ...deployed,
        config: createConfig([createChannel(0, 1)], dev),
        status: createTaskStatus({
          details: {
            task: deployed.key,
            running: true,
            configHash: deployed.configHash,
            rack: deployed.rack,
          },
        }),
      },
      NI.Task.DIGITAL_READ_SCHEMAS,
    );
    return deployed;
  };

  it("should hide redeploy once a re-added row is bound to its deployed channel", async () => {
    const bound = await createTestChannel(client, "di");
    const dev = await createMappedDevice("0l1", bound.key);
    const deployed = await createRunningWithReAddedRow(bound.key, dev.key);
    await renderNITaskForm(NI.Task.DigitalRead, { client, taskKey: deployed.key });
    await screen.findByText(bound.name);
    await waitFor(async () => {
      const saved = await client.tasks.retrieve({
        key: deployed.key,
        schemas: NI.Task.DIGITAL_READ_SCHEMAS,
      });
      expect(saved.configHash).toBe(deployed.configHash);
    });
    await waitFor(() => expect(isRedeployHidden()).toBe(true));
  });

  it("should keep redeploy shown when a re-added row cannot be bound", async () => {
    const bound = await createTestChannel(client, "di");
    const dev = await createNIDevice(client);
    const deployed = await createRunningWithReAddedRow(bound.key, dev.key);
    await renderNITaskForm(NI.Task.DigitalRead, { client, taskKey: deployed.key });
    await screen.findByText("No channel");
    await waitFor(() => expect(isRedeployHidden()).toBe(false));
  });

  it("should drop a row's stale channel when the device map has no entry for its line", async () => {
    const stale = await createTestChannel(client, "di");
    const dev = await createNIDevice(client);
    await renderDigitalRead(
      createConfig([createChannel(0, 1, { channel: stale.key })], dev.key),
    );
    await screen.findByText("No channel");
    expect(screen.queryByText(stale.name)).toBeNull();
  });

  it("should keep a row's channel while no device is selected", async () => {
    const own = await createTestChannel(client, "di");
    await renderDigitalRead(
      createConfig([createChannel(0, 1, { channel: own.key })], ""),
    );
    await screen.findByText(own.name);
  });

  it("should rebind a row when its line is edited to one the device map holds", async () => {
    const bound = await createTestChannel(client, "di");
    const dev = await createMappedDevice("3l1", bound.key);
    await renderDigitalRead(createConfig([createChannel(3, 0)], dev.key));
    await screen.findByText("No channel");
    commitFieldInput(screen.getByDisplayValue("0"), "1");
    await screen.findByText(bound.name);
  });
});
