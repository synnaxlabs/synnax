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
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { type z } from "zod";

import { OPCUA } from "@/feature/opcua";
import { createOPCDevice, createScannedDevice } from "@/feature/opcua/testutil";
import {
  answerNextCommand,
  awaitEditableForm,
  deployAndAwaitTask,
  renderTaskFormTab,
  type RenderTaskFormTabOptions,
  reportTaskStopped,
} from "@/platform/task/testutil";
import { uniqueName } from "@/testutil";

const client = createTestClient();

// The form renders read-only until the update grant lands, and a preview field renders
// no input, so wait for it to become editable before querying fields.
const renderEvent = async (options: RenderTaskFormTabOptions = {}) => {
  const rendered = await renderTaskFormTab(OPCUA.Task.Event, options);
  await awaitEditableForm();
  return rendered;
};

type Config = z.infer<OPCUA.Task.EventSchemas["config"]>;

const createDraft = async (
  config: Partial<Config> & Pick<Config, "device">,
  name: string = uniqueName("events"),
): Promise<task.Task<OPCUA.Task.EventSchemas>> =>
  await client.tasks.create(
    {
      name,
      type: OPCUA.Task.EVENT_TYPE,
      config: { ...OPCUA.Task.EVENT_SCHEMAS.config.parse({}), ...config },
    },
    OPCUA.Task.EVENT_SCHEMAS,
  );

const retrieveConfig = async (key: task.Key): Promise<Config> =>
  (await client.tasks.retrieve({ key, schemas: OPCUA.Task.EVENT_SCHEMAS })).config;

const STANDARD_FIELD_HINT = /Each event records its ID/;

const createNode = (name: string, nodeClass: string): OPCUA.Task.ScannedNode => ({
  key: `ns=1;s=${name}`,
  nodeId: `ns=1;s=${name}`,
  name,
  nodeClass,
  dataType: "",
  isArray: false,
  field: "",
});

