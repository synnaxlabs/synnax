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
): Promise<task.Task<OPCUA.Task.EventSchemas>> =>
  await client.tasks.create(
    {
      name: uniqueName("events"),
      type: OPCUA.Task.EVENT_TYPE,
      config: { ...OPCUA.Task.EVENT_SCHEMAS.config.parse({}), ...config },
    },
    OPCUA.Task.EVENT_SCHEMAS,
  );

const retrieveConfig = async (key: task.Key): Promise<Config> =>
  (await client.tasks.retrieve({ key, schemas: OPCUA.Task.EVENT_SCHEMAS })).config;

const STANDARD_FIELD_HINT = /Each event records its ID/;

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

  it("should add a dropped object as a source, and remove it", async () => {
    const { scan, device } = await createScannedDevice(client);
    const draft = await createDraft({ device: device.key });
    const pump: OPCUA.Task.ScannedNode = {
      key: "ns=1;s=Pump",
      nodeId: "ns=1;s=Pump",
      name: "Pump",
      nodeClass: "Object",
      dataType: "",
      isArray: false,
      field: "",
    };
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
});
