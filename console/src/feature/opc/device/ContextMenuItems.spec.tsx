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
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { OPC } from "@/feature/opc";
import { createOPCDevice } from "@/feature/opc/testutil";
import { createDeviceResource, renderMenuItem } from "@/platform/device/testutil";
import {
  createBaseProps,
  createSelection,
  createState,
} from "@/platform/tree/testutil";
import { Session } from "@/session";
import { createTestStore, resolveFocusedTab, uniqueName } from "@/testutil";

const client = createTestClient();

const renderItems = async () => {
  const dev = await createOPCDevice(client);
  const propsStore = await createTestStore();
  const resource = createDeviceResource({
    key: dev.key,
    name: dev.name,
    configured: true,
  });
  const handle = await renderMenuItem(
    <OPC.Device.ContextMenuItems
      {...createBaseProps({ client, store: propsStore })}
      selection={createSelection({ ids: [resource.id] })}
      state={createState([resource])}
    />,
    { client },
  );
  const proj = await client.projects.create({
    name: uniqueName("proj"),
    layout: {},
  });
  handle.store.dispatch(Session.Project.select(proj.key));
  return { ...handle, dev };
};

describe("OPC.Device.ContextMenuItems", () => {
  it("should create a read task draft bound to the device", async () => {
    const { store, dev } = await renderItems();
    fireEvent.click(await screen.findByText("Create read task"));
    const tab = await resolveFocusedTab(store, client);
    if (tab.variant !== "resource") throw new Error("expected a resource tab");
    expect(tab.resource.type).toBe(task.TYPE_ONTOLOGY_ID.type);
    const created = await client.tasks.retrieve({ key: tab.resource.key });
    expect(created.type).toBe(OPC.Task.READ_TYPE);
    expect(created.config).toMatchObject({ device: dev.key });
  });

  it("should create a write task draft bound to the device", async () => {
    const { store, dev } = await renderItems();
    fireEvent.click(await screen.findByText("Create write task"));
    const tab = await resolveFocusedTab(store, client);
    if (tab.variant !== "resource") throw new Error("expected a resource tab");
    expect(tab.resource.type).toBe(task.TYPE_ONTOLOGY_ID.type);
    const created = await client.tasks.retrieve({ key: tab.resource.key });
    expect(created.type).toBe(OPC.Task.WRITE_TYPE);
    expect(created.config).toMatchObject({ device: dev.key });
  });

  it("should open the connect modal from the edit connection item", async () => {
    const { dev } = await renderItems();
    await screen.findByText("Create read task");
    fireEvent.click(screen.getByText("Edit connection"));
    await screen.findByText("Server");
    await waitFor(() => expect(screen.getByDisplayValue(dev.name)).toBeTruthy());
  });
});