describe("Event", () => {
  it("should create a JSON channel with an index on deploy", async () => {
    const dev = await createOPCDevice(client);
    const draft = await createDraft({ device: dev.key });
    const { container } = await renderEvent({ client, taskKey: draft.key });
    await screen.findByText(STANDARD_FIELD_HINT);
    const deployed = await deployAndAwaitTask(
      client,
      container,
      draft.key,
      OPCUA.Task.EVENT_SCHEMAS,
    );
    expect(deployed.rack).toBe(dev.rack);
    const events = await client.channels.retrieve(deployed.config.channel);
    expect(events.name).toBe(`${draft.name}_events`);
    expect(events.dataType.toString()).toBe("json");
    const index = await client.channels.retrieve(events.index);
    expect(index.name).toBe(`${draft.name}_events_time`);
    expect(index.isIndex).toBe(true);
  });

  it("should keep the channel when redeploying", async () => {
    const dev = await createOPCDevice(client);
    const draft = await createDraft({ device: dev.key });
    const first = await renderEvent({ client, taskKey: draft.key });
    await screen.findByText(STANDARD_FIELD_HINT);
    const deployed = await deployAndAwaitTask(
      client,
      first.container,
      draft.key,
      OPCUA.Task.EVENT_SCHEMAS,
    );
    await reportTaskStopped(client, deployed.payload);
    first.unmount();
    const second = await renderEvent({ client, taskKey: draft.key });
    await screen.findByText(STANDARD_FIELD_HINT);
    const redeployed = await deployAndAwaitTask(
      client,
      second.container,
      draft.key,
      OPCUA.Task.EVENT_SCHEMAS,
    );
    expect(redeployed.config.channel).toBe(deployed.config.channel);
  });

  it("should add and remove a field", async () => {
    const dev = await createOPCDevice(client);
    const draft = await createDraft({ device: dev.key });
    await renderEvent({ client, taskKey: draft.key });
    fireEvent.click(await screen.findByRole("button", { name: "Add field" }));
    await waitFor(async () =>
      expect((await retrieveConfig(draft.key)).fields).toEqual([
        { key: expect.any(String), name: "", browsePath: "" },
      ]),
    );
    fireEvent.click(await screen.findByRole("button", { name: "Remove field" }));
    await waitFor(async () =>
      expect((await retrieveConfig(draft.key)).fields).toEqual([]),
    );
  });

  it("should give a second task with the same name channels with a free name", async () => {
    const dev = await createOPCDevice(client);
    const name = uniqueName("events");
    const first = await createDraft({ device: dev.key }, name);
    const rendered = await renderEvent({ client, taskKey: first.key });
    await screen.findByText(STANDARD_FIELD_HINT);
    await deployAndAwaitTask(
      client,
      rendered.container,
      first.key,
      OPCUA.Task.EVENT_SCHEMAS,
    );
    rendered.unmount();
    const second = await createDraft({ device: dev.key }, name);
    const { container } = await renderEvent({ client, taskKey: second.key });
    await screen.findByText(STANDARD_FIELD_HINT);
    const deployed = await deployAndAwaitTask(
      client,
      container,
      second.key,
      OPCUA.Task.EVENT_SCHEMAS,
    );
    const events = await client.channels.retrieve(deployed.config.channel);
    expect(events.name).toBe(`${name}_events_2`);
    const index = await client.channels.retrieve(events.index);
    expect(index.name).toBe(`${name}_events_2_time`);
  });

  it("should add a dropped object as a source, and remove it", async () => {
    const { scan, device } = await createScannedDevice(client);
    const draft = await createDraft({ device: device.key });
    const pump = createNode("Pump", "Object");
    const root = await answerNextCommand(client, scan, {
      type: OPCUA.Task.BROWSE_COMMAND_TYPE,
      data: { channels: [pump] },
    });
    await renderEvent({ client, taskKey: draft.key });
    await root.answered;
    fireEvent.dragStart(await screen.findByText("Pump"));
    fireEvent.drop(await screen.findByText(/Records the events of the Server object/));
    await waitFor(async () =>
      expect((await retrieveConfig(draft.key)).notifiers).toEqual([pump.nodeId]),
    );
    fireEvent.click(
      await screen.findByRole("button", { name: `Remove ${pump.nodeId}` }),
    );
    await waitFor(async () =>
      expect((await retrieveConfig(draft.key)).notifiers).toEqual([]),
    );
  });

  it("should add a dropped event type, and refuse a dropped object", async () => {
    const { scan, device } = await createScannedDevice(client);
    const draft = await createDraft({ device: device.key });
    const alarm = createNode("PumpAlarm", "ObjectType");
    const pump = createNode("Pump", "Object");
    const root = await answerNextCommand(client, scan, {
      type: OPCUA.Task.BROWSE_COMMAND_TYPE,
      data: { channels: [alarm, pump] },
    });
    await renderEvent({ client, taskKey: draft.key });
    await root.answered;
    const types = await screen.findByText(/Records events of every type/);
    fireEvent.dragStart(await screen.findByText("Pump"));
    fireEvent.drop(types);
    fireEvent.dragStart(await screen.findByText("PumpAlarm"));
    fireEvent.drop(types);
    await waitFor(async () =>
      expect((await retrieveConfig(draft.key)).eventTypes).toEqual([alarm.nodeId]),
    );
    expect((await retrieveConfig(draft.key)).notifiers).toEqual([]);
  });

  it("should list the EventTypes folder after the objects, and browse its types", async () => {
    const { scan, device } = await createScannedDevice(client);
    const draft = await createDraft({ device: device.key });
    const root = await answerNextCommand(client, scan, {
      type: OPCUA.Task.BROWSE_COMMAND_TYPE,
      data: { channels: [createNode("Pump", "Object")] },
    });
    await renderEvent({ client, taskKey: draft.key });
    await root.answered;
    const alarm = createNode("PumpAlarm", "ObjectType");
    const types = await answerNextCommand(client, scan, {
      type: OPCUA.Task.BROWSE_COMMAND_TYPE,
      data: { channels: [alarm] },
    });
    fireEvent.click(await screen.findByText("EventTypes"));
    expect((await types.answered).args).toMatchObject({ nodeId: "i=3048" });
    fireEvent.dragStart(await screen.findByText("PumpAlarm"));
    fireEvent.drop(await screen.findByText(/Records events of every type/));
    await waitFor(async () =>
      expect((await retrieveConfig(draft.key)).eventTypes).toEqual([alarm.nodeId]),
    );
  });
});
