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
import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { MQTT } from "@/feature/mqtt";
import { createBroker } from "@/feature/mqtt/testutil";
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
  const dev = await createBroker(client);
  const propsStore = await createTestStore();
  const resource = createDeviceResource({
    key: dev.key,
    name: dev.name,
    configured: true,
  });
  const handle = await renderMenuItem(
    <MQTT.Device.ContextMenuItems
      {...createBaseProps({ client, store: propsStore })}
      selection={createSelection({ ids: [resource.id] })}
      state={createState([resource])}
      onConfigure={vi.fn()}
    />,
    { client },
  );
  const proj = await client.projects.create({ name: uniqueName("proj") });
  handle.store.dispatch(Session.Project.select(proj.key));
  return { ...handle, dev };
};

describe("MQTT.Device.ContextMenuItems", () => {
  it("should create read and write task drafts bound to the device", async () => {
    const { store, dev } = await renderItems();
    fireEvent.click(await screen.findByText("Create read task"));
    const readTab = await resolveFocusedTab(store, client);
    if (readTab.variant !== "resource") throw new Error("expected a resource tab");
    expect(readTab.resource.type).toBe(task.TYPE_ONTOLOGY_ID.type);
    const read = await client.tasks.retrieve({ key: readTab.resource.key });
    expect(read.type).toBe(MQTT.Task.READ_TYPE);
    expect(read.config).toMatchObject({ device: dev.key });
    fireEvent.click(screen.getByText("Create write task"));
    const writeTab = await resolveFocusedTab(
      store,
      client,
      (t) => t.variant === "resource" && t.resource.key !== readTab.resource.key,
    );
    if (writeTab.variant !== "resource") throw new Error("expected a resource tab");
    const write = await client.tasks.retrieve({ key: writeTab.resource.key });
    expect(write.type).toBe(MQTT.Task.WRITE_TYPE);
    expect(write.config).toMatchObject({ device: dev.key });
  });

  it("should create an edge node draft bound to the device", async () => {
    const { store, dev } = await renderItems();
    fireEvent.click(await screen.findByText("Create Sparkplug edge node"));
    const tab = await resolveFocusedTab(store, client);
    if (tab.variant !== "resource") throw new Error("expected a resource tab");
    const edge = await client.tasks.retrieve({ key: tab.resource.key });
    expect(edge.type).toBe(MQTT.Task.EDGE_TYPE);
    expect(edge.config).toMatchObject({ device: dev.key });
  });
});
